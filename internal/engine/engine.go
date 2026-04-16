package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/user/clotho/internal/domain"
	"github.com/user/clotho/internal/storage"
	"github.com/user/clotho/internal/store"
	"github.com/user/clotho/internal/util/redact"
)

// manifestOutputPreviewBytes caps inline text output captured in the manifest
// so the sidecar JSON stays human-readable when an agent emits a long body.
const manifestOutputPreviewBytes = 1024

// Engine orchestrates the execution of a pipeline graph.
type Engine struct {
	registry    *ExecutorRegistry
	eventBus    *EventBus
	executions  store.ExecutionStore
	stepResults store.StepResultStore
	fileStore   storage.Store
}

// NewEngine creates a new Engine with the given dependencies. fileStore may be
// nil; when nil, the engine skips manifest writes and per-node Location
// plumbing is a no-op for downstream executors.
func NewEngine(
	registry *ExecutorRegistry,
	eventBus *EventBus,
	executions store.ExecutionStore,
	stepResults store.StepResultStore,
	fileStore storage.Store,
) *Engine {
	return &Engine{
		registry:    registry,
		eventBus:    eventBus,
		executions:  executions,
		stepResults: stepResults,
		fileStore:   fileStore,
	}
}

// ExecuteWorkflow validates, sorts, and executes each node in the pipeline graph.
func (e *Engine) ExecuteWorkflow(ctx context.Context, execution domain.Execution, graph domain.PipelineGraph) error {
	startTime := time.Now()

	// Pull the top-level Location attached by the worker (project/pipeline
	// slugs + execution ID). If absent, the storage layer falls back to the
	// "unsorted" bucket — execution still succeeds.
	baseLoc, _ := storage.LocationFromContext(ctx)

	// 1. Validate graph
	if errs := ValidateGraph(graph); len(errs) > 0 {
		msgs := make([]string, 0, len(errs))
		for _, ve := range errs {
			msgs = append(msgs, ve.Error())
		}
		errMsg := fmt.Sprintf("graph validation failed: %s", strings.Join(msgs, "; "))
		errStr := errMsg
		if updateErr := e.executions.UpdateStatus(ctx, execution.ID, domain.StatusFailed, &errStr); updateErr != nil {
			slog.Error("failed to update execution status", "error", updateErr)
		}
		e.eventBus.Publish(execution.ID, Event{
			Type:        EventExecutionFailed,
			ExecutionID: execution.ID,
			Error:       errMsg,
			Timestamp:   time.Now(),
		})
		return fmt.Errorf("%s", errMsg)
	}

	// 2. Topological sort
	sortedNodes, err := TopoSort(graph)
	if err != nil {
		errStr := err.Error()
		if updateErr := e.executions.UpdateStatus(ctx, execution.ID, domain.StatusFailed, &errStr); updateErr != nil {
			slog.Error("failed to update execution status", "error", updateErr)
		}
		return fmt.Errorf("topo sort: %w", err)
	}

	// Mark execution as running
	if err := e.executions.UpdateStatus(ctx, execution.ID, domain.StatusRunning, nil); err != nil {
		return fmt.Errorf("update execution to running: %w", err)
	}

	// Inject tenant ID into context for executor credential lookups
	ctx = ContextWithTenantID(ctx, execution.TenantID)

	// Build edge lookup: targetNodeID -> targetPortID -> (sourceNodeID, sourcePortID)
	type edgeRef struct {
		SourceNodeID string
		SourcePortID string
	}
	edgeLookup := make(map[string]map[string]edgeRef)
	for _, edge := range graph.Edges {
		if edgeLookup[edge.Target] == nil {
			edgeLookup[edge.Target] = make(map[string]edgeRef)
		}
		edgeLookup[edge.Target][edge.TargetPort] = edgeRef{
			SourceNodeID: edge.Source,
			SourcePortID: edge.SourcePort,
		}
	}

	// Store step outputs by nodeID -> portID -> data
	nodeOutputs := make(map[string]json.RawMessage)

	var totalCost float64
	var totalTokens int

	// 3. Execute each node
	for _, node := range sortedNodes {
		// Check for cancellation between nodes
		select {
		case <-ctx.Done():
			return e.failExecution(ctx, execution.ID, "execution cancelled")
		default:
		}

		// Collect inputs from upstream
		inputs := make(map[string]json.RawMessage)
		if targets, ok := edgeLookup[node.ID]; ok {
			for targetPort, ref := range targets {
				if output, exists := nodeOutputs[ref.SourceNodeID]; exists {
					inputs[targetPort] = output
				}
			}
		}

		// Create step result record. Swallowing the marshal error here
		// would quietly corrupt the row (InputData becomes nil); fail the
		// whole execution instead so the bug surfaces immediately.
		inputJSON, err := json.Marshal(inputs)
		if err != nil {
			return e.failExecution(ctx, execution.ID, fmt.Sprintf("marshal inputs for node %s: %s", node.ID, err))
		}
		stepResult := domain.StepResult{
			ID:          uuid.New(),
			ExecutionID: execution.ID,
			NodeID:      node.ID,
			Status:      domain.StatusRunning,
			InputData:   inputJSON,
		}
		stepResult, err = e.stepResults.Create(ctx, stepResult)
		if err != nil {
			return e.failExecution(ctx, execution.ID, fmt.Sprintf("create step result for node %s: %s", node.ID, err))
		}

		// Publish step started
		e.eventBus.Publish(execution.ID, Event{
			Type:        EventStepStarted,
			ExecutionID: execution.ID,
			NodeID:      node.ID,
			Timestamp:   time.Now(),
		})

		// Get executor
		executor, err := e.registry.Get(node.Type)
		if err != nil {
			errStr := err.Error()
			if updateErr := e.stepResults.UpdateStatus(ctx, stepResult.ID, domain.StatusFailed, nil, &errStr, nil, nil, nil); updateErr != nil {
				slog.Error("failed to update step status", "error", updateErr)
			}
			return e.failExecution(ctx, execution.ID, fmt.Sprintf("no executor for node %s (type %s): %s", node.ID, node.Type, err))
		}

		// Attach a per-node Location so media providers can route output to
		// the correct execution folder under {dataDir}/{project}/{pipeline}/{exec}/.
		nodeLoc := baseLoc
		nodeLoc.NodeID = node.ID
		nodeCtx := storage.WithLocation(ctx, nodeLoc)

		// Apply per-step timeout. Defaults to 120s; agent nodes may
		// override via cfg.StepTimeoutSec (per Phase A reliability work).
		// cancelTimeout is called explicitly at end-of-iteration rather
		// than via defer so the loop doesn't accumulate timers across
		// nodes of a long pipeline.
		timeoutCtx, cancelTimeout := context.WithTimeout(nodeCtx, stepTimeoutFor(node))

		// Execute with streaming
		start := time.Now()
		chunks, resultCh, errCh := executor.ExecuteStream(timeoutCtx, node, inputs)

		// Forward streaming chunks as events. Payload must be a JSON object
		// with a named `chunk` field so the frontend can read it directly
		// via event.data.chunk — raw-marshaled strings would arrive as
		// undefined and accumulate as "undefinedundefined" in the store.
		for chunk := range chunks {
			chunkPayload, err := json.Marshal(map[string]string{
				"chunk": chunk.Content,
			})
			if err != nil {
				slog.Error("failed to marshal chunk payload", "error", err)
				continue
			}
			e.eventBus.Publish(execution.ID, Event{
				Type:        EventStepChunk,
				ExecutionID: execution.ID,
				NodeID:      node.ID,
				Data:        chunkPayload,
				Timestamp:   time.Now(),
			})
		}

		// Collect final result or error
		var stepOut StepOutput
		var execErr error
		select {
		case stepOut = <-resultCh:
		case execErr = <-errCh:
		}

		durationMs := time.Since(start).Milliseconds()

		if execErr != nil {
			cancelTimeout()
			errStr := redact.Secrets(execErr.Error())

			// Recover the structured StepFailure (when the executor
			// returned *FailureError) or classify the raw error so the
			// FailureDrawer always has a class + hint to render. The
			// classifier doesn't know which provider/model the executor
			// was using; for non-FailureError paths we accept the empty
			// strings — engine-internal failures (marshal, store) are
			// rare and a generic "internal" class is honest.
			failure := ClassifyExecutionError(execErr, "", "")
			failureBytes, _ := json.Marshal(failure)

			if updateErr := e.stepResults.UpdateStatus(ctx, stepResult.ID, domain.StatusFailed, nil, &errStr, nil, nil, &durationMs); updateErr != nil {
				slog.Error("failed to update step status", "error", updateErr)
			}
			if persistErr := e.stepResults.SetFailure(ctx, stepResult.ID, failureBytes); persistErr != nil {
				// Non-fatal: error column still has the 1-line summary.
				slog.Warn("failed to persist step failure_json", "error", persistErr)
			}

			// Build the SSE payload. Frontend reads event.data.failure
			// for the structured object and event.data.error for the
			// 1-line summary; both stay populated for back-compat.
			failedPayload, marshalErr := json.Marshal(map[string]any{
				"failure": json.RawMessage(failureBytes),
				"error":   errStr,
			})
			if marshalErr != nil {
				failedPayload = nil
			}
			e.eventBus.Publish(execution.ID, Event{
				Type:        EventStepFailed,
				ExecutionID: execution.ID,
				NodeID:      node.ID,
				Data:        failedPayload,
				Error:       errStr,
				Timestamp:   time.Now(),
			})

			// Persist execution-level failure_json so the executions list
			// and the FailureDrawer's diagnostic block can show class +
			// hint without re-fetching every step row.
			if persistErr := e.executions.SetFailure(ctx, execution.ID, failureBytes, &errStr); persistErr != nil {
				slog.Warn("failed to persist execution failure_json", "error", persistErr)
			}

			return e.failExecution(ctx, execution.ID, fmt.Sprintf("node %s execution failed: %s", node.ID, errStr))
		}

		// Accumulate cost and token data
		if stepOut.TokensUsed != nil {
			totalTokens += *stepOut.TokensUsed
		}
		if stepOut.CostUSD != nil {
			totalCost += *stepOut.CostUSD
		}

		// Save output
		nodeOutputs[node.ID] = stepOut.Data

		// Side-write agent text output to disk so the user can open the
		// generated script/prompt/etc. directly from their file manager.
		// Non-fatal: media nodes already land on disk via their providers;
		// for tool outputs we skip (they're usually pre-authored values).
		var agentFileRel string
		if node.Type == domain.NodeTypeAgent {
			agentFileRel = writeAgentOutputFile(ctx, e.fileStore, nodeLoc, node, stepOut.Data)
		}

		// Update step result as completed
		if updateErr := e.stepResults.UpdateStatus(ctx, stepResult.ID, domain.StatusCompleted, stepOut.Data, nil, stepOut.TokensUsed, stepOut.CostUSD, &durationMs); updateErr != nil {
			slog.Error("failed to update step result", "error", updateErr)
		}

		// Build an output_file URL the frontend can hand to Reveal-in-
		// Finder. Media outputs already ship as clotho://file/... URLs;
		// agents get one minted from the side-written .txt; tools leave
		// it empty (no on-disk artifact).
		outputFileURL := deriveOutputFileURL(node.Type, stepOut.Data, agentFileRel)

		// Publish step completed with a named payload. Frontend reads each
		// field by name from event.data — do NOT put the raw step output
		// directly on Event.Data, or the frontend cannot tell it apart
		// from metadata. See also the mirror shape in SSE handler tests.
		completionPayload, err := json.Marshal(map[string]any{
			"output":      json.RawMessage(stepOut.Data),
			"output_file": outputFileURL, // "" when the node has no on-disk artifact
			"tokens_used": stepOut.TokensUsed,
			"cost_usd":    stepOut.CostUSD,
			"duration_ms": durationMs,
		})
		if err != nil {
			slog.Error("failed to marshal step_completed payload", "error", err)
			completionPayload = []byte("{}")
		}
		e.eventBus.Publish(execution.ID, Event{
			Type:        EventStepCompleted,
			ExecutionID: execution.ID,
			NodeID:      node.ID,
			Data:        completionPayload,
			Timestamp:   time.Now(),
		})
		cancelTimeout()
	}

	// 4. Atomically mark execution completed with cost/tokens
	if err := e.executions.Complete(ctx, execution.ID, totalCost, totalTokens); err != nil {
		slog.Error("failed to complete execution", "error", err)
	}

	// 5. Best-effort manifest sidecar. Failure here does not fail the run —
	// the DB still has every step result and the user can re-export later.
	e.writeManifestBestEffort(ctx, execution, graph, baseLoc, sortedNodes, startTime, totalCost, totalTokens)

	// Surface the artifact directory so the frontend can offer an
	// "Open folder" action that reveals all outputs (agent .txt files,
	// media assets, manifest.json) in one place.
	completedPayload, err := json.Marshal(map[string]any{
		"total_cost":   totalCost,
		"total_tokens": totalTokens,
		"artifact_dir": storage.RelDir(baseLoc),
	})
	if err != nil {
		// Fall back to the legacy shape — the frontend treats a missing
		// data payload as a no-op and stays functional.
		completedPayload = nil
	}
	e.eventBus.Publish(execution.ID, Event{
		Type:        EventExecutionCompleted,
		ExecutionID: execution.ID,
		Data:        completedPayload,
		Timestamp:   time.Now(),
	})

	return nil
}

