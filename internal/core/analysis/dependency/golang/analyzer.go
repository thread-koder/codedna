package godependency

import (
	"codedna/internal/core/analysis/dependency"
	"codedna/internal/core/parser/ast"
	goparser "codedna/internal/core/parser/golang"
	"fmt"
	"maps"
)

// Analyzer implements the dependency.Analyzer interface for Go code
type Analyzer struct {
	config *dependency.Config
	graph  *dependency.Graph // Stores the current graph being analyzed
}

// Creates a new Go dependency analyzer
func NewAnalyzer(config *dependency.Config) *Analyzer {
	if config == nil {
		config = dependency.NewConfig()
	}
	return &Analyzer{
		config: config,
		graph:  dependency.NewGraph(),
	}
}

// Performs dependency analysis on the given AST node
func (a *Analyzer) Analyze(node ast.Node) error {
	// Reset the graph for a new analysis
	a.graph = dependency.NewGraph()

	// Get package name from module node
	pkgName, ok := node.Attributes()["package_name"].(string)
	if !ok {
		return fmt.Errorf("missing package name in module node")
	}

	// Create package node
	pkgNode := &dependency.Node{
		ID:   pkgName,
		Type: dependency.ModuleNode,
		Name: pkgName,
		Path: pkgName,
	}
	a.graph.AddNode(pkgNode)

	// Process all children recursively
	for _, child := range node.Children() {
		if err := a.analyzeNode(child, pkgName, a.graph); err != nil {
			return fmt.Errorf("failed to analyze node: %w", err)
		}
	}

	return nil
}

// Processes a single AST node and its children
func (a *Analyzer) analyzeNode(node ast.Node, pkgName string, graph *dependency.Graph) error {
	switch node.Type() {
	case string(ast.Import):
		attrs := node.Attributes()
		path, ok := attrs["path"].(string)
		if !ok {
			return nil
		}

		// Create node for imported package
		importNode := &dependency.Node{
			ID:         path,
			Type:       dependency.ModuleNode,
			Name:       path,
			Path:       path,
			IsExternal: true,
		}
		graph.AddNode(importNode)

		// Add import dependency
		dep := dependency.Dependency{
			From:       pkgName,
			To:         path,
			Type:       dependency.Include,
			IsExternal: true,
			Location: dependency.Location{
				File:   node.Attributes()["file_path"].(string),
				Line:   node.Position().Line,
				Column: node.Position().Column,
			},
		}
		graph.AddDependency(dep)

	case string(ast.Type):
		if err := a.analyzeType(node, pkgName, graph); err != nil {
			return fmt.Errorf("failed to analyze type: %w", err)
		}

	case string(ast.Interface):
		if err := a.analyzeInterface(node, pkgName, graph); err != nil {
			return fmt.Errorf("failed to analyze interface: %w", err)
		}

	case string(ast.Function), string(ast.Method):
		if err := a.analyzeFunction(node, pkgName, graph); err != nil {
			return fmt.Errorf("failed to analyze function: %w", err)
		}

	case string(ast.Block):
		// Process block nodes recursively
		for _, child := range node.Children() {
			if err := a.analyzeNode(child, pkgName, graph); err != nil {
				return fmt.Errorf("failed to analyze node in block: %w", err)
			}
		}
	}

	return nil
}

