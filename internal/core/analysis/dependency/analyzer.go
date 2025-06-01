package dependency

import (
	"codedna/internal/core/parser/ast"
)

// Defines the interface for dependency analysis
type Analyzer interface {
	// Performs dependency analysis on the given AST node
	Analyze(node ast.Node) error

	// Checks if a specific dependency exists
	// Includes indirect dependencies if configured
	HasDependency(from, to string, depType DependencyType) bool

	// Returns all dependencies for a given node identifier
	// Includes indirect dependencies if configured
	Dependencies(nodeID string) []Dependency

	// Returns all nodes that depend on the given node identifier
	// Includes indirect dependents if configured
	Dependents(nodeID string) []Dependency

	// Returns all external dependencies
	// Includes indirect external dependencies if configured
	ExternalDependencies() map[string]*Node

	// Returns all nodes of a specific type
	NodesOfType(nodeType NodeType) []*Node

	// Returns all dependencies of a specific type
	DependenciesOfType(depType DependencyType) []Dependency

	// Returns all root nodes (nodes with no incoming dependencies)
	RootNodes() []*Node

	// Returns all leaf nodes (nodes with no outgoing dependencies)
	LeafNodes() []*Node

	// Returns whether a node exists
	HasNode(nodeID string) bool

	// Returns a node by ID if it exists
	Node(nodeID string) (*Node, bool)

	// Returns all circular dependencies
	FindCircularDependencies() [][]string

	// Merges analysis results from another analyzer
	Merge(other Analyzer) error

	// Clears all analysis results
	Clear()
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
