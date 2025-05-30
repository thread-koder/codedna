package gostructure

import (
	"fmt"

	"codedna/internal/core/analysis/structure"
	"codedna/internal/core/parser/ast"
	goparser "codedna/internal/core/parser/golang"
)

// Implements the structure.Analyzer interface for Go code
type Analyzer struct{}

// Creates a new Go analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// Returns the language this analyzer handles
func (a *Analyzer) Language() string {
	return "go"
}

// Analyzes the structural analysis on Go code
func (a *Analyzer) Analyze(node structure.Node) (structure.Analysis, error) {
	// Type assert to Go node
	goNode, ok := node.(*Node)
	if !ok {
		return nil, fmt.Errorf("expected Go node, got %T", node)
	}

	// Create analysis result
	analysis := NewAnalysis()

	// Analyze the code
	_, err := a.analyzeNode(goNode.Node, analysis)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze Go code: %w", err)
	}

	// Detect language-specific patterns
	if err := a.detectPatterns(analysis); err != nil {
		return nil, fmt.Errorf("failed to detect Go patterns: %w", err)
	}

	return analysis, nil
}

// Analyzes a single Go AST node
func (a *Analyzer) analyzeNode(node ast.Node, analysis *Analysis) (*Element, error) {
	// Skip creating elements for blocks
	if node.Type() == "Block" {
		// Process block children directly
		for _, child := range node.Children() {
			childElement, err := a.analyzeNode(child, analysis)
			if err != nil {
				return nil, err
			}
			// Add child directly to the package
			if childElement != nil {
				if pkg := a.findPackage(analysis); pkg != nil {
					rel := &Relationship{
						Type:   RelationContains,
						Source: pkg,
						Target: childElement,
					}
					if !a.hasRelationship(analysis, rel) {
						analysis.Structure.Relationships = append(analysis.Structure.Relationships, rel)
					}
				}
			}
		}
		return nil, nil
	}

	// Get element type
	elemType := mapNodeType(node.Type())
	if elemType == "" {
		return nil, nil // Skip creating element
	}

	// Create element for the node
	element := &Element{
		Type:       elemType,
		Name:       nodeName(node),
		Attributes: node.Attributes(),
	}

	// Add element to structure
	analysis.Structure.Elements = append(analysis.Structure.Elements, element)

	// Process children
	for _, child := range node.Children() {
		childElement, err := a.analyzeNode(child, analysis)
		if err != nil {
			return nil, err
		}
		if childElement == nil {
			continue // Skip nil elements
		}

		// Create contains relationship with the correct source
		var source *Element
		if element.Type == ElementPackage {
			source = element // Use package element directly
		} else {
			source = a.findPackage(analysis) // Use package for non-package elements
		}

		if source != nil {
			rel := &Relationship{
				Type:   RelationContains,
				Source: source,
				Target: childElement,
			}
			if !a.hasRelationship(analysis, rel) {
				analysis.Structure.Relationships = append(analysis.Structure.Relationships, rel)
			}
		}
	}

	return element, nil
}

// Finds the package element
func (a *Analyzer) findPackage(analysis *Analysis) *Element {
	for _, elem := range analysis.Structure.Elements {
		if elem.Type == ElementPackage {
			return elem
		}
	}
	return nil
}

// Checks if a relationship already exists
func (a *Analyzer) hasRelationship(analysis *Analysis, rel *Relationship) bool {
	for _, existing := range analysis.Structure.Relationships {
		if existing.Type == rel.Type &&
			existing.Source == rel.Source &&
			existing.Target == rel.Target {
			return true
		}
	}
	return false
}

// Maps AST node types to element types
func mapNodeType(nodeType string) ElementType {
	switch nodeType {
	case "Package", "Module":
		return ElementPackage
	case "Type":
		return ElementTypeDecl
	case "Function":
		return ElementFunction
	case "Method":
		return ElementMethod
	case "Interface":
		return ElementInterface
	case "Variable":
		return ElementVariable
	case "Block":
		return "" // Don't create elements for blocks
	default:
		return ElementTypeDecl
	}
}

