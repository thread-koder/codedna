package dependency

import (
	"strings"
)

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
}

// Creates a new dependency graph
func NewGraph() *Graph {
	return &Graph{
		Nodes:        make(map[string]*Node),
		Dependencies: make([]Dependency, 0),
	}
}

// Adds a node to the graph
func (g *Graph) AddNode(node *Node) {
	g.Nodes[node.ID] = node
}

// Adds a dependency to the graph
func (g *Graph) AddDependency(dep Dependency) {
	g.Dependencies = append(g.Dependencies, dep)
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

// Returns all external nodes
func (g *Graph) ExternalNodes() map[string]*Node {
	externals := make(map[string]*Node)
	for id, node := range g.Nodes {
		if node.IsExternal {
			externals[id] = node
		}
	}
	return externals
}

// Finds all circular dependencies in the graph
func (g *Graph) FindCircularDependencies() [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	path := make(map[string]bool)
	uniqueCycles := make(map[string]struct{})

	// Start DFS from each unvisited node
	for id := range g.Nodes {
		if !visited[id] {
			g.findCycles(id, []string{}, visited, path, uniqueCycles, &cycles)
		}
	}

	return cycles
}

// Helper function to find cycles using DFS
func (g *Graph) findCycles(nodeID string, currentPath []string, visited, path map[string]bool, uniqueCycles map[string]struct{}, cycles *[][]string) {
	visited[nodeID] = true
	path[nodeID] = true
	// Use defer to ensure we clean up the path entry even if we panic or return early
	defer delete(path, nodeID)

	currentPath = append(currentPath, nodeID)

	// Check all dependencies from this node
	for _, dep := range g.DependenciesFrom(nodeID) {
		if !path[dep.To] {
			if !visited[dep.To] {
				g.findCycles(dep.To, currentPath, visited, path, uniqueCycles, cycles)
			}
		} else {
			// Found a cycle
			cycle := []string{}
			// Find where the cycle starts
			start := -1
			for i, node := range currentPath {
				if node == dep.To {
					start = i
					break
				}
			}
			if start >= 0 {
				// Add nodes in the correct order and complete the cycle
				cycle = append(cycle, currentPath[start:]...)
				cycle = append(cycle, dep.To) // Add the closing node to complete the cycle

				// Normalize the cycle and remove the duplicate closing node
				normalized := g.normalizeCycle(cycle[:len(cycle)-1])

				// Convert the normalized cycle to a string for deduplication
				cycleKey := g.cycleToString(normalized)

				// Only add if we haven't seen this cycle before
				if _, exists := uniqueCycles[cycleKey]; !exists {
					uniqueCycles[cycleKey] = struct{}{}
					*cycles = append(*cycles, normalized)
				}
			}
		}
	}
}

// Helper function to normalize a cycle by finding the lexicographically smallest rotation
func (g *Graph) normalizeCycle(cycle []string) []string {
	if len(cycle) <= 1 {
		return cycle
	}

	// Find the lexicographically smallest rotation
	minRotation := cycle
	for i := 1; i < len(cycle); i++ {
		// Create a rotation by moving i elements from front to back
		rotation := append(cycle[i:], cycle[:i]...)
		// Compare with current minimum
		if g.compareStringSlices(rotation, minRotation) < 0 {
			minRotation = rotation
		}
	}
	return minRotation
}

// Helper function to compare two string slices lexicographically
func (g *Graph) compareStringSlices(a, b []string) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}

// Helper function to convert a cycle to a string for deduplication
func (g *Graph) cycleToString(cycle []string) string {
	// Since the cycle is already normalized, we can just join it
	return strings.Join(cycle, "|")
}

// Checks if a specific direct dependency exists
func (g *Graph) HasDependency(from, to string, depType DependencyType) bool {
	for _, dep := range g.Dependencies {
		if dep.From == from && dep.To == to && dep.Type == depType {
			return true
		}
	}
	return false
}

// Returns all indirect dependencies from a given node ID
func (g *Graph) IndirectDependenciesFrom(nodeID string, maxDepth int) []Dependency {
	visited := make(map[string]bool)
	visited[nodeID] = true
	return g.findIndirectDependencies(nodeID, visited, maxDepth)
}

