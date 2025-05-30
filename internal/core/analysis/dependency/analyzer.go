package dependency

import (
	"codedna/internal/core/parser/ast"
)

// Defines the interface for dependency analysis
type Analyzer interface {
	// Performs dependency analysis on the given AST node
	Analyze(node ast.Node) (*Graph, error)

	// Checks if a specific dependency exists
	// from: source identifier
	// to: target identifier
	// depType: type of dependency to check for
	HasDependency(from, to string, depType DependencyType) bool

	// Returns all dependencies for a given node identifier
	Dependencies(nodeID string) []Dependency

	// Returns all nodes that depend on the given node identifier
	Dependents(nodeID string) []Dependency

	// Detects and returns any circular dependencies
	FindCircularDependencies() [][]string

	// Returns all external dependencies
	ExternalDependencies() map[string]*Node

	// Merges two dependency graphs
	// This is useful when analyzing multiple files or packages
	Merge(other *Graph) error
}

// Holds configuration for dependency analysis
type Config struct {
	// Whether to include indirect dependencies
	IncludeIndirect bool

	// Whether to analyze external dependencies
	AnalyzeExternal bool

	// Maximum depth for indirect dependency analysis
	MaxDepth int

	// Additional analyzer-specific options
	Options map[string]any
}

// Creates a default configuration
func NewConfig() *Config {
	return &Config{
		IncludeIndirect: true,
		AnalyzeExternal: true,
		MaxDepth:        10,
		Options:         make(map[string]any),
	}
}
