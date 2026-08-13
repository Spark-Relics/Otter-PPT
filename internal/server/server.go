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

// ──────────────────────────────────────────────────────────────
// Server setup
// ──────────────────────────────────────────────────────────────

// Config holds server configuration.
type Config struct {
	Port         string
	APIKey       string
	BaseURL      string
	Model        string
	ImageAPIKey  string
	ImageBaseURL string
	ImageModel   string
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
	s := &Server{cfg: cfg, router: gin.New()}
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

// ──────────────────────────────────────────────────────────────
// Shared helpers (used by all generate handlers)
// ──────────────────────────────────────────────────────────────

// generateRequest is shared by all generation modes.
type generateRequest struct {
	Topic    string `json:"topic" binding:"required"`
	Language string `json:"language"`
	Slides   int    `json:"slides"`
	Style    string `json:"style"`
	Mode     string `json:"mode"` // "simple" | "workflow" (default: "workflow")
}

// makeAgentConfig builds an AgentConfig from server config + request params.
func (s *Server) makeAgentConfig(lang string) agent.AgentConfig {
	cfg := agent.AgentConfig{
		APIKey:   s.cfg.APIKey,
		BaseURL:  s.cfg.BaseURL,
		Model:    s.cfg.Model,
		Language: lang,
	}
	if s.cfg.ImageAPIKey != "" {
		cfg.ImageGenerator = ai.NewImageGenerator(ai.ImageConfig{
			APIKey: s.cfg.ImageAPIKey, BaseURL: s.cfg.ImageBaseURL, Model: s.cfg.ImageModel,
		})
	}
	return cfg
}

// buildAndStorePPTX builds a PPTX from the presentation, stores it for download,
// and returns the download URL. On failure it writes an HTTP error and returns "".
func (s *Server) buildAndStorePPTX(c *gin.Context, pres *model.Presentation) string {
	tmpFile, err := os.CreateTemp("", "otter-ppt-*.pptx")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temp file"})
		return ""
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	if err := builder.New(pres).Save(tmpPath); err != nil {
		os.Remove(tmpPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build PPTX: " + err.Error()})
		return ""
	}

	id := uuid.NewString()
	s.downloads.Store(id, tmpPath)
	return "/api/v1/download?id=" + id
}

// requireAPIKey writes a 401 error if no text-model API key is configured.
// Returns true if OK to proceed.
func (s *Server) requireAPIKey(c *gin.Context) bool {
	if s.cfg.APIKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "text model API key not configured; use /api/v1/execute or /api/v1/build for external AI mode",
		})
		return false
	}
	return true
}

// ──────────────────────────────────────────────────────────────
// POST /generate  (also serves /generate-v2 via alias)
// ──────────────────────────────────────────────────────────────

func (s *Server) handleGenerate(c *gin.Context) {
	var req generateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !s.requireAPIKey(c) {
		return
	}

	mode := req.Mode
	if mode == "" {
		mode = "workflow"
	}

	ag := agent.NewAgent(s.makeAgentConfig(req.Language))

	if mode == "workflow" {
		s.runWorkflowMode(c, ag, &req)
	} else {
		s.runSimpleMode(c, ag, &req)
	}
}

// runWorkflowMode executes the multi-phase pipeline (plan → gather → build → review → refine → polish).
func (s *Server) runWorkflowMode(c *gin.Context, ag *agent.Agent, req *generateRequest) {
	wfCfg := agent.DefaultWorkflowConfig()
	wfCfg.Topic = req.Topic
	wfCfg.SlideCount = req.Slides
	wfCfg.Style = req.Style
	wfCfg.Language = req.Language

	wfResult, err := agent.NewWorkflow(ag, wfCfg).Run(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	pres := ag.Session().Presentation()
	dlURL := s.buildAndStorePPTX(c, pres)
	if dlURL == "" {
		return
	}

	resp := gin.H{
		"presentation":  pres,
		"done":          wfResult.AgentResult.Done,
		"total_steps":   wfResult.AgentResult.TotalSteps,
		"refine_rounds": wfResult.RefineRounds,
		"mode":          "workflow",
		"download_url":  dlURL,
	}
	if wfResult.Plan != nil {
		resp["plan"] = wfResult.Plan
	}
	if wfResult.VisionReport != nil {
		resp["vision_score"] = wfResult.VisionReport.OverallScore
		resp["vision_report"] = wfResult.VisionReport
	}
	if wfResult.LayoutReport != nil {
		resp["layout_score"] = wfResult.LayoutReport.Score
	}
	c.JSON(http.StatusOK, resp)
}

// runSimpleMode executes the direct agent loop (no planning/vision).
func (s *Server) runSimpleMode(c *gin.Context, ag *agent.Agent, req *generateRequest) {
	prompt := agent.SplitPrompt(req.Topic, req.Slides, req.Style)
	result, err := ag.Run(c.Request.Context(), prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	pres := ag.Session().Presentation()
	dlURL := s.buildAndStorePPTX(c, pres)
	if dlURL == "" {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"presentation": pres,
		"steps":        result.Steps,
		"total_steps":  result.TotalSteps,
		"done":         result.Done,
		"mode":         "simple",
		"download_url": dlURL,
	})
}

// ──────────────────────────────────────────────────────────────
// POST /execute — apply pre-generated tool calls (no AI model)
// ──────────────────────────────────────────────────────────────

type externalToolCall struct {
	Name      string         `json:"name" binding:"required"`
	Arguments map[string]any `json:"arguments"`
}

type executeRequest struct {
	Presentation *model.Presentation `json:"presentation"`
	Calls        []externalToolCall  `json:"calls" binding:"required,min=1"`
}

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
				"error":              result.Message,
				"failed_call_index":  index,
				"results":            results,
				"presentation":       session.Presentation(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"presentation": session.Presentation(),
		"results":      results,
	})
}

// ──────────────────────────────────────────────────────────────
// POST /build — build PPTX from presentation JSON
// ──────────────────────────────────────────────────────────────

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

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.presentationml.presentation")
	c.Header("Content-Disposition", "attachment; filename=presentation.pptx")

	if err := builder.New(&pres).Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// ──────────────────────────────────────────────────────────────
// GET /tools — list available agent tools
// ──────────────────────────────────────────────────────────────

func (s *Server) handleListTools(c *gin.Context) {
	tools := pptoolkit.ToolDefinitions()
	c.JSON(http.StatusOK, gin.H{
		"tools":      tools,
		"tool_count": len(tools),
	})
}

// ──────────────────────────────────────────────────────────────
// Fonts
// ──────────────────────────────────────────────────────────────

func (s *Server) handleListFonts(c *gin.Context) {
	registry := fonts.GetRegistry()
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

func (s *Server) handleScanFonts(c *gin.Context) {
	registry := fonts.GetRegistry()
	entries, err := registry.Scan()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "scanned",
		"count":  len(entries),
		"fonts":  entries,
	})
}

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

	entries, _ := registry.Scan()
	var installed *fonts.FontEntry
	for i := range entries {
		if filepath.Base(entries[i].Path) == safeName {
			installed = &entries[i]
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "installed",
		"file":   safeName,
		"font":   installed,
		"total":  len(entries),
	})
}

// ──────────────────────────────────────────────────────────────
// GET /download — serve a previously generated PPTX (one-time)
// ──────────────────────────────────────────────────────────────

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
