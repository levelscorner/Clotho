package engine

import (
	"testing"

	"github.com/user/clotho/internal/domain"
)

// --- Test helpers ---

func makeNode(t *testing.T, id string, nodeType domain.NodeType, ports []domain.Port) domain.NodeInstance {
	t.Helper()
	return domain.NodeInstance{
		ID:    id,
		Type:  nodeType,
		Label: id,
		Ports: ports,
	}
}

func makeEdge(t *testing.T, id, src, srcPort, tgt, tgtPort string) domain.Edge {
	t.Helper()
	return domain.Edge{
		ID:         id,
		Source:     src,
		SourcePort: srcPort,
		Target:     tgt,
		TargetPort: tgtPort,
	}
}

func makeGraph(t *testing.T, nodes []domain.NodeInstance, edges []domain.Edge) domain.PipelineGraph {
	t.Helper()
	return domain.PipelineGraph{
		Nodes: nodes,
		Edges: edges,
	}
}

func agentPorts() []domain.Port {
	return []domain.Port{
		{ID: "in", Name: "Input", Type: domain.PortTypeText, Direction: domain.PortInput},
		{ID: "out", Name: "Output", Type: domain.PortTypeText, Direction: domain.PortOutput},
	}
}

func outputOnlyPorts() []domain.Port {
	return []domain.Port{
		{ID: "out", Name: "Output", Type: domain.PortTypeText, Direction: domain.PortOutput},
	}
}

func inputOnlyPorts() []domain.Port {
	return []domain.Port{
		{ID: "in", Name: "Input", Type: domain.PortTypeText, Direction: domain.PortInput},
	}
}

func requiredInputPorts() []domain.Port {
	return []domain.Port{
		{ID: "in", Name: "Input", Type: domain.PortTypeText, Direction: domain.PortInput, Required: true},
		{ID: "out", Name: "Output", Type: domain.PortTypeText, Direction: domain.PortOutput},
	}
}

func imageOutputPorts() []domain.Port {
	return []domain.Port{
		{ID: "out", Name: "Output", Type: domain.PortTypeImage, Direction: domain.PortOutput},
	}
}

// --- ValidateGraph tests ---