// Processes a type declaration and its relationships
func (a *Analyzer) analyzeType(node ast.Node, pkgName string, graph *dependency.Graph) error {
	name, ok := node.Attributes()["name"].(string)
	if !ok {
		return nil // Skip unnamed types
	}

	// Create node for the type
	typeNode := &dependency.Node{
		ID:       fmt.Sprintf("%s.%s", pkgName, name),
		Type:     dependency.TypeNode,
		Name:     name,
		Path:     pkgName,
		Metadata: make(map[string]any),
	}
	graph.AddNode(typeNode)

	// Store methods in metadata
	if methods, ok := node.Attributes()["methods"].([]map[string]any); ok {
		typeNode.Metadata["methods"] = methods
	}

	// Handle embedded types (composition)
	embeddedTypes := make(map[string]bool)
	if fields, ok := node.Attributes()["fields"].([]map[string]any); ok {
		for _, field := range fields {
			if fieldType, ok := field["type"].(*goparser.TypeInfo); ok {
				// Check if it's an embedded field
				if embedded, ok := field["embedded"].(bool); ok && embedded {
					// Add composition dependency
					var embeddedID string
					if fieldType.Kind == "pointer" && fieldType.ElemType != nil {
						// For pointer types, use the element type name
						embeddedID = fmt.Sprintf("%s.%s", pkgName, fieldType.ElemType.Name)
					} else {
						embeddedID = fmt.Sprintf("%s.%s", pkgName, fieldType.Name)
					}
					embeddedTypes[embeddedID] = true
					dep := dependency.Dependency{
						From: typeNode.ID,
						To:   embeddedID,
						Type: dependency.Compose,
						Location: dependency.Location{
							File:   node.Attributes()["file_path"].(string),
							Line:   node.Position().Line,
							Column: node.Position().Column,
						},
					}
					graph.AddDependency(dep)
				} else {
					// Add regular type reference
					a.addTypeReference(typeNode, fieldType, pkgName, node.Position(), graph)
				}
			}
		}
	}

	// Check interface implementations
	if methods, ok := node.Attributes()["methods"].([]map[string]any); ok {
		// Build a map of method signatures for this struct
		structMethods := make(map[string]map[string]any)
		for _, method := range methods {
			if name, ok := method["name"].(string); ok {
				if sig, ok := method["signature"].(map[string]any); ok {
					structMethods[name] = sig
				}
			}
		}

		// Add methods from embedded types
		if fields, ok := node.Attributes()["fields"].([]map[string]any); ok {
			for _, field := range fields {
				if embedded, ok := field["embedded"].(bool); ok && embedded {
					if fieldType, ok := field["type"].(*goparser.TypeInfo); ok {
						// Get the base type name (handle pointer types)
						typeName := fieldType.Name
						if fieldType.Kind == "pointer" && fieldType.ElemType != nil {
							typeName = fieldType.ElemType.Name
						}
						// Find the embedded type node
						embeddedID := fmt.Sprintf("%s.%s", pkgName, typeName)
						if embeddedNode, ok := graph.Node(embeddedID); ok {
							if embeddedMethods, ok := embeddedNode.Metadata["methods"].([]map[string]any); ok {
								for _, embeddedMethod := range embeddedMethods {
									if name, ok := embeddedMethod["name"].(string); ok {
										if sig, ok := embeddedMethod["signature"].(map[string]any); ok {
											structMethods[name] = sig
										}
									}
								}
							}
						}
					}
				}
			}
		}

		// Find all interface nodes in the graph
		for _, n := range graph.Nodes {
			if n.Type != dependency.ContractNode {
				continue
			}

			// Get interface methods from metadata
			ifaceMethods, ok := n.Metadata["methods"].([]map[string]any)
			if !ok {
				continue
			}

			// Get embedded interfaces
			var allIfaceMethods []map[string]any
			allIfaceMethods = append(allIfaceMethods, ifaceMethods...)
			if embedded, ok := n.Metadata["embedded"].([]map[string]any); ok {
				for _, embeddedIface := range embedded {
					if embeddedType, ok := embeddedIface["type"].(*goparser.TypeInfo); ok {
						embeddedID := fmt.Sprintf("%s.%s", pkgName, embeddedType.Name)
						if embeddedNode, ok := graph.Node(embeddedID); ok {
							if embeddedMethods, ok := embeddedNode.Metadata["methods"].([]map[string]any); ok {
								allIfaceMethods = append(allIfaceMethods, embeddedMethods...)
							}
							// Also get methods from interfaces embedded in the embedded interface
							if embeddedEmbedded, ok := embeddedNode.Metadata["embedded"].([]map[string]any); ok {
								for _, embeddedEmbeddedIface := range embeddedEmbedded {
									if embeddedEmbeddedType, ok := embeddedEmbeddedIface["type"].(*goparser.TypeInfo); ok {
										embeddedEmbeddedID := fmt.Sprintf("%s.%s", pkgName, embeddedEmbeddedType.Name)
										if embeddedEmbeddedNode, ok := graph.Node(embeddedEmbeddedID); ok {
											if embeddedEmbeddedMethods, ok := embeddedEmbeddedNode.Metadata["methods"].([]map[string]any); ok {
												allIfaceMethods = append(allIfaceMethods, embeddedEmbeddedMethods...)
											}
										}
									}
								}
							}
						}
					}
				}
			}

			// Check if all interface methods are implemented
			implementsAll := true
			for _, ifaceMethod := range allIfaceMethods {
				ifaceMethodName, ok := ifaceMethod["name"].(string)
				if !ok {
					continue
				}

				// Find matching struct method
				structMethod, ok := structMethods[ifaceMethodName]
				if !ok {
					implementsAll = false
					break
				}

				// Compare method signatures
				if !a.compareMethodSignatures(structMethod, ifaceMethod["signature"].(map[string]any)) {
					implementsAll = false
					break
				}
			}

			// If all methods are implemented, add a Satisfy dependency
			if implementsAll {
				dep := dependency.Dependency{
					From: typeNode.ID,
					To:   n.ID,
					Type: dependency.Satisfy,
					Location: dependency.Location{
						File:   node.Attributes()["file_path"].(string),
						Line:   node.Position().Line,
						Column: node.Position().Column,
					},
				}
				graph.AddDependency(dep)
			}
		}
	}

	return nil
}