// Gets the name from a node's attributes
func nodeName(node ast.Node) string {
	// Handle package nodes
	if node.Type() == "Package" || node.Type() == "Module" {
		if pkgName, ok := node.Attributes()["package_name"].(string); ok {
			return pkgName
		}
	}

	// Handle other nodes
	if name, ok := node.Attributes()["name"].(string); ok {
		return name
	}

	return "" // Don't generate fallback names
}

// Detects Go-specific patterns in the code
func (a *Analyzer) detectPatterns(analysis *Analysis) error {
	// First detect type references since other detections may need them
	if err := a.detectTypeReferences(analysis); err != nil {
		return err
	}

	// Then detect method receivers and interface implementations
	if err := a.detectMethodReceivers(analysis); err != nil {
		return err
	}

	// Then detect interface embeddings
	if err := a.detectInterfaceEmbeddings(analysis); err != nil {
		return err
	}

	// Then detect interface implementations
	if err := a.detectInterfaceImplementations(analysis); err != nil {
		return err
	}

	// Finally detect composition relationships
	if err := a.detectComposition(analysis); err != nil {
		return err
	}

	return nil
}

// Detects all method receiver relationships
func (a *Analyzer) detectMethodReceivers(analysis *Analysis) error {
	// For each method element
	for _, method := range a.findElementsByType(analysis, ElementMethod) {
		// Get receiver type directly from attributes
		if recv, ok := method.Attributes["receiver_type"].(*goparser.TypeInfo); ok && recv != nil {
			// Find the actual receiver type (handle pointer receivers)
			typeName := recv.Name
			if recv.Kind == "pointer" && recv.ElemType != nil {
				typeName = recv.ElemType.Name
			}
			if recvType := a.findTypeByName(analysis, typeName); recvType != nil {
				// Add method receiver relationship
				rel := &Relationship{
					Type:   RelationMethodReceiver,
					Source: method,
					Target: recvType,
				}
				if !a.hasRelationship(analysis, rel) {
					analysis.Structure.Relationships = append(analysis.Structure.Relationships, rel)
				}
			}
		}
	}
	return nil
}

// Detects all type usage relationships
func (a *Analyzer) detectTypeReferences(analysis *Analysis) error {
	// For each type element
	for _, typ := range a.findElementsByType(analysis, ElementTypeDecl) {
		// Check field types
		if fields, ok := typ.Attributes["fields"].([]map[string]any); ok {
			for _, field := range fields {
				if fieldType, ok := field["type"].(*goparser.TypeInfo); ok {
					// Add reference for the field type
					a.addTypeReference(analysis, typ, fieldType)
				}
			}
		}
	}

	// For each function element
	for _, fn := range a.findElementsByType(analysis, ElementFunction) {
		// Check signature types
		if sig, ok := fn.Attributes["signature"].(map[string]any); ok {
			// Check parameter types
			if params, ok := sig["params"].([]*goparser.TypeInfo); ok {
				for _, param := range params {
					a.addTypeReference(analysis, fn, param)
				}
			}

			// Check return types
			if returns, ok := sig["returns"].([]*goparser.TypeInfo); ok {
				for _, ret := range returns {
					a.addTypeReference(analysis, fn, ret)
				}
			}
		}
	}

	// For each method element
	for _, method := range a.findElementsByType(analysis, ElementMethod) {
		// Check signature types
		if sig, ok := method.Attributes["signature"].(map[string]any); ok {
			// Check receiver type
			if recv, ok := sig["receiver_type"].(*goparser.TypeInfo); ok && recv != nil {
				a.addTypeReference(analysis, method, recv)
			}

			// Check parameter types
			if params, ok := sig["params"].([]*goparser.TypeInfo); ok {
				for _, param := range params {
					a.addTypeReference(analysis, method, param)
				}
			}

			// Check return types
			if returns, ok := sig["returns"].([]*goparser.TypeInfo); ok {
				for _, ret := range returns {
					a.addTypeReference(analysis, method, ret)
				}
			}
		}
	}

	// For each interface element
	for _, iface := range a.findElementsByType(analysis, ElementInterface) {
		// Check method signatures
		if methods, ok := iface.Attributes["methods"].([]map[string]any); ok {
			for _, method := range methods {
				if sig, ok := method["signature"].(map[string]any); ok {
					// Check parameter types
					if params, ok := sig["params"].([]*goparser.TypeInfo); ok {
						for _, param := range params {
							a.addTypeReference(analysis, iface, param)
						}
					}

					// Check return types
					if returns, ok := sig["returns"].([]*goparser.TypeInfo); ok {
						for _, ret := range returns {
							a.addTypeReference(analysis, iface, ret)
						}
					}
				}
			}
		}

		// Check embedded interfaces
		if embedded, ok := iface.Attributes["embedded"].([]map[string]any); ok {
			for _, embed := range embedded {
				if embedType, ok := embed["type"].(*goparser.TypeInfo); ok {
					a.addTypeReference(analysis, iface, embedType)
				}
			}
		}
	}

	return nil
}