func TestValidateGraph(t *testing.T) {
	t.Parallel()

	t.Run("valid linear graph A->B->C", func(t *testing.T) {
		t.Parallel()
		g := makeGraph(t,
			[]domain.NodeInstance{
				makeNode(t, "A", domain.NodeTypeTool, outputOnlyPorts()),
				makeNode(t, "B", domain.NodeTypeAgent, agentPorts()),
				makeNode(t, "C", domain.NodeTypeAgent, agentPorts()),
			},
			[]domain.Edge{
				makeEdge(t, "e1", "A", "out", "B", "in"),
				makeEdge(t, "e2", "B", "out", "C", "in"),
			},
		)
		errs := ValidateGraph(g)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %v", errs)
		}
	})

	t.Run("empty graph", func(t *testing.T) {
		t.Parallel()
		g := makeGraph(t, nil, nil)
		errs := ValidateGraph(g)
		if len(errs) != 0 {
			t.Errorf("expected no errors for empty graph, got %v", errs)
		}
	})

	t.Run("duplicate node IDs", func(t *testing.T) {
		t.Parallel()
		g := makeGraph(t,
			[]domain.NodeInstance{
				makeNode(t, "A", domain.NodeTypeAgent, agentPorts()),
				makeNode(t, "A", domain.NodeTypeAgent, agentPorts()),
			},
			nil,
		)
		errs := ValidateGraph(g)
		if len(errs) == 0 {
			t.Fatal("expected error for duplicate node IDs")
		}
		found := false
		for _, e := range errs {
			if e.Field == "nodes" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected error on field 'nodes', got %v", errs)
		}
	})

	t.Run("edge references non-existent source node", func(t *testing.T) {
		t.Parallel()
		g := makeGraph(t,
			[]domain.NodeInstance{
				makeNode(t, "B", domain.NodeTypeAgent, agentPorts()),
			},
			[]domain.Edge{
				makeEdge(t, "e1", "MISSING", "out", "B", "in"),
			},
		)
		errs := ValidateGraph(g)
		if len(errs) == 0 {
			t.Fatal("expected error for missing source node")
		}
	})

	t.Run("edge references non-existent target node", func(t *testing.T) {
		t.Parallel()
		g := makeGraph(t,
			[]domain.NodeInstance{
				makeNode(t, "A", domain.NodeTypeAgent, agentPorts()),
			},
			[]domain.Edge{
				makeEdge(t, "e1", "A", "out", "MISSING", "in"),
			},
		)
		errs := ValidateGraph(g)
		if len(errs) == 0 {
			t.Fatal("expected error for missing target node")
		}
	})

	t.Run("edge references non-existent source port", func(t *testing.T) {
		t.Parallel()
		g := makeGraph(t,
			[]domain.NodeInstance{
				makeNode(t, "A", domain.NodeTypeAgent, agentPorts()),
				makeNode(t, "B", domain.NodeTypeAgent, agentPorts()),
			},
			[]domain.Edge{
				makeEdge(t, "e1", "A", "bad_port", "B", "in"),
			},
		)
		errs := ValidateGraph(g)
		if len(errs) == 0 {
			t.Fatal("expected error for missing source port")
		}
	})

	t.Run("edge references non-existent target port", func(t *testing.T) {
		t.Parallel()
		g := makeGraph(t,
			[]domain.NodeInstance{
				makeNode(t, "A", domain.NodeTypeAgent, agentPorts()),
				makeNode(t, "B", domain.NodeTypeAgent, agentPorts()),
			},
			[]domain.Edge{
				makeEdge(t, "e1", "A", "out", "B", "bad_port"),
			},
		)
		errs := ValidateGraph(g)
		if len(errs) == 0 {
			t.Fatal("expected error for missing target port")
		}
	})

	t.Run("port type incompatibility image to text", func(t *testing.T) {
		t.Parallel()
		g := makeGraph(t,
			[]domain.NodeInstance{
				makeNode(t, "A", domain.NodeTypeTool, imageOutputPorts()),
				makeNode(t, "B", domain.NodeTypeAgent, agentPorts()),
			},
			[]domain.Edge{
				makeEdge(t, "e1", "A", "out", "B", "in"),
			},
		)
		errs := ValidateGraph(g)
		if len(errs) == 0 {
			t.Fatal("expected error for type incompatibility")
		}
		found := false
		for _, e := range errs {
			if e.Field == "edges[e1]" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected type compatibility error on edge, got %v", errs)
		}
	})

	t.Run("required input port not connected", func(t *testing.T) {
		t.Parallel()
		g := makeGraph(t,
			[]domain.NodeInstance{
				makeNode(t, "A", domain.NodeTypeAgent, requiredInputPorts()),
			},
			nil,
		)
		errs := ValidateGraph(g)
		if len(errs) == 0 {
			t.Fatal("expected error for unconnected required port")
		}
		found := false
		for _, e := range errs {
			if e.Field == "nodes[A].ports[in]" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected required port error, got %v", errs)
		}
	})

	t.Run("cycle A->B->A", func(t *testing.T) {
		t.Parallel()
		g := makeGraph(t,
			[]domain.NodeInstance{
				makeNode(t, "A", domain.NodeTypeAgent, agentPorts()),
				makeNode(t, "B", domain.NodeTypeAgent, agentPorts()),
			},
			[]domain.Edge{
				makeEdge(t, "e1", "A", "out", "B", "in"),
				makeEdge(t, "e2", "B", "out", "A", "in"),
			},
		)
		errs := ValidateGraph(g)
		if len(errs) == 0 {
			t.Fatal("expected error for cycle")
		}
		found := false
		for _, e := range errs {
			if e.Field == "graph" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected cycle error on field 'graph', got %v", errs)
		}
	})

	t.Run("source port is not output direction", func(t *testing.T) {
		t.Parallel()
		g := makeGraph(t,
			[]domain.NodeInstance{
				makeNode(t, "A", domain.NodeTypeAgent, agentPorts()),
				makeNode(t, "B", domain.NodeTypeAgent, agentPorts()),
			},
			[]domain.Edge{
				makeEdge(t, "e1", "A", "in", "B", "in"), // using input port as source
			},
		)
		errs := ValidateGraph(g)
		if len(errs) == 0 {
			t.Fatal("expected error for source port direction")
		}
	})
}

// --- TopoSort tests ---