// Compares two method signatures for compatibility
func (a *Analyzer) compareMethodSignatures(sig1, sig2 map[string]any) bool {
	// Compare parameter types
	params1, ok1 := sig1["params"].([]*goparser.TypeInfo)
	params2, ok2 := sig2["params"].([]*goparser.TypeInfo)
	if !ok1 || !ok2 || len(params1) != len(params2) {
		return false
	}

	// Compare each parameter type
	for i := range params1 {
		if !a.compareTypes(params1[i], params2[i]) {
			return false
		}
	}

	// Compare return types
	returns1, ok1 := sig1["returns"].([]*goparser.TypeInfo)
	returns2, ok2 := sig2["returns"].([]*goparser.TypeInfo)
	if !ok1 || !ok2 || len(returns1) != len(returns2) {
		return false
	}

	// Compare each return type
	for i := range returns1 {
		if !a.compareTypes(returns1[i], returns2[i]) {
			return false
		}
	}

	return true
}

// Compares two types for compatibility
func (a *Analyzer) compareTypes(t1, t2 *goparser.TypeInfo) bool {
	if t1 == nil || t2 == nil {
		return t1 == t2
	}

	// Handle pointer types - consider them equal to their base types
	if t1.Kind == "pointer" && t1.ElemType != nil {
		t1 = t1.ElemType
	}
	if t2.Kind == "pointer" && t2.ElemType != nil {
		t2 = t2.ElemType
	}

	// Compare kind and name
	if t1.Kind != t2.Kind {
		return false
	}

	// For basic types, compare names
	if t1.Kind == "basic" {
		return t1.Name == t2.Name
	}

	// For slice types, compare element types
	if t1.Kind == "slice" {
		return a.compareTypes(t1.ElemType, t2.ElemType)
	}

	// For map types, compare key and value types
	if t1.Kind == "map" {
		return a.compareTypes(t1.KeyType, t2.KeyType) &&
			a.compareTypes(t1.ValueType, t2.ValueType)
	}

	return true
}