// RerunFromNode re-executes a pipeline starting from a specific node, using
// cached outputs from a prior execution for all nodes before fromNodeID.
func (e *Engine) RerunFromNode(ctx context.Context, execution domain.Execution, graph domain.PipelineGraph, fromNodeID string) error {
	startTime := time.Now()
	baseLoc, _ := storage.LocationFromContext(ctx)

	// 1. Validate graph
	if errs := ValidateGraph(graph); len(errs) > 0 {
		msgs := make([]string, 0, len(errs))
		for _, ve := range errs {
			msgs = append(msgs, ve.Error())
		}
		errMsg := fmt.Sprintf("graph validation failed: %s", strings.Join(msgs, "; "))
		errStr := errMsg
		if updateErr := e.executions.UpdateStatus(ctx, execution.ID, domain.StatusFailed, &errStr); updateErr != nil {
			slog.Error("failed to update execution status", "error", updateErr)
		}
		e.eventBus.Publish(execution.ID, Event{
			Type:        EventExecutionFailed,
			ExecutionID: execution.ID,
			Error:       errMsg,
			Timestamp:   time.Now(),
		})
		return fmt.Errorf("%s", errMsg)
	}

	// 2. Topological sort
	sortedNodes, err := TopoSort(graph)
	if err != nil {
		errStr := err.Error()
		if updateErr := e.executions.UpdateStatus(ctx, execution.ID, domain.StatusFailed, &errStr); updateErr != nil {
			slog.Error("failed to update execution status", "error", updateErr)
		}
		return fmt.Errorf("topo sort: %w", err)
	}

	// 3. Load prior step results and build cache
	priorResults, err := e.stepResults.ListByExecution(ctx, execution.ID)
	if err != nil {
		return e.failExecution(ctx, execution.ID, fmt.Sprintf("failed to load prior step results: %s", err))
	}
	cachedOutputs := make(map[string]json.RawMessage, len(priorResults))
	for _, sr := range priorResults {
		if sr.Status == domain.StatusCompleted && sr.OutputData != nil {
			cachedOutputs[sr.NodeID] = sr.OutputData
		}
	}

	// Validate that fromNodeID exists in the sorted nodes
	found := false
	for _, node := range sortedNodes {
		if node.ID == fromNodeID {
			found = true
			break
		}
	}
	if !found {
		return e.failExecution(ctx, execution.ID, fmt.Sprintf("from_node_id %q not found in pipeline graph", fromNodeID))
	}

	// Mark execution as running
	if err := e.executions.UpdateStatus(ctx, execution.ID, domain.StatusRunning, nil); err != nil {
		return fmt.Errorf("update execution to running: %w", err)
	}

	ctx = ContextWithTenantID(ctx, execution.TenantID)

	// Build edge lookup
	type edgeRef struct {
		SourceNodeID string
		SourcePortID string
	}
	edgeLookup := make(map[string]map[string]edgeRef)
	for _, edge := range graph.Edges {
		if edgeLookup[edge.Target] == nil {
			edgeLookup[edge.Target] = make(map[string]edgeRef)
		}
		edgeLookup[edge.Target][edge.TargetPort] = edgeRef{
			SourceNodeID: edge.Source,
			SourcePortID: edge.SourcePort,
		}
	}

	nodeOutputs := make(map[string]json.RawMessage)
	var totalCost float64
	var totalTokens int

	reachedFromNode := false

	for _, node := range sortedNodes {
		select {
		case <-ctx.Done():
			return e.failExecution(ctx, execution.ID, "execution cancelled")
		default:
		}

		if node.ID == fromNodeID {
			reachedFromNode = true
		}

		if !reachedFromNode {
			// Use cached output for nodes before fromNodeID
			cached, ok := cachedOutputs[node.ID]
			if !ok {
				return e.failExecution(ctx, execution.ID, fmt.Sprintf("no cached output for node %s (required for re-run from %s)", node.ID, fromNodeID))
			}
			nodeOutputs[node.ID] = cached

			e.eventBus.Publish(execution.ID, Event{
				Type:        EventStepCompleted,
				ExecutionID: execution.ID,
				NodeID:      node.ID,
				Data:        cached,
				Timestamp:   time.Now(),
			})
			continue
		}

		// Execute normally for fromNodeID and all subsequent nodes
		inputs := make(map[string]json.RawMessage)
		if targets, ok := edgeLookup[node.ID]; ok {
			for targetPort, ref := range targets {
				if output, exists := nodeOutputs[ref.SourceNodeID]; exists {
					inputs[targetPort] = output
				}
			}
		}

		inputJSON, err := json.Marshal(inputs)
		if err != nil {
			return e.failExecution(ctx, execution.ID, fmt.Sprintf("marshal inputs for node %s: %s", node.ID, err))
		}
		stepResult := domain.StepResult{
			ID:          uuid.New(),
			ExecutionID: execution.ID,
			NodeID:      node.ID,
			Status:      domain.StatusRunning,
			InputData:   inputJSON,
		}
		stepResult, err = e.stepResults.Create(ctx, stepResult)
		if err != nil {
			return e.failExecution(ctx, execution.ID, fmt.Sprintf("create step result for node %s: %s", node.ID, err))
		}

		e.eventBus.Publish(execution.ID, Event{
			Type:        EventStepStarted,
			ExecutionID: execution.ID,
			NodeID:      node.ID,
			Timestamp:   time.Now(),
		})

		executor, err := e.registry.Get(node.Type)
		if err != nil {
			errStr := err.Error()
			if updateErr := e.stepResults.UpdateStatus(ctx, stepResult.ID, domain.StatusFailed, nil, &errStr, nil, nil, nil); updateErr != nil {
				slog.Error("failed to update step status", "error", updateErr)
			}
			return e.failExecution(ctx, execution.ID, fmt.Sprintf("no executor for node %s (type %s): %s", node.ID, node.Type, err))
		}

		nodeLoc := baseLoc
		nodeLoc.NodeID = node.ID
		nodeCtx := storage.WithLocation(ctx, nodeLoc)

		start := time.Now()
		chunks, resultCh, errCh := executor.ExecuteStream(nodeCtx, node, inputs)

		for chunk := range chunks {
			// Marshal failure here is extraordinary — chunk.Content is a
			// string. Log and skip the chunk so a pathological byte
			// sequence doesn't kill the whole execution.
			chunkData, mErr := json.Marshal(chunk.Content)
			if mErr != nil {
				slog.Warn("drop chunk: marshal failed", "node_id", node.ID, "error", mErr)
				continue
			}
			e.eventBus.Publish(execution.ID, Event{
				Type:        EventStepChunk,
				ExecutionID: execution.ID,
				NodeID:      node.ID,
				Data:        json.RawMessage(chunkData),
				Timestamp:   time.Now(),
			})
		}

		var stepOut StepOutput
		var execErr error
		select {
		case stepOut = <-resultCh:
		case execErr = <-errCh:
		}

		durationMs := time.Since(start).Milliseconds()

		if execErr != nil {
			errStr := redact.Secrets(execErr.Error())
			if updateErr := e.stepResults.UpdateStatus(ctx, stepResult.ID, domain.StatusFailed, nil, &errStr, nil, nil, &durationMs); updateErr != nil {
				slog.Error("failed to update step status", "error", updateErr)
			}
			e.eventBus.Publish(execution.ID, Event{
				Type:        EventStepFailed,
				ExecutionID: execution.ID,
				NodeID:      node.ID,
				Error:       errStr,
				Timestamp:   time.Now(),
			})
			return e.failExecution(ctx, execution.ID, fmt.Sprintf("node %s execution failed: %s", node.ID, errStr))
		}

		if stepOut.TokensUsed != nil {
			totalTokens += *stepOut.TokensUsed
		}
		if stepOut.CostUSD != nil {
			totalCost += *stepOut.CostUSD
		}

		nodeOutputs[node.ID] = stepOut.Data

		// Same side-write as ExecuteWorkflow — keep re-runs writing
		// agent outputs so a "re-run from here" updates the .txt file.
		var agentFileRel string
		if node.Type == domain.NodeTypeAgent {
			agentFileRel = writeAgentOutputFile(ctx, e.fileStore, nodeLoc, node, stepOut.Data)
		}

		if updateErr := e.stepResults.UpdateStatus(ctx, stepResult.ID, domain.StatusCompleted, stepOut.Data, nil, stepOut.TokensUsed, stepOut.CostUSD, &durationMs); updateErr != nil {
			slog.Error("failed to update step result", "error", updateErr)
		}

		outputFileURL := deriveOutputFileURL(node.Type, stepOut.Data, agentFileRel)

		// Use the same named-payload shape as ExecuteWorkflow so the
		// frontend's SSE listener gets a consistent contract regardless
		// of which engine path produced the event.
		rerunStepPayload, err := json.Marshal(map[string]any{
			"output":      json.RawMessage(stepOut.Data),
			"output_file": outputFileURL,
			"tokens_used": stepOut.TokensUsed,
			"cost_usd":    stepOut.CostUSD,
			"duration_ms": durationMs,
		})
		if err != nil {
			rerunStepPayload = []byte("{}")
		}
		e.eventBus.Publish(execution.ID, Event{
			Type:        EventStepCompleted,
			ExecutionID: execution.ID,
			NodeID:      node.ID,
			Data:        rerunStepPayload,
			Timestamp:   time.Now(),
		})
	}

	if err := e.executions.Complete(ctx, execution.ID, totalCost, totalTokens); err != nil {
		slog.Error("failed to complete execution", "error", err)
	}

	e.writeManifestBestEffort(ctx, execution, graph, baseLoc, sortedNodes, startTime, totalCost, totalTokens)

	rerunPayload, err := json.Marshal(map[string]any{
		"total_cost":   totalCost,
		"total_tokens": totalTokens,
		"artifact_dir": storage.RelDir(baseLoc),
	})
	if err != nil {
		rerunPayload = nil
	}
	e.eventBus.Publish(execution.ID, Event{
		Type:        EventExecutionCompleted,
		ExecutionID: execution.ID,
		Data:        rerunPayload,
		Timestamp:   time.Now(),
	})

	return nil
}

