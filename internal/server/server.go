// Package server provides the HTTP API for Otter PPT.
package server

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/otter-ppt/otter-ppt/internal/agent"
	"github.com/otter-ppt/otter-ppt/internal/builder"
	"github.com/otter-ppt/otter-ppt/internal/model"
	"github.com/otter-ppt/otter-ppt/internal/pptoolkit"
)

// Config holds server configuration.
type Config struct {
	Port    string
	APIKey  string
	BaseURL string
	Model   string
}

// Server is the HTTP server.
type Server struct {
	cfg    Config
	router *gin.Engine
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
		api.POST("/build", s.handleBuild)
		api.GET("/tools", s.handleListTools)
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "OPENAI_API_KEY not configured"})
		return
	}

	ag := agent.NewAgent(agent.AgentConfig{
		APIKey:   s.cfg.APIKey,
		BaseURL:  s.cfg.BaseURL,
		Model:    s.cfg.Model,
		Language: req.Language,
	})

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

	c.JSON(http.StatusOK, gin.H{
		"presentation": pres,
		"steps":        result.Steps,
		"total_steps":  result.TotalSteps,
		"done":         result.Done,
		"download_url": "/api/v1/download?file=" + tmpPath,
	})
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

// handleDownload serves a generated PPTX file.
func (s *Server) handleDownload(c *gin.Context) {
	filePath := c.Query("file")
	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file parameter required"})
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.presentationml.presentation")
	c.Header("Content-Disposition", "attachment; filename=presentation.pptx")
	c.File(filePath)
}