// Processes an interface declaration and its relationships
func (a *Analyzer) analyzeInterface(node ast.Node, pkgName string, graph *dependency.Graph) error {
	name, ok := node.Attributes()["name"].(string)
	if !ok {
		return nil // Skip unnamed interfaces
	}

	// Create node for the interface
	ifaceNode := &dependency.Node{
		ID:       fmt.Sprintf("%s.%s", pkgName, name),
		Type:     dependency.ContractNode,
		Name:     name,
		Path:     pkgName,
		Metadata: make(map[string]any),
	}
	graph.AddNode(ifaceNode)

	// Store methods in metadata
	if methods, ok := node.Attributes()["methods"].([]map[string]any); ok {
		ifaceNode.Metadata["methods"] = methods
	}

	// Handle embedded interfaces
	if embedded, ok := node.Attributes()["embedded"].([]map[string]any); ok {
		for _, embed := range embedded {
			if embedType, ok := embed["type"].(*goparser.TypeInfo); ok {
				// Add inheritance dependency
				dep := dependency.Dependency{
					From: ifaceNode.ID,
					To:   fmt.Sprintf("%s.%s", pkgName, embedType.Name),
					Type: dependency.Inherit,
					Location: dependency.Location{
						File:   node.Attributes()["file_path"].(string),
						Line:   node.Position().Line,
						Column: node.Position().Column,
					},
				}
				graph.AddDependency(dep)
			}
		}
	}

	// Handle method signatures
	if methods, ok := node.Attributes()["methods"].([]map[string]any); ok {
		for _, method := range methods {
			if sig, ok := method["signature"].(map[string]any); ok {
				// Add dependencies for parameter types
				if params, ok := sig["params"].([]*goparser.TypeInfo); ok {
					for _, param := range params {
						a.addTypeReference(ifaceNode, param, pkgName, node.Position(), graph)
					}
				}

				// Add dependencies for return types
				if returns, ok := sig["returns"].([]*goparser.TypeInfo); ok {
					for _, ret := range returns {
						a.addTypeReference(ifaceNode, ret, pkgName, node.Position(), graph)
					}
				}
			}
		}
	}

	return nil
}

// Processes a function declaration and its dependencies
func (a *Analyzer) analyzeFunction(node ast.Node, pkgName string, graph *dependency.Graph) error {
	name, ok := node.Attributes()["name"].(string)
	if !ok {
		return nil // Skip unnamed functions
	}

	// Create node for the function
	var funcID string
	if node.Type() == string(ast.Method) {
		// For methods, include the receiver type in the ID
		if recv, ok := node.Attributes()["receiver_type"].(*goparser.TypeInfo); ok {
			var recvName string
			if recv.Kind == "pointer" && recv.ElemType != nil {
				recvName = recv.ElemType.Name
			} else {
				recvName = recv.Name
			}
			funcID = fmt.Sprintf("%s.%s.%s", pkgName, recvName, name)
		} else {
			funcID = fmt.Sprintf("%s.%s", pkgName, name)
		}
	} else {
		funcID = fmt.Sprintf("%s.%s", pkgName, name)
	}

	funcNode := &dependency.Node{
		ID:   funcID,
		Type: dependency.FunctionNode,
		Name: name,
		Path: pkgName,
	}
	graph.AddNode(funcNode)

	// Process function signature
	if sig, ok := node.Attributes()["signature"].(map[string]any); ok {
		// Handle receiver type for methods
		if node.Type() == string(ast.Method) {
			if recv, ok := sig["receiver_type"].(*goparser.TypeInfo); ok {
				a.addTypeReference(funcNode, recv, pkgName, node.Position(), graph)
			}
		}

		// Handle parameter types
		if params, ok := sig["params"].([]*goparser.TypeInfo); ok {
			for _, param := range params {
				a.addTypeReference(funcNode, param, pkgName, node.Position(), graph)
			}
		}

		// Handle return types
		if returns, ok := sig["returns"].([]*goparser.TypeInfo); ok {
			for _, ret := range returns {
				a.addTypeReference(funcNode, ret, pkgName, node.Position(), graph)
			}
		}
	}

	// Process function body for dependencies
	if body, ok := node.Attributes()["body"].([]map[string]any); ok {
		for _, stmt := range body {
			if refs, ok := stmt["references"].([]map[string]any); ok {
				for _, ref := range refs {
					if refType, ok := ref["type"].(*goparser.TypeInfo); ok {
						// Add package reference if it's a package selector
						if refType.Kind == "package" {
							dep := dependency.Dependency{
								From: funcNode.ID,
								To:   refType.Name,
								Type: dependency.Reference,
								Location: dependency.Location{
									File:   funcNode.Path,
									Line:   node.Position().Line,
									Column: node.Position().Column,
								},
							}
							graph.AddDependency(dep)
						} else {
							a.addTypeReference(funcNode, refType, pkgName, node.Position(), graph)
						}
					}
				}
			}
		}
	}

	return nil
}