func TestTopoSort(t *testing.T) {
	t.Parallel()

	t.Run("linear A->B->C", func(t *testing.T) {
		t.Parallel()
		g := makeGraph(t,
			[]domain.NodeInstance{
				makeNode(t, "A", domain.NodeTypeTool, outputOnlyPorts()),
				makeNode(t, "B", domain.NodeTypeAgent, agentPorts()),
				makeNode(t, "C", domain.NodeTypeAgent, agentPorts()),
			},
			[]domain.Edge{
				makeEdge(t, "e1", "A", "out", "B", "in"),
				makeEdge(t, "e2", "B", "out", "C", "in"),
			},
		)
		sorted, err := TopoSort(g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sorted) != 3 {
			t.Fatalf("expected 3 nodes, got %d", len(sorted))
		}
		if sorted[0].ID != "A" {
			t.Errorf("sorted[0] = %q, want %q", sorted[0].ID, "A")
		}
		if sorted[1].ID != "B" {
			t.Errorf("sorted[1] = %q, want %q", sorted[1].ID, "B")
		}
		if sorted[2].ID != "C" {
			t.Errorf("sorted[2] = %q, want %q", sorted[2].ID, "C")
		}
	})

	t.Run("diamond A->B,A->C,B->D,C->D", func(t *testing.T) {
		t.Parallel()
		g := makeGraph(t,
			[]domain.NodeInstance{
				makeNode(t, "A", domain.NodeTypeTool, outputOnlyPorts()),
				makeNode(t, "B", domain.NodeTypeAgent, agentPorts()),
				makeNode(t, "C", domain.NodeTypeAgent, agentPorts()),
				makeNode(t, "D", domain.NodeTypeAgent, agentPorts()),
			},
			[]domain.Edge{
				makeEdge(t, "e1", "A", "out", "B", "in"),
				makeEdge(t, "e2", "A", "out", "C", "in"),
				makeEdge(t, "e3", "B", "out", "D", "in"),
				makeEdge(t, "e4", "C", "out", "D", "in"),
			},
		)
		sorted, err := TopoSort(g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sorted) != 4 {
			t.Fatalf("expected 4 nodes, got %d", len(sorted))
		}
		// A must be first, D must be last
		if sorted[0].ID != "A" {
			t.Errorf("sorted[0] = %q, want %q", sorted[0].ID, "A")
		}
		if sorted[3].ID != "D" {
			t.Errorf("sorted[3] = %q, want %q", sorted[3].ID, "D")
		}
	})

	t.Run("single node", func(t *testing.T) {
		t.Parallel()
		g := makeGraph(t,
			[]domain.NodeInstance{
				makeNode(t, "A", domain.NodeTypeAgent, agentPorts()),
			},
			nil,
		)
		sorted, err := TopoSort(g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sorted) != 1 {
			t.Fatalf("expected 1 node, got %d", len(sorted))
		}
		if sorted[0].ID != "A" {
			t.Errorf("sorted[0] = %q, want %q", sorted[0].ID, "A")
		}
	})

	t.Run("disconnected nodes returns all", func(t *testing.T) {
		t.Parallel()
		g := makeGraph(t,
			[]domain.NodeInstance{
				makeNode(t, "A", domain.NodeTypeAgent, agentPorts()),
				makeNode(t, "B", domain.NodeTypeAgent, agentPorts()),
				makeNode(t, "C", domain.NodeTypeAgent, agentPorts()),
			},
			nil,
		)
		sorted, err := TopoSort(g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sorted) != 3 {
			t.Fatalf("expected 3 nodes, got %d", len(sorted))
		}
	})

	t.Run("cycle returns error", func(t *testing.T) {
		t.Parallel()
		g := makeGraph(t,
			[]domain.NodeInstance{
				makeNode(t, "A", domain.NodeTypeAgent, agentPorts()),
				makeNode(t, "B", domain.NodeTypeAgent, agentPorts()),
			},
			[]domain.Edge{
				makeEdge(t, "e1", "A", "out", "B", "in"),
				makeEdge(t, "e2", "B", "out", "A", "in"),
			},
		)
		_, err := TopoSort(g)
		if err == nil {
			t.Fatal("expected error for cycle, got nil")
		}
	})

	t.Run("empty graph", func(t *testing.T) {
		t.Parallel()
		g := makeGraph(t, nil, nil)
		sorted, err := TopoSort(g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sorted) != 0 {
			t.Errorf("expected 0 nodes, got %d", len(sorted))
		}
	})
}
