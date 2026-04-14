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
	"github.com/user/clotho/internal/store"
)

// Engine orchestrates the execution of a pipeline graph.
type Engine struct {
	registry    *ExecutorRegistry
	eventBus    *EventBus
	executions  store.ExecutionStore
	stepResults store.StepResultStore
}

// NewEngine creates a new Engine with the given dependencies.
func NewEngine(
	registry *ExecutorRegistry,
	eventBus *EventBus,
	executions store.ExecutionStore,
	stepResults store.StepResultStore,
) *Engine {
	return &Engine{
		registry:    registry,
		eventBus:    eventBus,
		executions:  executions,
		stepResults: stepResults,
	}
}

// ExecuteWorkflow validates, sorts, and executes each node in the pipeline graph.
func (e *Engine) ExecuteWorkflow(ctx context.Context, execution domain.Execution, graph domain.PipelineGraph) error {
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

		// Create step result record
		inputJSON, _ := json.Marshal(inputs)
		stepResult := domain.StepResult{
			ID:          uuid.New(),
			ExecutionID: execution.ID,
			NodeID:      node.ID,
			Status:      domain.StatusRunning,
			InputData:   inputJSON,
		}
		stepResult, err := e.stepResults.Create(ctx, stepResult)
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

		// Execute with streaming
		start := time.Now()
		chunks, resultCh, errCh := executor.ExecuteStream(ctx, node, inputs)

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
			errStr := execErr.Error()
			if updateErr := e.stepResults.UpdateStatus(ctx, stepResult.ID, domain.StatusFailed, nil, &errStr, nil, nil, &durationMs); updateErr != nil {
				slog.Error("failed to update step status", "error", updateErr)
			}
			e.eventBus.Publish(execution.ID, Event{
				Type:        EventStepFailed,
				ExecutionID: execution.ID,
				NodeID:      node.ID,
				Error:       execErr.Error(),
				Timestamp:   time.Now(),
			})
			return e.failExecution(ctx, execution.ID, fmt.Sprintf("node %s execution failed: %s", node.ID, execErr))
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

		// Update step result as completed
		if updateErr := e.stepResults.UpdateStatus(ctx, stepResult.ID, domain.StatusCompleted, stepOut.Data, nil, stepOut.TokensUsed, stepOut.CostUSD, &durationMs); updateErr != nil {
			slog.Error("failed to update step result", "error", updateErr)
		}

		// Publish step completed with a named payload. Frontend reads each
		// field by name from event.data — do NOT put the raw step output
		// directly on Event.Data, or the frontend cannot tell it apart
		// from metadata. See also the mirror shape in SSE handler tests.
		completionPayload, err := json.Marshal(map[string]any{
			"output":      json.RawMessage(stepOut.Data),
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
	}

	// 4. Atomically mark execution completed with cost/tokens
	if err := e.executions.Complete(ctx, execution.ID, totalCost, totalTokens); err != nil {
		slog.Error("failed to complete execution", "error", err)
	}

	e.eventBus.Publish(execution.ID, Event{
		Type:        EventExecutionCompleted,
		ExecutionID: execution.ID,
		Timestamp:   time.Now(),
	})

	return nil
}

// RerunFromNode re-executes a pipeline starting from a specific node, using
// cached outputs from a prior execution for all nodes before fromNodeID.
func (e *Engine) RerunFromNode(ctx context.Context, execution domain.Execution, graph domain.PipelineGraph, fromNodeID string) error {
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

		inputJSON, _ := json.Marshal(inputs)
		stepResult := domain.StepResult{
			ID:          uuid.New(),
			ExecutionID: execution.ID,
			NodeID:      node.ID,
			Status:      domain.StatusRunning,
			InputData:   inputJSON,
		}
		stepResult, err := e.stepResults.Create(ctx, stepResult)
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

		start := time.Now()
		chunks, resultCh, errCh := executor.ExecuteStream(ctx, node, inputs)

		for chunk := range chunks {
			chunkData, _ := json.Marshal(chunk.Content)
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
			errStr := execErr.Error()
			if updateErr := e.stepResults.UpdateStatus(ctx, stepResult.ID, domain.StatusFailed, nil, &errStr, nil, nil, &durationMs); updateErr != nil {
				slog.Error("failed to update step status", "error", updateErr)
			}
			e.eventBus.Publish(execution.ID, Event{
				Type:        EventStepFailed,
				ExecutionID: execution.ID,
				NodeID:      node.ID,
				Error:       execErr.Error(),
				Timestamp:   time.Now(),
			})
			return e.failExecution(ctx, execution.ID, fmt.Sprintf("node %s execution failed: %s", node.ID, execErr))
		}

		if stepOut.TokensUsed != nil {
			totalTokens += *stepOut.TokensUsed
		}
		if stepOut.CostUSD != nil {
			totalCost += *stepOut.CostUSD
		}

		nodeOutputs[node.ID] = stepOut.Data

		if updateErr := e.stepResults.UpdateStatus(ctx, stepResult.ID, domain.StatusCompleted, stepOut.Data, nil, stepOut.TokensUsed, stepOut.CostUSD, &durationMs); updateErr != nil {
			slog.Error("failed to update step result", "error", updateErr)
		}

		e.eventBus.Publish(execution.ID, Event{
			Type:        EventStepCompleted,
			ExecutionID: execution.ID,
			NodeID:      node.ID,
			Data:        stepOut.Data,
			Timestamp:   time.Now(),
		})
	}

	if err := e.executions.Complete(ctx, execution.ID, totalCost, totalTokens); err != nil {
		slog.Error("failed to complete execution", "error", err)
	}

	e.eventBus.Publish(execution.ID, Event{
		Type:        EventExecutionCompleted,
		ExecutionID: execution.ID,
		Timestamp:   time.Now(),
	})

	return nil
}

// failExecution marks the execution as failed and publishes a failure event.
func (e *Engine) failExecution(ctx context.Context, executionID uuid.UUID, errMsg string) error {
	if updateErr := e.executions.UpdateStatus(ctx, executionID, domain.StatusFailed, &errMsg); updateErr != nil {
		slog.Error("failed to update execution status", "error", updateErr)
	}
	e.eventBus.Publish(executionID, Event{
		Type:        EventExecutionFailed,
		ExecutionID: executionID,
		Error:       errMsg,
		Timestamp:   time.Now(),
	})
	return fmt.Errorf("%s", errMsg)
}