// Adds a dependency for a type reference
func (a *Analyzer) addTypeReference(source *dependency.Node, typeInfo *goparser.TypeInfo, pkgName string, pos ast.Position, graph *dependency.Graph) {
	if typeInfo == nil {
		return
	}

	// Handle pointer types
	if typeInfo.Kind == "pointer" && typeInfo.ElemType != nil {
		a.addTypeReference(source, typeInfo.ElemType, pkgName, pos, graph)
		return
	}

	// Handle slice types
	if typeInfo.Kind == "slice" && typeInfo.ElemType != nil {
		a.addTypeReference(source, typeInfo.ElemType, pkgName, pos, graph)
		return
	}

	// Handle map types
	if typeInfo.Kind == "map" {
		if typeInfo.KeyType != nil {
			a.addTypeReference(source, typeInfo.KeyType, pkgName, pos, graph)
		}
		if typeInfo.ValueType != nil {
			a.addTypeReference(source, typeInfo.ValueType, pkgName, pos, graph)
		}
		return
	}

	// Skip primitive types
	if typeInfo.Kind == "basic" {
		primitiveTypes := map[string]bool{
			"bool":       true,
			"string":     true,
			"int":        true,
			"int8":       true,
			"int16":      true,
			"int32":      true,
			"int64":      true,
			"uint":       true,
			"uint8":      true,
			"uint16":     true,
			"uint32":     true,
			"uint64":     true,
			"float32":    true,
			"float64":    true,
			"complex64":  true,
			"complex128": true,
			"byte":       true,
			"rune":       true,
			"error":      true,
		}
		if primitiveTypes[typeInfo.Name] {
			return
		}
	}

	// Add reference dependency
	dep := dependency.Dependency{
		From: source.ID,
		To:   fmt.Sprintf("%s.%s", pkgName, typeInfo.Name),
		Type: dependency.Reference,
		Location: dependency.Location{
			File:   source.Path,
			Line:   pos.Line,
			Column: pos.Column,
		},
	}
	graph.AddDependency(dep)
}

// Returns all dependencies for a given node identifier
func (a *Analyzer) Dependencies(nodeID string) []dependency.Dependency {
	if a.graph == nil {
		return nil
	}

	// Get direct dependencies
	deps := a.graph.DependenciesFrom(nodeID)

	// If indirect dependencies are enabled, add them
	if a.config.IncludeIndirect {
		deps = append(deps, a.graph.IndirectDependenciesFrom(nodeID, a.config.MaxDepth)...)
	}

	return deps
}

// Returns all nodes that depend on the given node identifier
func (a *Analyzer) Dependents(nodeID string) []dependency.Dependency {
	if a.graph == nil {
		return nil
	}

	// Get direct dependents
	deps := a.graph.DependenciesTo(nodeID)

	// If indirect dependencies are enabled, add them
	if a.config.IncludeIndirect {
		deps = append(deps, a.graph.IndirectDependenciesTo(nodeID, a.config.MaxDepth)...)
	}

	return deps
}

// Returns all external dependencies
func (a *Analyzer) ExternalDependencies() map[string]*dependency.Node {
	if a.graph == nil {
		return nil
	}

	if a.config.AnalyzeExternal {
		return a.graph.ExternalNodesRecursive(a.config.MaxDepth)
	}

	return a.graph.ExternalNodes()
}

// Checks if a specific dependency exists
func (a *Analyzer) HasDependency(from, to string, depType dependency.DependencyType) bool {
	if a.graph == nil {
		return false
	}

	// Check direct dependencies first
	if a.graph.HasDependency(from, to, depType) {
		return true
	}

	// If indirect dependencies are enabled, check them
	if a.config.IncludeIndirect {
		deps := a.graph.IndirectDependenciesFrom(from, a.config.MaxDepth)
		for _, dep := range deps {
			if dep.To == to && dep.Type == depType {
				return true
			}
		}
	}

	return false
}

