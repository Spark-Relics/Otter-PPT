// Package server provides the HTTP API for Otter PPT.
package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/otter-ppt/otter-ppt/internal/agent"
	"github.com/otter-ppt/otter-ppt/internal/ai"
	"github.com/otter-ppt/otter-ppt/internal/builder"
	"github.com/otter-ppt/otter-ppt/internal/fonts"
	"github.com/otter-ppt/otter-ppt/internal/model"
	"github.com/otter-ppt/otter-ppt/internal/pptoolkit"
)

// Config holds server configuration.
type Config struct {
	Port          string
	APIKey        string
	BaseURL       string
	Model         string
	ImageAPIKey   string
	ImageBaseURL  string
	ImageModel    string
}

// Server is the HTTP server.
type Server struct {
	cfg       Config
	router    *gin.Engine
	downloads sync.Map
}

// New creates a new server.
func New(cfg Config) *Server {
	gin.SetMode(gin.ReleaseMode)
	s := &Server{
		cfg:    cfg,
		router: gin.New(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := s.router.Group("/api/v1")
	{
		api.POST("/generate", s.handleGenerate)
		api.POST("/execute", s.handleExecute)
		api.POST("/build", s.handleBuild)
		api.GET("/tools", s.handleListTools)
		api.GET("/fonts", s.handleListFonts)
		api.POST("/fonts/scan", s.handleScanFonts)
		api.POST("/fonts/install", s.handleInstallFont)
		api.GET("/download", s.handleDownload)
	}
}

// Run starts the HTTP server.
func (s *Server) Run() error {
	return s.router.Run(":" + s.cfg.Port)
}

// generateRequest is the request body for POST /generate.
type generateRequest struct {
	Topic    string `json:"topic" binding:"required"`
	Language string `json:"language"`
	Slides   int    `json:"slides"`
	Style    string `json:"style"`
}

// handleGenerate runs the AI agent to create a PPT, returns the JSON + PPTX.
func (s *Server) handleGenerate(c *gin.Context) {
	var req generateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if s.cfg.APIKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "text model API key not configured; use /api/v1/execute or /api/v1/build for external AI mode"})
		return
	}

	agentConfig := agent.AgentConfig{
		APIKey:   s.cfg.APIKey,
		BaseURL:  s.cfg.BaseURL,
		Model:    s.cfg.Model,
		Language: req.Language,
	}
	if s.cfg.ImageAPIKey != "" {
		agentConfig.ImageGenerator = ai.NewImageGenerator(ai.ImageConfig{
			APIKey: s.cfg.ImageAPIKey, BaseURL: s.cfg.ImageBaseURL, Model: s.cfg.ImageModel,
		})
	}
	ag := agent.NewAgent(agentConfig)

	prompt := agent.SplitPrompt(req.Topic, req.Slides, req.Style)
	result, err := ag.Run(c.Request.Context(), prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	pres := ag.Session().Presentation()

	// Build PPTX to temp file
	tmpFile, err := os.CreateTemp("", "otter-ppt-*.pptx")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temp file"})
		return
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	b := builder.New(pres)
	if err := b.Save(tmpPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build PPTX: " + err.Error()})
		return
	}

	downloadID := uuid.NewString()
	s.downloads.Store(downloadID, tmpPath)

	c.JSON(http.StatusOK, gin.H{
		"presentation": pres,
		"steps":        result.Steps,
		"total_steps":  result.TotalSteps,
		"done":         result.Done,
		"download_url": "/api/v1/download?id=" + downloadID,
	})
}

type externalToolCall struct {
	Name      string         `json:"name" binding:"required"`
	Arguments map[string]any `json:"arguments"`
}

type executeRequest struct {
	Presentation *model.Presentation `json:"presentation"`
	Calls        []externalToolCall  `json:"calls" binding:"required,min=1"`
}

// handleExecute applies externally generated tool calls without invoking any AI model.
func (s *Server) handleExecute(c *gin.Context) {
	var req executeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session := pptoolkit.NewSessionFromPresentation(req.Presentation)
	results := make([]pptoolkit.ToolResult, 0, len(req.Calls))
	for index, call := range req.Calls {
		if call.Arguments == nil {
			call.Arguments = map[string]any{}
		}
		result := session.ExecuteTool(call.Name, call.Arguments)
		results = append(results, result)
		if !result.Success {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": result.Message, "failed_call_index": index, "results": results,
				"presentation": session.Presentation(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"presentation": session.Presentation(), "results": results})
}

// handleBuild builds a PPTX from a pre-made presentation JSON.
func (s *Server) handleBuild(c *gin.Context) {
	raw, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var pres model.Presentation
	if err := json.Unmarshal(raw, &pres); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid presentation JSON: " + err.Error()})
		return
	}

	b := builder.New(&pres)

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.presentationml.presentation")
	c.Header("Content-Disposition", "attachment; filename=presentation.pptx")

	if err := b.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
}

// handleListTools returns the available tool definitions.
func (s *Server) handleListTools(c *gin.Context) {
	tools := pptoolkit.ToolDefinitions()
	c.JSON(http.StatusOK, gin.H{
		"tools":      tools,
		"tool_count": len(tools),
	})
}

// handleListFonts returns all available fonts (installed in assets/fonts + built-in catalog).
func (s *Server) handleListFonts(c *gin.Context) {
	registry := fonts.GetRegistry()

	// Try to scan if directory exists
	installed, _ := registry.Scan()
	catalog := registry.Catalog()

	c.JSON(http.StatusOK, gin.H{
		"installed": installed,
		"catalog":   catalog,
		"count":     len(installed),
		"builtin":   fonts.BuiltInFonts,
		"fonts_dir": registry.FontsDir(),
	})
}

// handleScanFonts rescans the fonts directory for new files.
func (s *Server) handleScanFonts(c *gin.Context) {
	registry := fonts.GetRegistry()
	entries, err := registry.Scan()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "scanned",
		"count":   len(entries),
		"fonts":   entries,
	})
}

// handleInstallFont accepts a font file upload and saves it to the fonts directory.
func (s *Server) handleInstallFont(c *gin.Context) {
	file, err := c.FormFile("font")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file uploaded: " + err.Error()})
		return
	}

	registry := fonts.GetRegistry()
	fontsDir := registry.FontsDir()
	if err := os.MkdirAll(fontsDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create fonts dir: " + err.Error()})
		return
	}

	// Sanitize filename
	safeName := filepath.Base(file.Filename)
	ext := strings.ToLower(filepath.Ext(safeName))
	if ext != ".ttf" && ext != ".otf" && ext != ".ttc" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only .ttf, .otf, .ttc files are supported"})
		return
	}

	destPath := filepath.Join(fontsDir, safeName)
	if err := c.SaveUploadedFile(file, destPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save font: " + err.Error()})
		return
	}

	// Re-scan to include the new font
	entries, _ := registry.Scan()

	// Find the just-installed font
	var installed *fonts.FontEntry
	for i := range entries {
		if filepath.Base(entries[i].Path) == safeName {
			installed = &entries[i]
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "installed",
		"file":    safeName,
		"font":    installed,
		"total":   len(entries),
	})
}

// handleDownload serves only files created by this process.
func (s *Server) handleDownload(c *gin.Context) {
	id := c.Query("id")
	value, ok := s.downloads.LoadAndDelete(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "download not found or already consumed"})
		return
	}
	filePath, ok := value.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid download entry"})
		return
	}
	defer os.Remove(filePath)

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.presentationml.presentation")
	c.Header("Content-Disposition", "attachment; filename=presentation.pptx")
	c.File(filePath)
}