// Helper function to recursively find indirect dependencies
func (g *Graph) findIndirectDependencies(nodeID string, visited map[string]bool, depth int) []Dependency {
	if depth == 0 {
		return nil
	}

	var deps []Dependency

	// Get direct dependencies from this node
	for _, dep := range g.DependenciesFrom(nodeID) {
		// Skip if we've already visited this node
		if visited[dep.To] {
			continue
		}
		visited[dep.To] = true

		// Add this dependency as it's indirect from the original node
		deps = append(deps, dep)

		// Add indirect dependencies from this node
		indirectDeps := g.findIndirectDependencies(dep.To, visited, depth-1)
		deps = append(deps, indirectDeps...)
	}

	return deps
}

// Returns all indirect dependencies to a given node ID
func (g *Graph) IndirectDependenciesTo(nodeID string, maxDepth int) []Dependency {
	visited := make(map[string]bool)
	visited[nodeID] = true
	return g.findIndirectDependentsTo(nodeID, visited, maxDepth)
}

// Helper function to recursively find indirect dependents
func (g *Graph) findIndirectDependentsTo(nodeID string, visited map[string]bool, depth int) []Dependency {
	if depth == 0 {
		return nil
	}

	var deps []Dependency

	// Get direct dependents of this node
	for _, dep := range g.DependenciesTo(nodeID) {
		// Skip if we've already visited this node
		if visited[dep.From] {
			continue
		}
		visited[dep.From] = true

		// Add this dependency as it's indirect to the original node
		deps = append(deps, dep)

		// Add indirect dependents from this node
		indirectDeps := g.findIndirectDependentsTo(dep.From, visited, depth-1)
		deps = append(deps, indirectDeps...)
	}

	return deps
}

// Clears all nodes and dependencies from the graph
func (g *Graph) Clear() {
	g.Nodes = make(map[string]*Node)
	g.Dependencies = make([]Dependency, 0)
}

// Returns all nodes of a specific type
func (g *Graph) NodesOfType(nodeType NodeType) []*Node {
	var nodes []*Node
	for _, node := range g.Nodes {
		if node.Type == nodeType {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// Returns all dependencies of a specific type
func (g *Graph) DependenciesOfType(depType DependencyType) []Dependency {
	var deps []Dependency
	for _, dep := range g.Dependencies {
		if dep.Type == depType {
			deps = append(deps, dep)
		}
	}
	return deps
}

// Returns whether a node exists in the graph
func (g *Graph) HasNode(nodeID string) bool {
	_, exists := g.Nodes[nodeID]
	return exists
}

// Returns whether a node is a root (has no incoming dependencies)
func (g *Graph) isRoot(nodeID string) bool {
	for _, dep := range g.Dependencies {
		if dep.To == nodeID {
			return false
		}
	}
	return true
}

// Returns whether a node is a leaf (has no outgoing dependencies)
func (g *Graph) isLeaf(nodeID string) bool {
	for _, dep := range g.Dependencies {
		if dep.From == nodeID {
			return false
		}
	}
	return true
}

// Returns all root nodes (nodes with no incoming dependencies)
func (g *Graph) RootNodes() []*Node {
	var roots []*Node
	for id, node := range g.Nodes {
		if g.isRoot(id) {
			roots = append(roots, node)
		}
	}
	return roots
}

// Returns all leaf nodes (nodes with no outgoing dependencies)
func (g *Graph) LeafNodes() []*Node {
	var leaves []*Node
	for id, node := range g.Nodes {
		if g.isLeaf(id) {
			leaves = append(leaves, node)
		}
	}
	return leaves
}

// Returns all external nodes recursively up to maxDepth
func (g *Graph) ExternalNodesRecursive(maxDepth int) map[string]*Node {
	externals := g.ExternalNodes()
	if maxDepth <= 0 {
		return externals
	}

	// Track processed nodes to avoid cycles
	processed := make(map[string]bool)
	for id := range externals {
		g.findExternalNodesRecursive(id, processed, externals, maxDepth)
	}

	return externals
}

// Helper function to recursively find external nodes
func (g *Graph) findExternalNodesRecursive(nodeID string, processed map[string]bool, externals map[string]*Node, depth int) {
	// Skip if already processed or at max depth
	if processed[nodeID] || depth <= 0 {
		return
	}
	processed[nodeID] = true

	// Get all dependencies from this node
	for _, dep := range g.DependenciesFrom(nodeID) {
		// Skip if already collected
		if _, exists := externals[dep.To]; exists {
			continue
		}

		// Get the target node
		if node, exists := g.Nodes[dep.To]; exists && node.IsExternal {
			// Add to externals
			externals[dep.To] = node
			// Recursively process its dependencies
			g.findExternalNodesRecursive(dep.To, processed, externals, depth-1)
		}
	}
}