// Detects all interface implementations
func (a *Analyzer) detectInterfaceImplementations(analysis *Analysis) error {
	// For each interface element
	for _, iface := range a.findElementsByType(analysis, ElementInterface) {
		// Get interface methods (including embedded)
		ifaceMethods := a.interfaceMethods(iface, analysis)
		if len(ifaceMethods) == 0 {
			continue
		}

		// For each type element
		for _, typ := range a.findElementsByType(analysis, ElementTypeDecl) {
			// Get type methods (including from embedded types)
			typeMethods := a.typeMethods(typ, analysis)

			// Check if type implements interface
			if a.typeImplementsInterface(ifaceMethods, typeMethods) {
				// Add implements relationship
				rel := &Relationship{
					Type:   RelationImplements,
					Source: typ,
					Target: iface,
				}
				if !a.hasRelationship(analysis, rel) {
					analysis.Structure.Relationships = append(analysis.Structure.Relationships, rel)
				}
			}
		}
	}
	return nil
}

// Detects all type embedding relationships
func (a *Analyzer) detectComposition(analysis *Analysis) error {
	// For each type element
	for _, typ := range a.findElementsByType(analysis, ElementTypeDecl) {
		// Get fields
		fields, ok := typ.Attributes["fields"].([]map[string]any)
		if !ok {
			continue
		}

		// Check each field for embedding
		for _, field := range fields {
			embedded, ok := field["embedded"].(bool)
			if !ok || !embedded {
				continue
			}

			// Find the embedded type element
			if fieldType, ok := field["type"].(*goparser.TypeInfo); ok {
				// Handle pointer to embedded type
				typeName := fieldType.Name
				if fieldType.Kind == "pointer" && fieldType.ElemType != nil {
					typeName = fieldType.ElemType.Name
				}
				if embedded := a.findTypeByName(analysis, typeName); embedded != nil {
					// Add composition relationship
					rel := &Relationship{
						Type:   RelationEmbeds,
						Source: typ,
						Target: embedded,
					}
					if !a.hasRelationship(analysis, rel) {
						analysis.Structure.Relationships = append(analysis.Structure.Relationships, rel)
					}
				}
			}
		}
	}
	return nil
}

// Detects all interface embedding relationships
func (a *Analyzer) detectInterfaceEmbeddings(analysis *Analysis) error {
	// For each interface element
	for _, iface := range a.findElementsByType(analysis, ElementInterface) {
		// Check if the interface has embedded interfaces
		if embedded, ok := iface.Attributes["embedded"].([]map[string]any); ok {
			// For each embedded interface
			for _, embed := range embedded {
				// Check if the embedded interface is a type
				if embedType, ok := embed["type"].(*goparser.TypeInfo); ok {
					// Check if the embedded interface is an interface
					if target := a.findTypeByName(analysis, embedType.Name); target != nil && target.Type == ElementInterface {
						// Add interface embedding relationship
						rel := &Relationship{
							Type:   RelationInterfaceEmbeds,
							Source: iface,
							Target: target,
						}
						if !a.hasRelationship(analysis, rel) {
							analysis.Structure.Relationships = append(analysis.Structure.Relationships, rel)
						}
					}
				}
			}
		}
	}
	return nil
}

// Finds elements by type
func (a *Analyzer) findElementsByType(analysis *Analysis, elemType ElementType) []*Element {
	var result []*Element
	for _, elem := range analysis.Structure.Elements {
		if elem.Type == elemType {
			result = append(result, elem)
		}
	}
	return result
}