// writeManifestBestEffort builds a Manifest from the execution's persisted
// step results and writes it to {baseLoc}/manifest.json. Failures are logged
// at warn level and swallowed — the user-visible execution stays "completed"
// because the database has the authoritative record.
func (e *Engine) writeManifestBestEffort(
	ctx context.Context,
	execution domain.Execution,
	graph domain.PipelineGraph,
	baseLoc storage.Location,
	sortedNodes []domain.NodeInstance,
	startTime time.Time,
	totalCost float64,
	totalTokens int,
) {
	if e.fileStore == nil {
		return
	}

	stepResults, err := e.stepResults.ListByExecution(ctx, execution.ID)
	if err != nil {
		slog.Warn("manifest: list step results failed", "execution_id", execution.ID, "error", err)
		return
	}

	// Index nodes by ID so we can pull names + media config for the manifest.
	nodeByID := make(map[string]domain.NodeInstance, len(sortedNodes))
	for _, n := range sortedNodes {
		nodeByID[n.ID] = n
	}

	manifestNodes := make([]ManifestNode, 0, len(stepResults))
	for _, sr := range stepResults {
		node, hasNode := nodeByID[sr.NodeID]

		mn := ManifestNode{
			NodeID: sr.NodeID,
			Status: string(sr.Status),
		}
		if hasNode {
			mn.NodeName = node.Label
			mn.Type = string(node.Type)
			if node.Type == domain.NodeTypeMedia {
				var mcfg domain.MediaNodeConfig
				if err := json.Unmarshal(node.Config, &mcfg); err == nil {
					mn.Provider = mcfg.Provider
					mn.Model = mcfg.Model
					mn.Prompt = mcfg.Prompt
				}
			}
		}

		// Decode the persisted output to decide whether it is an on-disk
		// reference or inline text.
		if len(sr.OutputData) > 0 {
			var s string
			if err := json.Unmarshal(sr.OutputData, &s); err == nil {
				if rel, ok := strings.CutPrefix(s, "clotho://file/"); ok {
					// Manifest stores just the basename — the file lives in
					// the same dir as manifest.json.
					mn.OutputFile = lastSegment(rel)
				} else if s != "" {
					mn.Output = truncateForManifest(s)
				}
			}
		}

		if sr.DurationMs != nil {
			mn.DurationMs = *sr.DurationMs
		}
		if sr.CostUSD != nil {
			mn.CostUSD = *sr.CostUSD
		}
		if sr.TokensUsed != nil {
			mn.TokensUsed = *sr.TokensUsed
		}
		if sr.Error != nil {
			mn.Error = *sr.Error
		}

		manifestNodes = append(manifestNodes, mn)
	}

	m := Manifest{
		ExecutionID:  execution.ID,
		PipelineID:   baseLoc.PipelineID,
		PipelineName: baseLoc.PipelineSlug,
		ProjectID:    baseLoc.ProjectID,
		StartedAt:    startTime,
		CompletedAt:  time.Now(),
		TotalCostUSD: totalCost,
		TotalTokens:  totalTokens,
		Nodes:        manifestNodes,
	}

	if _, err := WriteManifest(ctx, e.fileStore, baseLoc, m); err != nil {
		slog.Warn("manifest: write failed", "execution_id", execution.ID, "error", err)
	}
}

