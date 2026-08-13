// workflow.go implements the complete multi-phase PPT generation pipeline:
//
//   Phase 1: PLAN     — LLM creates outline + design strategy
//   Phase 2: GATHER   — Pre-generate all needed images in parallel
//   Phase 3: BUILD    — Agent tool-calling loop with rich plan context
//   Phase 4: REVIEW   — Render slides → send to vision model → get feedback
//   Phase 5: REFINE   — Feed vision feedback back to agent for fixes
//   Phase 6: POLISH   — Final auto-fix + layout validation
//
// This replaces the simple "prompt → tools → done" loop with a true
// AI-driven creative workflow that iterates until quality is achieved.
package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/otter-ppt/otter-ppt/internal/builder"
	"github.com/otter-ppt/otter-ppt/internal/layout"
	"github.com/otter-ppt/otter-ppt/internal/renderer"
)

// WorkflowConfig controls the multi-phase pipeline behavior.
type WorkflowConfig struct {
	Topic            string
	SlideCount       int
	Style            string
	Language         string
	MaxSteps         int           // max agent steps per build round (default 60)
	VisionThreshold  float64       // minimum vision score to accept (default 75)
	MaxRefineRounds  int           // max REVIEW→REFINE iterations (default 2)
	EnableVision     bool          // use vision model for visual review
	EnablePlanning   bool          // use pre-build planning phase
	GenerateImages   bool          // pre-generate images before building
	WorkDir          string        // temp working directory for intermediate files
}

// WorkflowResult captures the full pipeline output including all phases.
type WorkflowResult struct {
	Plan         *DesignPlan     `json:"plan,omitempty"`
	VisionReport *VisionReport   `json:"vision_report,omitempty"`
	LayoutReport *layout.PresentationReport `json:"layout_report,omitempty"`
	AgentResult  *AgentResult    `json:"agent_result"`
	RefineRounds int             `json:"refine_rounds"`
	PPTXPath     string          `json:"pptx_path"`
	piler        []string        // log messages for debugging
}

// Workflow runs the complete multi-phase PPT generation pipeline.
type Workflow struct {
	cfg       WorkflowConfig
	agent     *Agent
	planner   *Planner
	vision    *VisionEvaluator
	renderer  *renderer.Renderer
	log       func(format string, args ...any)
}

// NewWorkflow creates a workflow from agent config.
func NewWorkflow(a *Agent, cfg WorkflowConfig) *Workflow {
	if cfg.MaxSteps == 0 {
		cfg.MaxSteps = 60
	}
	if cfg.VisionThreshold == 0 {
		cfg.VisionThreshold = 75
	}
	if cfg.MaxRefineRounds == 0 {
		cfg.MaxRefineRounds = 2
	}
	if cfg.WorkDir == "" {
		cfg.WorkDir, _ = os.MkdirTemp("", "otter-workflow-*")
	}

	return &Workflow{
		cfg:      cfg,
		agent:    a,
		planner:  NewPlanner(a.client, a.cfg.Model),
		vision:   NewVisionEvaluator(a.client, a.cfg.Model),
		renderer: renderer.NewRenderer(),
		log: func(format string, args ...any) {
			log.Printf("[Workflow] "+format, args...)
		},
	}
}

// Run executes all phases sequentially and returns the final result.
func (w *Workflow) Run(ctx context.Context) (*WorkflowResult, error) {
	result := &WorkflowResult{AgentResult: &AgentResult{Steps: []StepLog{}}}

	// ──── Phase 1: PLAN ────
	var plan *DesignPlan
	if w.cfg.EnablePlanning {
		w.log("Phase 1: PLANNING — creating design outline")
		var err error
		plan, err = w.planner.Plan(ctx, w.cfg.Topic, w.cfg.SlideCount, w.cfg.Style, w.cfg.Language)
		if err != nil {
			w.log("Planning failed (continuing without plan): %v", err)
		} else {
			result.Plan = plan
			w.log("Plan created: %d slides, style: %s", len(plan.Slides), plan.StyleDirection)
		}
	}

	// ──── Phase 2: GATHER (pre-generate images) ────
	imageAssets := make(map[int]string) // slideNum → local image path
	if w.cfg.GenerateImages && plan != nil && w.agent.cfg.ImageGenerator != nil {
		w.log("Phase 2: GATHERING — pre-generating %d images", len(plan.ImageNeeds))
		imageAssets = w.gatherImages(ctx, plan.ImageNeeds)
	}

	// ──── Phase 3: BUILD ────
	w.log("Phase 3: BUILDING — agent creating slides")
	buildPrompt := w.buildEnhancedPrompt(plan, imageAssets)
	agentResult, err := w.agent.Run(ctx, buildPrompt)
	if err != nil {
		return result, fmt.Errorf("build phase failed: %w", err)
	}
	result.AgentResult = agentResult

	// ──── Phase 4-5: REVIEW → REFINE loop ────
	if w.cfg.EnableVision {
		for round := 0; round < w.cfg.MaxRefineRounds; round++ {
			w.log("Phase 4: REVIEW (round %d) — rendering and visual evaluation", round+1)

			pptxPath, visionReport, renderErr := w.reviewCurrentState(ctx)
			if renderErr != nil {
				w.log("Vision review failed: %v", renderErr)
				break
			}
			result.PPTXPath = pptxPath
			result.VisionReport = visionReport

			if !ShouldRefine(visionReport, w.cfg.VisionThreshold) {
				w.log("Vision score %.0f ≥ threshold %.0f — accepting", visionReport.OverallScore, w.cfg.VisionThreshold)
				break
			}

			w.log("Phase 5: REFINE (round %d) — score %.0f, feeding feedback to agent", round+1, visionReport.OverallScore)
			result.RefineRounds = round + 1

			// Feed vision feedback back to agent for targeted fixes
			feedback := FormatVisionReport(visionReport)
			refineResult, err := w.agent.Refine(ctx, feedback)
			if err != nil {
				w.log("Refine failed: %v", err)
				break
			}
			// Merge agent results
			result.AgentResult.Steps = append(result.AgentResult.Steps, refineResult.Steps...)
			result.AgentResult.TotalSteps = len(result.AgentResult.Steps)
		}
	} else {
		// No vision: still build PPTX for export
		pptxPath := fmt.Sprintf("%s/output.pptx", w.cfg.WorkDir)
		pres := w.agent.Session().Presentation()
		b := builder.New(pres)
		if err := b.Save(pptxPath); err != nil {
			return result, fmt.Errorf("final build failed: %w", err)
		}
		result.PPTXPath = pptxPath
	}

	// ──── Phase 6: POLISH — final auto-fix ────
	w.log("Phase 6: POLISH — final layout auto-fix")
	pres := w.agent.Session().Presentation()
	fixes := layout.AutoFixPresentation(pres)
	result.LayoutReport = layout.ValidatePresentation(pres)
	w.log("Polish: auto-fixed %d issues, final layout score: %.0f/100", fixes, result.LayoutReport.Score)

	return result, nil
}

