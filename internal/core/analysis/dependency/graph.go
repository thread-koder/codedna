package dependency

// Represents the type of a node in the dependency graph
type NodeType string

const (
	// Represents a deployable code unit
	// This covers all forms of distributable code:
	// - Go: packages (as compiled units)
	// - Python: modules (as .py files)
	// - JavaScript: modules (as .js files)
	// - Java: compiled .class files
	// - Ruby: .rb files
	ModuleNode NodeType = "module"

	// Represents a type definition
	// This covers all forms of type definitions:
	// - Classes
	// - Structs
	// - Type aliases
	// - Enums
	TypeNode NodeType = "type"

	// Represents a callable unit
	// This covers all forms of callable code:
	// - Functions
	// - Methods
	// - Procedures
	// - Lambdas/Closures
	FunctionNode NodeType = "function"

	// Represents a type contract/specification
	// This covers all forms of type contracts:
	// - Interfaces
	// - Protocols
	// - Abstract classes
	// - Traits
	ContractNode NodeType = "contract"

	// Represents a value binding
	// This covers all forms of value storage:
	// - Variables
	// - Constants
	// - Properties
	// - Fields
	VariableNode NodeType = "variable"

	// Represents a logical grouping of symbols
	// This covers all forms of name organization without physical structure:
	// - C++: namespaces
	// - Java: package declarations
	// - Python: module qualifiers
	// - PHP: namespaces
	// - .NET: namespaces
	NamespaceNode NodeType = "namespace"
)

// Represents a node in the dependency graph
type Node struct {
	// Unique identifier for the node
	ID string

	// Type of the node
	Type NodeType

	// Name of the element
	Name string

	// Full path or qualified name
	Path string

	// Whether this is an external node
	IsExternal bool

	// Additional node information
	Metadata map[string]any
}

// Represents a dependency graph
type Graph struct {
	// All nodes in the graph, keyed by ID
	Nodes map[string]*Node

	// All dependencies (edges) in the graph
	Dependencies []Dependency

	// Direct module-level dependencies
	// These represent immediate Include relationships
	DirectDependencies []Dependency

	// Indirect module-level dependencies
	// These are derived from transitive Include relationships
	IndirectDependencies []Dependency

	// External module dependencies
	// These represent dependencies on code outside the analyzed codebase
	ExternalDependencies map[string]*Node

	// Circular dependencies found in the graph
	// Each slice represents a cycle in the dependency chain
	CircularDependencies [][]string
}

// Creates a new dependency graph
func NewGraph() *Graph {
	return &Graph{
		Nodes:                make(map[string]*Node),
		Dependencies:         make([]Dependency, 0),
		DirectDependencies:   make([]Dependency, 0),
		IndirectDependencies: make([]Dependency, 0),
		ExternalDependencies: make(map[string]*Node),
		CircularDependencies: make([][]string, 0),
	}
}

// Adds a node to the graph
func (g *Graph) AddNode(node *Node) {
	g.Nodes[node.ID] = node
}

// Adds a dependency to the graph
func (g *Graph) AddDependency(dep Dependency) {
	g.Dependencies = append(g.Dependencies, dep)

	// Categorize the dependency based on its type and external status
	if dep.IsExternal {
		if node, exists := g.Nodes[dep.To]; exists {
			g.ExternalDependencies[dep.To] = node
		}
	} else if dep.Type == Include {
		g.DirectDependencies = append(g.DirectDependencies, dep)
	}
}

// Retrieves a node by ID
func (g *Graph) Node(id string) (*Node, bool) {
	node, exists := g.Nodes[id]
	return node, exists
}

// Retrieves all dependencies originating from a given node ID
func (g *Graph) DependenciesFrom(nodeID string) []Dependency {
	var deps []Dependency
	for _, dep := range g.Dependencies {
		if dep.From == nodeID {
			deps = append(deps, dep)
		}
	}
	return deps
}

// Retrieves all nodes that depend on the given node ID
func (g *Graph) DependenciesTo(nodeID string) []Dependency {
	var deps []Dependency
	for _, dep := range g.Dependencies {
		if dep.To == nodeID {
			deps = append(deps, dep)
		}
	}
	return deps
}