// Finds a type element by name
func (a *Analyzer) findTypeByName(analysis *Analysis, name string) *Element {
	for _, elem := range analysis.Structure.Elements {
		if (elem.Type == ElementTypeDecl || elem.Type == ElementInterface) && elem.Name == name {
			return elem
		}
	}
	return nil
}

// Returns all methods of an interface (including embedded)
func (a *Analyzer) interfaceMethods(iface *Element, analysis *Analysis) []map[string]any {
	var methods []map[string]any

	// Get direct methods
	if methodList, ok := iface.Attributes["methods"].([]map[string]any); ok {
		for _, method := range methodList {
			// Skip embedded interfaces (they have no name)
			if name, ok := method["name"].(string); ok && name != "" {
				methods = append(methods, method)
			} else {
				// For embedded interfaces, get their methods
				if sig, ok := method["signature"].(map[string]any); ok {
					if params, ok := sig["params"].([]*goparser.TypeInfo); ok && len(params) > 0 {
						// The first param is the embedded interface type
						embedType := params[0]
						if embedded := a.findTypeByName(analysis, embedType.Name); embedded != nil && embedded.Type == ElementInterface {
							methods = append(methods, a.interfaceMethods(embedded, analysis)...)
						}
					}
				}
			}
		}
	}

	return methods
}

// Returns all methods of a type (including from embedded types)
func (a *Analyzer) typeMethods(typ *Element, analysis *Analysis) []map[string]any {
	var methods []map[string]any

	// Get direct methods
	for _, method := range a.findElementsByType(analysis, ElementMethod) {
		// Get receiver type directly from attributes
		if recv, ok := method.Attributes["receiver_type"].(*goparser.TypeInfo); ok && recv != nil {
			// Check if method belongs to this type
			typeName := recv.Name
			if recv.Kind == "pointer" && recv.ElemType != nil {
				typeName = recv.ElemType.Name
			}
			if typeName == typ.Name {
				if sig, ok := method.Attributes["signature"].(map[string]any); ok {
					methods = append(methods, map[string]any{
						"name":               method.Name,
						"signature":          sig,
						"receiver_type_name": typeName,
					})
				}
			}
		}
	}

	// Get methods from embedded types
	if fields, ok := typ.Attributes["fields"].([]map[string]any); ok {
		for _, field := range fields {
			if embedded, ok := field["embedded"].(bool); ok && embedded {
				if fieldType, ok := field["type"].(*goparser.TypeInfo); ok {
					// Handle pointer to embedded type
					typeName := fieldType.Name
					if fieldType.Kind == "pointer" && fieldType.ElemType != nil {
						typeName = fieldType.ElemType.Name
					}
					if embedded := a.findTypeByName(analysis, typeName); embedded != nil {
						embeddedMethods := a.typeMethods(embedded, analysis)
						// Add embedded type name to each method
						for _, method := range embeddedMethods {
							method["receiver_type_name"] = typeName
						}
						methods = append(methods, embeddedMethods...)
					}
				}
			}
		}
	}

	return methods
}

