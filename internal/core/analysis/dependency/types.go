// Package dependency provides types and interfaces for dependency analysis
package dependency

// Represents the type of relationship between nodes
type DependencyType string

const (
	// Represents module inclusion dependency
	// This covers bringing in external code through:
	// - Go: imports
	// - Python: imports
	// - JavaScript: imports/requires
	// - C/C++: includes
	// - Ruby: requires
	Include DependencyType = "include"

	// Represents any usage/reference of a symbol
	Reference DependencyType = "reference"

	// Represents fulfilling a contract/interface/protocol
	Satisfy DependencyType = "satisfy"

	// Represents composition/containment relationships
	Compose DependencyType = "compose"

	// Represents type inheritance/extension relationships
	Inherit DependencyType = "inherit"
)

// Represents a position in source code
type Location struct {
	File   string
	Line   int
	Column int
}

// Represents a relationship between two nodes
type Dependency struct {
	// Source identifier (module, namespace, type, function, etc.)
	From string

	// Target identifier
	To string

	// Type of dependency
	Type DependencyType

	// Where this dependency occurs
	Location Location

	// Whether this is an external dependency
	IsExternal bool

	// Additional metadata about the dependency
	Metadata map[string]any
}
