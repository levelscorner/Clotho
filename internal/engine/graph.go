package engine

import (
	"fmt"

	"github.com/user/clotho/internal/domain"
)

// ValidationError describes a single graph validation failure.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateGraph checks graph integrity: node/port existence, port compatibility,
// required ports connected, and absence of cycles.
func ValidateGraph(graph domain.PipelineGraph) []ValidationError {
	var errs []ValidationError

	nodeMap := make(map[string]domain.NodeInstance, len(graph.Nodes))
	for _, n := range graph.Nodes {
		if _, exists := nodeMap[n.ID]; exists {
			errs = append(errs, ValidationError{
				Field:   "nodes",
				Message: fmt.Sprintf("duplicate node ID: %s", n.ID),
			})
		}
		nodeMap[n.ID] = n
	}

	// Build port lookup: nodeID -> portID -> Port
	portMap := make(map[string]map[string]domain.Port)
	for _, n := range graph.Nodes {
		ports := make(map[string]domain.Port, len(n.Ports))
		for _, p := range n.Ports {
			ports[p.ID] = p
		}
		portMap[n.ID] = ports
	}

	// Track which input ports are connected (for required-port checking)
	connectedInputs := make(map[string]map[string]bool)
	for _, n := range graph.Nodes {
		connectedInputs[n.ID] = make(map[string]bool)
	}

	// Validate edges
	for _, edge := range graph.Edges {
		srcNode, srcExists := nodeMap[edge.Source]
		if !srcExists {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("edges[%s].source", edge.ID),
				Message: fmt.Sprintf("source node %q does not exist", edge.Source),
			})
			continue
		}

		_, tgtExists := nodeMap[edge.Target]
		if !tgtExists {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("edges[%s].target", edge.ID),
				Message: fmt.Sprintf("target node %q does not exist", edge.Target),
			})
			continue
		}

		srcPorts := portMap[edge.Source]
		srcPort, srcPortExists := srcPorts[edge.SourcePort]
		if !srcPortExists {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("edges[%s].source_port", edge.ID),
				Message: fmt.Sprintf("source port %q does not exist on node %q", edge.SourcePort, srcNode.ID),
			})
			continue
		}

		tgtPorts := portMap[edge.Target]
		tgtPort, tgtPortExists := tgtPorts[edge.TargetPort]
		if !tgtPortExists {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("edges[%s].target_port", edge.ID),
				Message: fmt.Sprintf("target port %q does not exist on node %q", edge.TargetPort, edge.Target),
			})
			continue
		}

		if srcPort.Direction != domain.PortOutput {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("edges[%s].source_port", edge.ID),
				Message: fmt.Sprintf("source port %q on node %q is not an output port", edge.SourcePort, edge.Source),
			})
		}

		if tgtPort.Direction != domain.PortInput {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("edges[%s].target_port", edge.ID),
				Message: fmt.Sprintf("target port %q on node %q is not an input port", edge.TargetPort, edge.Target),
			})
		}

		if !domain.CanConnect(srcPort.Type, tgtPort.Type) {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("edges[%s]", edge.ID),
				Message: fmt.Sprintf("port type %q cannot connect to %q", srcPort.Type, tgtPort.Type),
			})
		}

		connectedInputs[edge.Target][edge.TargetPort] = true
	}

	// Check required input ports are connected
	for _, n := range graph.Nodes {
		for _, p := range n.Ports {
			if p.Direction == domain.PortInput && p.Required {
				if !connectedInputs[n.ID][p.ID] {
					errs = append(errs, ValidationError{
						Field:   fmt.Sprintf("nodes[%s].ports[%s]", n.ID, p.ID),
						Message: fmt.Sprintf("required input port %q is not connected", p.Name),
					})
				}
			}
		}
	}

	// Cycle detection via topological sort attempt
	if _, err := TopoSort(graph); err != nil {
		errs = append(errs, ValidationError{
			Field:   "graph",
			Message: err.Error(),
		})
	}

	return errs
}

// TopoSort performs a topological sort of the graph nodes using Kahn's algorithm.
// Returns nodes in execution order, or an error if a cycle is detected.
func TopoSort(graph domain.PipelineGraph) ([]domain.NodeInstance, error) {
	nodeMap := make(map[string]domain.NodeInstance, len(graph.Nodes))
	for _, n := range graph.Nodes {
		nodeMap[n.ID] = n
	}

	// Build adjacency list and in-degree count
	inDegree := make(map[string]int, len(graph.Nodes))
	successors := make(map[string][]string, len(graph.Nodes))
	for _, n := range graph.Nodes {
		inDegree[n.ID] = 0
	}

	for _, edge := range graph.Edges {
		// Only count edges between existing nodes
		if _, ok := nodeMap[edge.Source]; !ok {
			continue
		}
		if _, ok := nodeMap[edge.Target]; !ok {
			continue
		}
		successors[edge.Source] = append(successors[edge.Source], edge.Target)
		inDegree[edge.Target]++
	}

	// Seed the queue with nodes that have no incoming edges
	var queue []string
	for _, n := range graph.Nodes {
		if inDegree[n.ID] == 0 {
			queue = append(queue, n.ID)
		}
	}

	var sorted []domain.NodeInstance
	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]
		sorted = append(sorted, nodeMap[nodeID])

		for _, succ := range successors[nodeID] {
			inDegree[succ]--
			if inDegree[succ] == 0 {
				queue = append(queue, succ)
			}
		}
	}

	if len(sorted) != len(graph.Nodes) {
		return nil, fmt.Errorf("cycle detected in graph: sorted %d of %d nodes", len(sorted), len(graph.Nodes))
	}

	return sorted, nil
}
