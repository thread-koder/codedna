// Package structure provides code structure analysis capabilities
package structure

// Node represents a parsed source code node
type Node interface {
	// Language returns the programming language of this node
	Language() string
}

// Analysis represents the results of code structure analysis
type Analysis interface {
	// Language returns the programming language that was analyzed
	Language() string
}

// Analyzer defines the interface for language-specific code structure analyzers
type Analyzer interface {
	// Language returns the programming language this analyzer handles
	Language() string

	// Analyze performs structural analysis on the given source code node
	Analyze(node Node) (Analysis, error)

	// Merge combines two analyses, handling language-specific differences
	Merge(base, other *Analysis) error
}