// Returns all nodes of a specific type
func (a *Analyzer) NodesOfType(nodeType dependency.NodeType) []*dependency.Node {
	if a.graph == nil {
		return nil
	}
	return a.graph.NodesOfType(nodeType)
}

// Returns all dependencies of a specific type
func (a *Analyzer) DependenciesOfType(depType dependency.DependencyType) []dependency.Dependency {
	if a.graph == nil {
		return nil
	}
	return a.graph.DependenciesOfType(depType)
}

// Returns all root nodes (nodes with no incoming dependencies)
func (a *Analyzer) RootNodes() []*dependency.Node {
	if a.graph == nil {
		return nil
	}
	return a.graph.RootNodes()
}

// Returns all leaf nodes (nodes with no outgoing dependencies)
func (a *Analyzer) LeafNodes() []*dependency.Node {
	if a.graph == nil {
		return nil
	}
	return a.graph.LeafNodes()
}

// Returns whether a node exists
func (a *Analyzer) HasNode(nodeID string) bool {
	if a.graph == nil {
		return false
	}
	return a.graph.HasNode(nodeID)
}

// Returns a node by ID if it exists
func (a *Analyzer) Node(nodeID string) (*dependency.Node, bool) {
	if a.graph == nil {
		return nil, false
	}
	return a.graph.Node(nodeID)
}

// Returns all circular dependencies
func (a *Analyzer) FindCircularDependencies() [][]string {
	if a.graph == nil {
		return nil
	}
	return a.graph.FindCircularDependencies()
}

// Merges analysis results from another analyzer
func (a *Analyzer) Merge(other dependency.Analyzer) error {
	if a.graph == nil {
		return fmt.Errorf("analyzer graph is nil")
	}

	// Type assert to get access to the other analyzer's graph
	otherAnalyzer, ok := other.(*Analyzer)
	if !ok {
		return fmt.Errorf("cannot merge with non-Go analyzer")
	}

	if otherAnalyzer.graph == nil {
		return nil // Nothing to merge
	}

	// Merge nodes first
	for id, node := range otherAnalyzer.graph.Nodes {
		if existing, exists := a.graph.Nodes[id]; exists {
			// Update existing node's metadata and flags
			if existing.Metadata == nil {
				existing.Metadata = make(map[string]any)
			}
			maps.Copy(existing.Metadata, node.Metadata)
			existing.IsExternal = existing.IsExternal || node.IsExternal
		} else {
			// Create a new node with copied metadata
			newNode := &dependency.Node{
				ID:         node.ID,
				Type:       node.Type,
				Name:       node.Name,
				Path:       node.Path,
				IsExternal: node.IsExternal,
				Metadata:   make(map[string]any),
			}
			maps.Copy(newNode.Metadata, node.Metadata)
			a.graph.AddNode(newNode)
		}
	}

	// Create a map for faster dependency existence checks
	existingDeps := make(map[string]bool)
	for _, dep := range a.graph.Dependencies {
		key := fmt.Sprintf("%s:%s:%s", dep.From, dep.To, dep.Type)
		existingDeps[key] = true
	}

	// Merge dependencies efficiently
	for _, dep := range otherAnalyzer.graph.Dependencies {
		// Skip if nodes don't exist
		if _, exists := a.graph.Nodes[dep.From]; !exists {
			continue
		}
		if _, exists := a.graph.Nodes[dep.To]; !exists {
			continue
		}

		// Check if dependency exists using the map
		key := fmt.Sprintf("%s:%s:%s", dep.From, dep.To, dep.Type)
		if !existingDeps[key] {
			newDep := dependency.Dependency{
				From:       dep.From,
				To:         dep.To,
				Type:       dep.Type,
				IsExternal: dep.IsExternal,
				Location:   dep.Location,
			}
			a.graph.AddDependency(newDep)
		}
	}

	return nil
}

// Clears all analysis results
func (a *Analyzer) Clear() {
	a.graph = dependency.NewGraph()
}
