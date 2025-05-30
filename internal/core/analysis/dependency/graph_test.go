package dependency_test

import (
	"testing"

	"codedna/internal/core/analysis/dependency"
)

func TestNewGraph(t *testing.T) {
	graph := dependency.NewGraph()

	if graph.Nodes == nil {
		t.Error("Expected Nodes map to be initialized")
	}

	if graph.Dependencies == nil {
		t.Error("Expected Dependencies slice to be initialized")
	}

	if graph.DirectDependencies == nil {
		t.Error("Expected DirectDependencies slice to be initialized")
	}

	if graph.IndirectDependencies == nil {
		t.Error("Expected IndirectDependencies slice to be initialized")
	}

	if graph.ExternalDependencies == nil {
		t.Error("Expected ExternalDependencies map to be initialized")
	}
}

func TestAddNode(t *testing.T) {
	graph := dependency.NewGraph()

	node := &dependency.Node{
		ID:   "test",
		Type: dependency.ModuleNode,
		Name: "testpkg",
		Path: "github.com/test/pkg",
	}

	graph.AddNode(node)

	if got, exists := graph.Node("test"); !exists {
		t.Error("Expected node to exist in graph")
	} else if got != node {
		t.Errorf("Expected to get same node back, got different node")
	}
}

func TestAddDependency(t *testing.T) {
	graph := dependency.NewGraph()

	// Add nodes
	fromNode := &dependency.Node{
		ID:   "from",
		Type: dependency.ModuleNode,
		Name: "frompkg",
	}
	toNode := &dependency.Node{
		ID:   "to",
		Type: dependency.ModuleNode,
		Name: "topkg",
	}
	graph.AddNode(fromNode)
	graph.AddNode(toNode)

	// Add dependency
	dep := dependency.Dependency{
		From: "from",
		To:   "to",
		Type: dependency.Include,
		Location: dependency.Location{
			File:   "test.go",
			Line:   1,
			Column: 1,
		},
	}
	graph.AddDependency(dep)

	// Check dependency was added
	if len(graph.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(graph.Dependencies))
	}

	// Check direct dependency was added
	if len(graph.DirectDependencies) != 1 {
		t.Errorf("Expected 1 direct dependency, got %d", len(graph.DirectDependencies))
	}

	// Test getting dependencies from source
	deps := graph.DependenciesFrom("from")
	if len(deps) != 1 {
		t.Errorf("Expected 1 dependency for 'from', got %d", len(deps))
	}

	// Test getting dependencies to target
	deps = graph.DependenciesTo("to")
	if len(deps) != 1 {
		t.Errorf("Expected 1 dependency to 'to', got %d", len(deps))
	}
}

func TestExternalDependency(t *testing.T) {
	graph := dependency.NewGraph()

	// Add external node
	extNode := &dependency.Node{
		ID:         "external",
		Type:       dependency.ModuleNode,
		Name:       "fmt",
		Path:       "fmt",
		IsExternal: true,
	}
	graph.AddNode(extNode)

	// Add dependency to external module
	dep := dependency.Dependency{
		From:       "main",
		To:         "external",
		Type:       dependency.Include,
		IsExternal: true,
	}
	graph.AddDependency(dep)

	// Check external dependency was recorded
	if len(graph.ExternalDependencies) != 1 {
		t.Errorf("Expected 1 external dependency, got %d", len(graph.ExternalDependencies))
	}

	if ext, exists := graph.ExternalDependencies["external"]; !exists {
		t.Error("Expected external dependency to exist")
	} else if ext != extNode {
		t.Error("Expected external dependency to match node")
	}
}