// gatherImages pre-generates all images from the plan in parallel.
func (w *Workflow) gatherImages(ctx context.Context, needs []ImageNeed) map[int]string {
	assets := make(map[int]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, need := range needs {
		if need.Prompt == "" {
			continue
		}
		wg.Add(1)
		go func(n ImageNeed) {
			defer wg.Done()
			path, err := w.agent.cfg.ImageGenerator.Generate(ctx, n.Prompt)
			if err != nil {
				w.log("Image generation failed for slide %d: %v", n.SlideNum, err)
				return
			}
			mu.Lock()
			assets[n.SlideNum] = path
			mu.Unlock()
			w.log("Generated image for slide %d [%s]", n.SlideNum, n.Role)
		}(need)
	}
	wg.Wait()
	return assets
}

// buildEnhancedPrompt creates the user prompt for the build phase,
// incorporating the plan and pre-generated image paths.
func (w *Workflow) buildEnhancedPrompt(plan *DesignPlan, images map[int]string) string {
	if plan == nil {
		// Fallback: use basic SplitPrompt
		return SplitPrompt(w.cfg.Topic, w.cfg.SlideCount, w.cfg.Style)
	}

	parts := []string{
		fmt.Sprintf("Create a presentation about: %s", w.cfg.Topic),
		"\n\n" + FormatPlan(plan),
	}

	if len(images) > 0 {
		parts = append(parts, "\n\nPRE-GENERATED IMAGES (use these paths in add_image):")
		for slideNum, path := range images {
			parts = append(parts, fmt.Sprintf("  - Slide %d: %s", slideNum, path))
		}
	}

	return joinStrings(parts, "\n")
}

// reviewCurrentState builds a PPTX, renders it, and runs vision evaluation.
func (w *Workflow) reviewCurrentState(ctx context.Context) (string, *VisionReport, error) {
	pres := w.agent.Session().Presentation()
	if len(pres.Slides) == 0 {
		return "", nil, fmt.Errorf("no slides to review")
	}

	// Build PPTX
	pptxPath := fmt.Sprintf("%s/review.pptx", w.cfg.WorkDir)
	b := builder.New(pres)
	if err := b.Save(pptxPath); err != nil {
		return "", nil, fmt.Errorf("review build failed: %w", err)
	}

	// Render to images
	slides, err := w.renderer.RenderPresentation(pptxPath, pres)
	if err != nil {
		w.log("Render failed, using fallback: %v", err)
	}
	if slides == nil {
		slides, _ = w.renderer.RenderPresentation("", pres) // force structural fallback
	}

	// Vision evaluation
	report, err := w.vision.EvaluateSlides(ctx, slides, w.cfg.Topic, w.cfg.Style)
	if err != nil {
		return pptxPath, nil, fmt.Errorf("vision evaluation failed: %w", err)
	}

	w.log("Vision: overall=%.0f, %d slides evaluated", report.OverallScore, len(report.SlideFeedback))
	return pptxPath, report, nil
}

// joinStrings is a helper to avoid strings.Join import bloat.
func joinStrings(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
}

// DefaultWorkflowConfig returns sensible defaults for the workflow.
func DefaultWorkflowConfig() WorkflowConfig {
	return WorkflowConfig{
		MaxSteps:        60,
		VisionThreshold:  75,
		MaxRefineRounds:  2,
		EnableVision:    true,
		EnablePlanning:  true,
		GenerateImages:  true,
	}
}
