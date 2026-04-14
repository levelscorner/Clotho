package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/engine"
	"github.com/user/clotho/internal/storage"
	"github.com/user/clotho/internal/store"
)

const (
	pollInterval      = 500 * time.Millisecond
	heartbeatInterval = 10 * time.Second
)

// Worker polls the job queue, executes pipeline workflows, and manages heartbeats.
type Worker struct {
	jobs             store.JobStore
	executions       store.ExecutionStore
	pipelineVersions store.PipelineVersionStore
	pipelines        store.PipelineStore
	projects         store.ProjectStore
	engine           *engine.Engine
}

// NewWorker creates a Worker with all required dependencies. pipelines and
// projects may be nil; when nil, executions are routed to the storage layer's
// "unsorted" bucket instead of the {project}/{pipeline}/{exec} path.
func NewWorker(
	jobs store.JobStore,
	executions store.ExecutionStore,
	pipelineVersions store.PipelineVersionStore,
	pipelines store.PipelineStore,
	projects store.ProjectStore,
	eng *engine.Engine,
) *Worker {
	return &Worker{
		jobs:             jobs,
		executions:       executions,
		pipelineVersions: pipelineVersions,
		pipelines:        pipelines,
		projects:         projects,
		engine:           eng,
	}
}

// Run starts the worker loop. It polls for jobs every 500ms and processes them.
// Blocks until the context is cancelled.
func (w *Worker) Run(ctx context.Context) {
	slog.Info("worker started")
	defer slog.Info("worker stopped")

	// Start zombie reaper in background
	go w.reapLoop(ctx)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processNext(ctx)
		}
	}
}

func (w *Worker) processNext(ctx context.Context) {
	job, err := w.jobs.Dequeue(ctx)
	if err != nil {
		slog.Error("failed to dequeue job", "error", err)
		return
	}
	if job == nil {
		return // no pending jobs
	}

	slog.Info("processing job", "job_id", job.ID, "execution_id", job.ExecutionID)

	// Start heartbeat goroutine
	heartbeatCtx, heartbeatCancel := context.WithCancel(ctx)
	defer heartbeatCancel()
	go w.heartbeatLoop(heartbeatCtx, job.ID)

	// Load execution
	execution, err := w.executions.GetByID(ctx, job.ExecutionID)
	if err != nil {
		slog.Error("failed to load execution", "error", err, "execution_id", job.ExecutionID)
		if failErr := w.jobs.Fail(ctx, job.ID, err.Error()); failErr != nil {
			slog.Error("failed to mark job as failed", "error", failErr)
		}
		return
	}

	// Load pipeline version
	pv, err := w.pipelineVersions.Get(ctx, execution.PipelineVersionID)
	if err != nil {
		slog.Error("failed to load pipeline version", "error", err, "version_id", execution.PipelineVersionID)
		if failErr := w.jobs.Fail(ctx, job.ID, err.Error()); failErr != nil {
			slog.Error("failed to mark job as failed", "error", failErr)
		}
		return
	}

	// Check for from_node_id in job payload
	var fromNodeID string
	if len(job.Payload) > 0 {
		var payload struct {
			FromNodeID string `json:"from_node_id"`
		}
		if err := json.Unmarshal(job.Payload, &payload); err == nil {
			fromNodeID = payload.FromNodeID
		}
	}

	// Build a storage Location so the engine and media providers can route
	// generated files into {dataDir}/{project}/{pipeline}/{exec}/.
	// Lookup failures are non-fatal — execution proceeds with a partial
	// Location, and the storage layer falls back to "unsorted/".
	loc := storage.Location{
		PipelineID:  pv.PipelineID,
		ExecutionID: execution.ID,
	}
	if w.pipelines != nil {
		pipeline, pipelineErr := w.pipelines.Get(ctx, pv.PipelineID)
		if pipelineErr != nil {
			slog.Warn("worker: load pipeline for storage location failed; routing to unsorted", "pipeline_id", pv.PipelineID, "error", pipelineErr)
		} else {
			loc.PipelineSlug = storage.Slugify(pipeline.Name)
			loc.ProjectID = pipeline.ProjectID

			if w.projects != nil {
				project, projectErr := w.projects.Get(ctx, pipeline.ProjectID)
				if projectErr != nil {
					slog.Warn("worker: load project for storage location failed; routing to unsorted", "project_id", pipeline.ProjectID, "error", projectErr)
				} else {
					loc.ProjectSlug = storage.Slugify(project.Name)
				}
			}
		}
	}
	execCtx := storage.WithLocation(ctx, loc)

	// Execute the workflow (full or partial re-run)
	var execErr error
	if fromNodeID != "" {
		slog.Info("re-running from node", "from_node_id", fromNodeID, "execution_id", execution.ID)
		execErr = w.engine.RerunFromNode(execCtx, execution, pv.Graph, fromNodeID)
	} else {
		execErr = w.engine.ExecuteWorkflow(execCtx, execution, pv.Graph)
	}
	if execErr != nil {
		slog.Error("workflow execution failed", "error", execErr, "execution_id", execution.ID)
		if failErr := w.jobs.Fail(ctx, job.ID, execErr.Error()); failErr != nil {
			slog.Error("failed to mark job as failed", "error", failErr)
		}
		return
	}

	// Mark job completed
	if err := w.jobs.Complete(ctx, job.ID); err != nil {
		slog.Error("failed to mark job as completed", "error", err)
	}

	slog.Info("job completed", "job_id", job.ID, "execution_id", job.ExecutionID)
}

func (w *Worker) heartbeatLoop(ctx context.Context, jobID uuid.UUID) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.jobs.Heartbeat(ctx, jobID); err != nil {
				slog.Warn("heartbeat failed", "job_id", jobID, "error", err)
			}
		}
	}
}