// Checks if a type implements an interface
func (a *Analyzer) typeImplementsInterface(ifaceMethods, typeMethods []map[string]any) bool {
	if len(typeMethods) == 0 {
		return false
	}

	// For each interface method
	for _, imethod := range ifaceMethods {
		found := false
		// Look for matching method
		for _, tmethod := range typeMethods {
			if tmethod["name"] == imethod["name"] &&
				a.signatureMatches(tmethod["signature"].(map[string]any), imethod["signature"].(map[string]any)) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// Checks if two method signatures match
func (a *Analyzer) signatureMatches(sig1, sig2 map[string]any) bool {
	// Compare receiver types if present
	if recv1, ok1 := sig1["receiver_type"].(*goparser.TypeInfo); ok1 {
		if recv2, ok2 := sig2["receiver_type"].(*goparser.TypeInfo); ok2 {
			if !a.typeMatches(recv1, recv2) {
				return false
			}
		} else if ok1 != ok2 {
			return false
		}
	}

	// Compare parameter types
	params1, ok1 := sig1["params"].([]*goparser.TypeInfo)
	params2, ok2 := sig2["params"].([]*goparser.TypeInfo)
	if !ok1 || !ok2 || !a.typeListMatches(params1, params2) {
		return false
	}

	// Compare return types
	returns1, ok1 := sig1["returns"].([]*goparser.TypeInfo)
	returns2, ok2 := sig2["returns"].([]*goparser.TypeInfo)
	if !ok1 || !ok2 || !a.typeListMatches(returns1, returns2) {
		return false
	}

	return true
}

// Checks if two type lists match
func (a *Analyzer) typeListMatches(types1, types2 []*goparser.TypeInfo) bool {
	if len(types1) != len(types2) {
		return false
	}
	for i := range types1 {
		if !a.typeMatches(types1[i], types2[i]) {
			return false
		}
	}
	return true
}

// Checks if two types match
func (a *Analyzer) typeMatches(t1, t2 *goparser.TypeInfo) bool {
	if t1 == nil || t2 == nil {
		return t1 == t2
	}

	// Check kind and name
	if t1.Kind != t2.Kind || t1.Name != t2.Name {
		return false
	}

	// For pointer types, check element type
	if t1.Kind == "pointer" {
		return a.typeMatches(t1.ElemType, t2.ElemType)
	}

	// For slice types, check element type
	if t1.Kind == "slice" {
		return a.typeMatches(t1.ElemType, t2.ElemType)
	}

	// For map types, check key and value types
	if t1.Kind == "map" {
		return a.typeMatches(t1.KeyType, t2.KeyType) &&
			a.typeMatches(t1.ValueType, t2.ValueType)
	}

	return true
}

func (a *Analyzer) addTypeReference(analysis *Analysis, source *Element, typeInfo *goparser.TypeInfo) {
	if typeInfo == nil {
		return
	}

	// For pointer types, reference the element type
	if typeInfo.Kind == "pointer" && typeInfo.ElemType != nil {
		a.addTypeReference(analysis, source, typeInfo.ElemType)
		return
	}

	// For slice types, reference the element type
	if typeInfo.Kind == "slice" && typeInfo.ElemType != nil {
		a.addTypeReference(analysis, source, typeInfo.ElemType)
		return
	}

	// For map types, reference both key and value types
	if typeInfo.Kind == "map" {
		if typeInfo.KeyType != nil {
			a.addTypeReference(analysis, source, typeInfo.KeyType)
		}
		if typeInfo.ValueType != nil {
			a.addTypeReference(analysis, source, typeInfo.ValueType)
		}
		return
	}

	// Skip actual primitive types
	if typeInfo.Kind == "basic" && typeInfo.Name != "" {
		// Check if it's a primitive type
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

		// Not a primitive - try to find the named type
		if target := a.findTypeByName(analysis, typeInfo.Name); target != nil {
			rel := &Relationship{
				Type:   RelationReferences,
				Source: source,
				Target: target,
			}
			if !a.hasRelationship(analysis, rel) {
				analysis.Structure.Relationships = append(analysis.Structure.Relationships, rel)
			}
		} else {
			// Try to find interface
			for _, iface := range a.findElementsByType(analysis, ElementInterface) {
				if iface.Name == typeInfo.Name {
					rel := &Relationship{
						Type:   RelationReferences,
						Source: source,
						Target: iface,
					}
					if !a.hasRelationship(analysis, rel) {
						analysis.Structure.Relationships = append(analysis.Structure.Relationships, rel)
					}
					return
				}
			}
		}
	}
}

// Merges two analyses
func (a *Analyzer) Merge(base, other *Analysis) error {
	if base == nil || other == nil {
		return fmt.Errorf("cannot merge nil analyses")
	}

	// Verify both analyses are Go analyses
	if base.Language() != "go" || other.Language() != "go" {
		return fmt.Errorf("can only merge Go analyses")
	}

	existingPackages := make(map[string]*Element)
	for _, elem := range base.Structure.Elements {
		if elem.Type == ElementPackage {
			existingPackages[elem.Name] = elem
		}
	}

	for _, elem := range other.Structure.Elements {
		if elem.Type == ElementPackage {
			if _, exists := existingPackages[elem.Name]; exists {
				continue
			}
		}
		base.Structure.Elements = append(base.Structure.Elements, elem)
	}

	// Merge relationships
	base.Structure.Relationships = append(base.Structure.Relationships, other.Structure.Relationships...)

	return nil
}