// lastSegment returns the path's final slash-separated component (basename).
func lastSegment(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// truncateForManifest caps a string to manifestOutputPreviewBytes runes,
// appending an ellipsis marker when truncated. Operates on bytes (not runes)
// because we only need a sanity cap on JSON size, not perfect text segmentation.
func truncateForManifest(s string) string {
	if len(s) <= manifestOutputPreviewBytes {
		return s
	}
	return s[:manifestOutputPreviewBytes] + "...[truncated]"
}

// failExecution marks the execution as failed and publishes a failure event.
// The incoming errMsg may contain an upstream provider body with an embedded
// API key (we've seen it with Gemini 403s). Scrub here so both the DB row
// and the SSE event carry a safe-to-display message.
func (e *Engine) failExecution(ctx context.Context, executionID uuid.UUID, errMsg string) error {
	safeMsg := redact.Secrets(errMsg)
	if updateErr := e.executions.UpdateStatus(ctx, executionID, domain.StatusFailed, &safeMsg); updateErr != nil {
		slog.Error("failed to update execution status", "error", updateErr)
	}
	e.eventBus.Publish(executionID, Event{
		Type:        EventExecutionFailed,
		ExecutionID: executionID,
		Error:       safeMsg,
		Timestamp:   time.Now(),
	})
	return fmt.Errorf("%s", safeMsg)
}
