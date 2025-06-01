// Provides the Go language parser implementation
package goparser

import (
	goast "go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"

	"codedna/internal/core/parser/ast"
)

// TypeInfo represents a type in a structural way
type TypeInfo struct {
	Kind        string      // The kind of type (e.g. "basic", "pointer", "array", "map", "chan", "interface", "generic")
	Name        string      // The name of the type (e.g. "int", "string", "MyStruct")
	ElemType    *TypeInfo   // For pointer, array, chan types
	KeyType     *TypeInfo   // For map types
	ValueType   *TypeInfo   // For map types
	TypeParams  []*TypeInfo // For generic types - list of type parameters
	TypeArgs    []*TypeInfo // For generic type instantiation - list of type arguments
	Constraints []*TypeInfo // For generic type parameters - list of constraints
	IsTypeParam bool        // Whether this type is a type parameter
}

// Implements the parser.Parser interface for Go
type Parser struct {
	fset *token.FileSet
	info *types.Info
	conf types.Config
}

// Creates a new Go parser
func New() *Parser {
	return &Parser{
		fset: token.NewFileSet(),
		info: &types.Info{
			Types: make(map[goast.Expr]types.TypeAndValue),
			Defs:  make(map[*goast.Ident]types.Object),
			Uses:  make(map[*goast.Ident]types.Object),
		},
		conf: types.Config{
			Importer: nil,                // We don't need imports for type checking
			Error:    func(err error) {}, // Ignore type checking errors
		},
	}
}

func (p *Parser) Language() string {
	return "Go"
}

func (p *Parser) FileExtensions() []string {
	return []string{".go"}
}

func (p *Parser) ParseFile(filename string) (ast.Node, error) {
	file, err := parser.ParseFile(p.fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// Type check the file
	pkg := types.NewPackage(file.Name.Name, "")
	files := []*goast.File{file}
	if err := types.NewChecker(&p.conf, p.fset, pkg, p.info).Files(files); err != nil {
		// Intentionally ignoring type errors:
		// - Type checking is best-effort for enhanced type information
		// - Parsing should succeed even with type errors
		// - Common with incomplete/partial files or missing dependencies
		_ = err
	}

	return p.convertFile(file), nil
}

func (p *Parser) ParseDir(dir string) ([]ast.Node, error) {
	pkgs, err := parser.ParseDir(p.fset, dir, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var nodes []ast.Node
	for _, pkg := range pkgs {
		// Type check all files in the package together
		files := make([]*goast.File, 0, len(pkg.Files))
		for _, file := range pkg.Files {
			files = append(files, file)
		}

		// Create a new package and type checker
		typePkg := types.NewPackage(pkg.Name, "")
		if err := types.NewChecker(&p.conf, p.fset, typePkg, p.info).Files(files); err != nil {
			// Intentionally ignoring type errors:
			// - Type checking is best-effort for enhanced type information
			// - Parsing should succeed even with type errors
			// - Common with incomplete/partial files or missing dependencies
			_ = err
		}

		// Convert each file to our AST
		for _, file := range pkg.Files {
			nodes = append(nodes, p.convertFile(file))
		}
	}
	return nodes, nil
}

// Converts Go AST file to our generic AST
func (p *Parser) convertFile(file *goast.File) ast.Node {
	pos := p.fset.Position(file.Pos())
	node := ast.NewBaseNode(ast.Module, ast.Position{
		Line:   pos.Line,
		Column: pos.Column,
		Offset: pos.Offset,
	})

	// Add package name and file path
	node.SetAttribute("package_name", file.Name.Name)
	node.SetAttribute("file_path", pos.Filename)

	// Track dependencies
	dependencies := make([]string, 0)

	// First pass: collect all type declarations and their methods
	typeNodes := make(map[string]*ast.BaseNode)
	methodsByType := make(map[string][]map[string]any)

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *goast.FuncDecl:
			if d.Recv != nil && len(d.Recv.List) > 0 {
				// This is a method
				recv := d.Recv.List[0]
				var typeName string
				switch rt := recv.Type.(type) {
				case *goast.StarExpr:
					// Pointer receiver
					if ident, ok := rt.X.(*goast.Ident); ok {
						typeName = ident.Name
					}
				case *goast.Ident:
					// Value receiver
					typeName = rt.Name
				}
				if typeName != "" {
					// Build method info
					methodInfo := map[string]any{
						"name": d.Name.Name,
						"signature": map[string]any{
							"params":  typeList(d.Type.Params),
							"returns": typeList(d.Type.Results),
						},
					}
					methodsByType[typeName] = append(methodsByType[typeName], methodInfo)
				}
			}
		case *goast.GenDecl:
			if d.Tok == token.TYPE {
				for _, spec := range d.Specs {
					if typeSpec, ok := spec.(*goast.TypeSpec); ok {
						if typeNode, ok := p.createTypeNode(typeSpec).(*ast.BaseNode); ok {
							if name, ok := typeNode.Attributes()["name"].(string); ok {
								typeNodes[name] = typeNode
							}
						}
					}
				}
			}
		}
	}

	// Add methods to type nodes
	for typeName, methods := range methodsByType {
		if typeNode, ok := typeNodes[typeName]; ok {
			typeNode.SetAttribute("methods", methods)
		}
	}

	// Second pass: add all declarations to the module node
	for _, decl := range file.Decls {
		if declNode := p.convertDecl(decl); declNode != nil {
			// If this is a type node, replace it with our annotated version
			if declNode.Type() == string(ast.Type) || declNode.Type() == string(ast.Interface) {
				if name, ok := declNode.Attributes()["name"].(string); ok {
					if annotatedNode, ok := typeNodes[name]; ok {
						declNode = annotatedNode
					}
				}
			}
			node.AddChild(declNode)
		}
	}

	for _, imp := range file.Imports {
		importNode := p.convertImport(imp)
		node.AddChild(importNode)
		if path, ok := importNode.Attributes()["path"]; ok {
			dependencies = append(dependencies, path.(string))
		}
	}

	node.SetAttribute("dependencies", dependencies)

	return node
}

// Converts Go import to our generic AST
func (p *Parser) convertImport(imp *goast.ImportSpec) ast.Node {
	pos := p.fset.Position(imp.Pos())
	node := ast.NewBaseNode(ast.Import, ast.Position{
		Line:   pos.Line,
		Column: pos.Column,
		Offset: pos.Offset,
	})

	node.SetAttribute("file_path", pos.Filename)

	// Store import path without quotes
	if imp.Path != nil {
		path := imp.Path.Value[1 : len(imp.Path.Value)-1] // Remove quotes
		node.SetAttribute("path", path)
		// Check if it's a standard library import
		node.SetAttribute("is_std_lib", !containsPath(path))

		// Store alias if present
		if imp.Name != nil {
			node.SetAttribute("alias", imp.Name.Name)
		}
	}

	return node
}

// Helper function to check if import path contains a path separator
func containsPath(path string) bool {
	return strings.Contains(path, "/") || strings.Contains(path, "\\")
}

// Converts Go declaration to our generic AST
func (p *Parser) convertDecl(decl goast.Decl) ast.Node {
	switch d := decl.(type) {
	case *goast.FuncDecl:
		return p.convertFunction(d)
	case *goast.GenDecl:
		return p.convertGenDecl(d)
	}
	return nil
}

// Converts Go function to our generic AST
func (p *Parser) convertFunction(fn *goast.FuncDecl) ast.Node {
	pos := p.fset.Position(fn.Pos())
	nodeType := ast.Function
	if fn.Recv != nil {
		nodeType = ast.Method
	}

	node := ast.NewBaseNode(nodeType, ast.Position{
		Line:   pos.Line,
		Column: pos.Column,
		Offset: pos.Offset,
	})

	// Store function name and export status
	node.SetAttribute("name", fn.Name.Name)
	node.SetAttribute("is_exported", fn.Name.IsExported())
	node.SetAttribute("file_path", pos.Filename)

	// Handle type parameters if present
	if fn.Type.TypeParams != nil {
		typeParams := make([]*TypeInfo, 0, len(fn.Type.TypeParams.List))
		for _, field := range fn.Type.TypeParams.List {
			for _, name := range field.Names {
				paramInfo := &TypeInfo{
					Kind:        "type_param",
					Name:        name.Name,
					IsTypeParam: true,
				}
				// Handle constraints
				if field.Type != nil {
					switch constraint := field.Type.(type) {
					case *goast.Ident:
						// Basic constraint like "any" or "comparable"
						paramInfo.Constraints = []*TypeInfo{{
							Kind: "constraint",
							Name: constraint.Name,
						}}
					case *goast.InterfaceType:
						// Interface constraint
						paramInfo.Constraints = []*TypeInfo{{
							Kind: "interface",
							Name: "interface{}",
						}}
					case *goast.UnaryExpr:
						// Tilde (~) expressions for type constraints
						if constraint.Op == token.TILDE {
							paramInfo.Constraints = []*TypeInfo{{
								Kind: "constraint",
								Name: "~" + typeToTypeInfo(constraint.X).Name,
							}}
						}
					case *goast.BinaryExpr:
						// Union type constraints (|)
						if constraint.Op == token.OR {
							paramInfo.Constraints = []*TypeInfo{
								typeToTypeInfo(constraint.X),
								typeToTypeInfo(constraint.Y),
							}
						}
					}
				}
				typeParams = append(typeParams, paramInfo)
			}
		}
		node.SetAttribute("type_params", typeParams)
	}

	// Build function signature
	params := make([]*TypeInfo, 0)
	if fn.Type.Params != nil {
		for _, param := range fn.Type.Params.List {
			paramType := typeToTypeInfo(param.Type)
			for range param.Names {
				params = append(params, paramType)
			}
		}
	}

	returns := make([]*TypeInfo, 0)
	if fn.Type.Results != nil {
		for _, result := range fn.Type.Results.List {
			resultType := typeToTypeInfo(result.Type)
			if len(result.Names) == 0 {
				returns = append(returns, resultType)
			} else {
				for range result.Names {
					returns = append(returns, resultType)
				}
			}
		}
	}

	signature := map[string]any{
		"params":  params,
		"returns": returns,
	}
	node.SetAttribute("signature", signature)

	// Store receiver information for methods
	if fn.Recv != nil {
		for _, recv := range fn.Recv.List {
			recvType := typeToTypeInfo(recv.Type)
			node.SetAttribute("receiver_type", recvType)
			break
		}
	}

	// Process function body for references
	if fn.Body != nil {
		body := make([]map[string]any, 0)
		for _, stmt := range fn.Body.List {
			stmtInfo := p.processStatement(stmt)
			if stmtInfo != nil {
				body = append(body, stmtInfo)
			}
		}
		node.SetAttribute("body", body)
	}

	return node
}

// Helper function to process a statement and extract references
func (p *Parser) processStatement(stmt goast.Stmt) map[string]any {
	refs := make([]map[string]any, 0)

	// Helper function to process an expression
	var processExpr func(expr goast.Expr)
	processExpr = func(expr goast.Expr) {
		if expr == nil {
			return
		}

		switch e := expr.(type) {
		case *goast.CallExpr:
			// Handle function calls
			switch fun := e.Fun.(type) {
			case *goast.SelectorExpr:
				if pkg, ok := fun.X.(*goast.Ident); ok {
					// Check if it's a package selector
					if obj := p.info.Uses[pkg]; obj != nil && obj.Pkg() != nil {
						ref := map[string]any{
							"type": &TypeInfo{
								Kind: "package",
								Name: pkg.Name,
							},
						}
						refs = append(refs, ref)
					} else {
						ref := map[string]any{
							"type": &TypeInfo{
								Kind: "basic",
								Name: pkg.Name + "." + fun.Sel.Name,
							},
						}
						refs = append(refs, ref)
					}
				}
			case *goast.Ident:
				// Handle direct function calls
				if obj := p.info.Uses[fun]; obj != nil {
					if pkg := obj.Pkg(); pkg != nil {
						ref := map[string]any{
							"type": &TypeInfo{
								Kind: "basic",
								Name: pkg.Name() + "." + fun.Name,
							},
						}
						refs = append(refs, ref)
					}
				}
			}
			// Process arguments
			for _, arg := range e.Args {
				processExpr(arg)
			}

		case *goast.SelectorExpr:
			// Handle field/method access
			if x, ok := e.X.(*goast.Ident); ok {
				// Check if it's a package selector
				if obj := p.info.Uses[x]; obj != nil && obj.Pkg() != nil {
					refs = append(refs, map[string]any{
						"type": &TypeInfo{
							Kind: "package",
							Name: x.Name,
						},
					})
				} else {
					refs = append(refs, map[string]any{
						"type": &TypeInfo{
							Kind: "basic",
							Name: x.Name + "." + e.Sel.Name,
						},
					})
				}
			}

		case *goast.CompositeLit:
			// Handle composite literals
			if e.Type != nil {
				refs = append(refs, map[string]any{
					"type": typeToTypeInfo(e.Type),
				})
			}
			for _, elt := range e.Elts {
				processExpr(elt)
			}

		case *goast.UnaryExpr:
			processExpr(e.X)

		case *goast.BinaryExpr:
			processExpr(e.X)
			processExpr(e.Y)

		case *goast.KeyValueExpr:
			processExpr(e.Key)
			processExpr(e.Value)
		}
	}

	// Process the statement based on its type
	switch s := stmt.(type) {
	case *goast.ReturnStmt:
		for _, result := range s.Results {
			processExpr(result)
		}

	case *goast.AssignStmt:
		for _, rhs := range s.Rhs {
			processExpr(rhs)
		}

	case *goast.DeclStmt:
		if decl, ok := s.Decl.(*goast.GenDecl); ok {
			for _, spec := range decl.Specs {
				if vs, ok := spec.(*goast.ValueSpec); ok {
					for _, val := range vs.Values {
						processExpr(val)
					}
				}
			}
		}

	case *goast.ExprStmt:
		processExpr(s.X)
	}

	if len(refs) > 0 {
		return map[string]any{
			"references": refs,
		}
	}
	return nil
}

// Helper function to convert Go AST type to TypeInfo
func typeToTypeInfo(expr goast.Expr) *TypeInfo {
	switch t := expr.(type) {
	case *goast.Ident:
		kind := "basic"
		name := t.Name
		// Convert float64 to float for language-agnostic representation
		if name == "float64" {
			name = "float"
		}
		return &TypeInfo{Kind: kind, Name: name}
	case *goast.StarExpr:
		return &TypeInfo{
			Kind:     "pointer",
			ElemType: typeToTypeInfo(t.X),
		}
	case *goast.ArrayType:
		if t.Len == nil {
			return &TypeInfo{
				Kind:     "slice",
				ElemType: typeToTypeInfo(t.Elt),
			}
		}
		return &TypeInfo{
			Kind:     "array",
			ElemType: typeToTypeInfo(t.Elt),
		}
	case *goast.MapType:
		return &TypeInfo{
			Kind:      "map",
			KeyType:   typeToTypeInfo(t.Key),
			ValueType: typeToTypeInfo(t.Value),
		}
	case *goast.InterfaceType:
		return &TypeInfo{Kind: "interface", Name: "interface{}"}
	case *goast.SelectorExpr:
		if x, ok := t.X.(*goast.Ident); ok {
			return &TypeInfo{Kind: "basic", Name: x.Name + "." + t.Sel.Name}
		}
	case *goast.ChanType:
		return &TypeInfo{
			Kind:     "chan",
			ElemType: typeToTypeInfo(t.Value),
		}
	case *goast.IndexExpr: // Single type parameter
		baseType := typeToTypeInfo(t.X)
		baseType.Kind = "generic"
		baseType.TypeArgs = []*TypeInfo{typeToTypeInfo(t.Index)}
		return baseType
	case *goast.IndexListExpr: // Multiple type parameters
		baseType := typeToTypeInfo(t.X)
		baseType.Kind = "generic"
		baseType.TypeArgs = make([]*TypeInfo, len(t.Indices))
		for i, arg := range t.Indices {
			baseType.TypeArgs[i] = typeToTypeInfo(arg)
		}
		return baseType
	}
	return &TypeInfo{Kind: "unknown"}
}

// Helper function to convert Go type to TypeInfo
func typeFromGoType(t types.Type) *TypeInfo {
	if t == nil {
		return &TypeInfo{Kind: "unknown"}
	}

	switch typ := t.(type) {
	case *types.Basic:
		name := typ.Name()
		// Convert float64 to float for language-agnostic representation
		if name == "float64" {
			name = "float"
		}
		return &TypeInfo{Kind: "basic", Name: name}
	case *types.Pointer:
		return &TypeInfo{
			Kind:     "pointer",
			ElemType: typeFromGoType(typ.Elem()),
		}
	case *types.Slice:
		return &TypeInfo{
			Kind:     "slice",
			ElemType: typeFromGoType(typ.Elem()),
		}
	case *types.Array:
		return &TypeInfo{
			Kind:     "array",
			ElemType: typeFromGoType(typ.Elem()),
		}
	case *types.Map:
		return &TypeInfo{
			Kind:      "map",
			KeyType:   typeFromGoType(typ.Key()),
			ValueType: typeFromGoType(typ.Elem()),
		}
	case *types.Chan:
		return &TypeInfo{
			Kind:     "chan",
			ElemType: typeFromGoType(typ.Elem()),
		}
	case *types.Interface:
		return &TypeInfo{Kind: "interface", Name: "interface{}"}
	case *types.Named:
		return &TypeInfo{Kind: "basic", Name: typ.Obj().Name()}
	default:
		return &TypeInfo{Kind: "unknown"}
	}
}

// Helper function to infer type from an expression
func (p *Parser) inferTypeFromExpr(expr goast.Expr) *TypeInfo {
	// First try to get the type from the type checker
	if tv, ok := p.info.Types[expr]; ok {
		return typeFromGoType(tv.Type)
	}

	// If type checker info is not available, fall back to AST-based inference
	switch e := expr.(type) {
	case *goast.BasicLit:
		switch e.Kind {
		case token.INT:
			return &TypeInfo{Kind: "basic", Name: "int"}
		case token.FLOAT:
			return &TypeInfo{Kind: "basic", Name: "float"}
		case token.STRING:
			return &TypeInfo{Kind: "basic", Name: "string"}
		case token.CHAR:
			return &TypeInfo{Kind: "basic", Name: "rune"}
		}
	case *goast.Ident:
		if obj := p.info.Uses[e]; obj != nil {
			if t := obj.Type(); t != nil {
				return typeFromGoType(t)
			}
		}
		// Handle boolean literals
		if e.Name == "true" || e.Name == "false" {
			return &TypeInfo{Kind: "basic", Name: "bool"}
		}
	case *goast.CompositeLit:
		if e.Type != nil {
			return typeToTypeInfo(e.Type)
		}
	case *goast.CallExpr:
		if fun, ok := e.Fun.(*goast.Ident); ok && fun.Name == "make" && len(e.Args) > 0 {
			return typeToTypeInfo(e.Args[0])
		}
	}
	return &TypeInfo{Kind: "unknown"}
}

// Create a node for each name in the ValueSpec
func (p *Parser) createValueNode(spec *goast.ValueSpec, i int) ast.Node {
	name := spec.Names[i]
	pos := p.fset.Position(name.Pos())

	node := ast.NewBaseNode(ast.Variable, ast.Position{
		Line:   pos.Line,
		Column: pos.Column,
		Offset: pos.Offset,
	})

	node.SetAttribute("name", name.Name)
	node.SetAttribute("is_exported", name.IsExported())
	node.SetAttribute("file_path", pos.Filename)

	var typeInfo *TypeInfo

	// Try to get type from type checker first
	if obj := p.info.Defs[name]; obj != nil {
		if typ := obj.Type(); typ != nil {
			typeInfo = typeFromGoType(typ)
		}
	}

	// If we have values, try to get type from the value expression
	if typeInfo == nil && i < len(spec.Values) {
		if typeAndValue, ok := p.info.Types[spec.Values[i]]; ok {
			typeInfo = typeFromGoType(typeAndValue.Type)
		}
	}

	// Fallback to AST-based type inference
	if typeInfo == nil {
		if spec.Type != nil {
			typeInfo = typeToTypeInfo(spec.Type)
		} else if i < len(spec.Values) {
			typeInfo = p.inferTypeFromExpr(spec.Values[i])
		}
	}

	node.SetAttribute("type", typeInfo)
	return node
}

// Converts Go generic declaration to our generic AST
func (p *Parser) convertGenDecl(decl *goast.GenDecl) ast.Node {
	switch decl.Tok {
	case token.TYPE:
		// For single declarations
		if len(decl.Specs) == 1 && !decl.Lparen.IsValid() {
			if spec, ok := decl.Specs[0].(*goast.TypeSpec); ok {
				return p.createTypeNode(spec)
			}
			return nil
		}

		// For grouped declarations
		if len(decl.Specs) > 0 {
			pos := p.fset.Position(decl.Pos())
			groupNode := ast.NewBaseNode(ast.Block, ast.Position{
				Line:   pos.Line,
				Column: pos.Column,
				Offset: pos.Offset,
			})
			groupNode.SetAttribute("file_path", pos.Filename)

			for _, spec := range decl.Specs {
				if typeSpec, ok := spec.(*goast.TypeSpec); ok {
					groupNode.AddChild(p.createTypeNode(typeSpec))
				}
			}
			return groupNode
		}

	case token.VAR, token.CONST:
		// For single declarations
		if len(decl.Specs) == 1 && !decl.Lparen.IsValid() {
			if spec, ok := decl.Specs[0].(*goast.ValueSpec); ok {
				if len(spec.Names) == 1 {
					return p.createValueNode(spec, 0)
				} else if len(spec.Names) > 1 {
					pos := p.fset.Position(decl.Pos())
					groupNode := ast.NewBaseNode(ast.Block, ast.Position{
						Line:   pos.Line,
						Column: pos.Column,
						Offset: pos.Offset,
					})
					groupNode.SetAttribute("file_path", pos.Filename)
					for i := range spec.Names {
						groupNode.AddChild(p.createValueNode(spec, i))
					}
					return groupNode
				}
			}
			return nil
		}

		// For grouped declarations
		if len(decl.Specs) > 0 {
			pos := p.fset.Position(decl.Pos())
			groupNode := ast.NewBaseNode(ast.Block, ast.Position{
				Line:   pos.Line,
				Column: pos.Column,
				Offset: pos.Offset,
			})
			groupNode.SetAttribute("file_path", pos.Filename)

			for _, spec := range decl.Specs {
				if valueSpec, ok := spec.(*goast.ValueSpec); ok {
					for i := range valueSpec.Names {
						groupNode.AddChild(p.createValueNode(valueSpec, i))
					}
				}
			}
			return groupNode
		}
	}

	return nil
}

// Helper function to extract a list of types from a FieldList
func typeList(fields *goast.FieldList) []*TypeInfo {
	types := make([]*TypeInfo, 0)
	if fields != nil {
		for _, field := range fields.List {
			fieldType := typeToTypeInfo(field.Type)
			if len(field.Names) == 0 {
				types = append(types, fieldType)
			} else {
				for range field.Names {
					types = append(types, fieldType)
				}
			}
		}
	}
	return types
}

// Create a node for a type declaration
func (p *Parser) createTypeNode(spec *goast.TypeSpec) ast.Node {
	specPos := p.fset.Position(spec.Pos())

	nodeType := ast.Type
	if _, isInterface := spec.Type.(*goast.InterfaceType); isInterface {
		nodeType = ast.Interface
	}

	node := ast.NewBaseNode(nodeType, ast.Position{
		Line:   specPos.Line,
		Column: specPos.Column,
		Offset: specPos.Offset,
	})

	node.SetAttribute("name", spec.Name.Name)
	node.SetAttribute("is_exported", spec.Name.IsExported())
	node.SetAttribute("file_path", specPos.Filename)

	// Handle type parameters if present
	if spec.TypeParams != nil {
		typeParams := make([]*TypeInfo, 0, len(spec.TypeParams.List))
		for _, field := range spec.TypeParams.List {
			for _, name := range field.Names {
				paramInfo := &TypeInfo{
					Kind:        "type_param",
					Name:        name.Name,
					IsTypeParam: true,
				}
				// Handle constraints
				if field.Type != nil {
					switch constraint := field.Type.(type) {
					case *goast.Ident:
						// Basic constraint like "any" or "comparable"
						paramInfo.Constraints = []*TypeInfo{{
							Kind: "constraint",
							Name: constraint.Name,
						}}
					case *goast.InterfaceType:
						// Interface constraint
						paramInfo.Constraints = []*TypeInfo{{
							Kind: "interface",
							Name: "interface{}",
						}}
					case *goast.UnaryExpr:
						// Tilde (~) expressions for type constraints
						if constraint.Op == token.TILDE {
							paramInfo.Constraints = []*TypeInfo{{
								Kind: "constraint",
								Name: "~" + typeToTypeInfo(constraint.X).Name,
							}}
						}
					case *goast.BinaryExpr:
						// Union type constraints (|)
						if constraint.Op == token.OR {
							paramInfo.Constraints = []*TypeInfo{
								typeToTypeInfo(constraint.X),
								typeToTypeInfo(constraint.Y),
							}
						}
					}
				}
				typeParams = append(typeParams, paramInfo)
			}
		}
		node.SetAttribute("type_params", typeParams)
	}

	switch t := spec.Type.(type) {
	case *goast.InterfaceType:
		methods := make([]map[string]any, 0)
		embedded := make([]map[string]any, 0)
		if t.Methods != nil {
			for _, method := range t.Methods.List {
				switch methodType := method.Type.(type) {
				case *goast.FuncType:
					// Regular method
					for _, name := range method.Names {
						methodInfo := map[string]any{
							"name": name.Name,
							"signature": map[string]any{
								"params":  typeList(methodType.Params),
								"returns": typeList(methodType.Results),
							},
						}
						methods = append(methods, methodInfo)
					}
				case *goast.Ident:
					// Embedded interface
					if obj := p.info.Uses[methodType]; obj != nil {
						if named, ok := obj.Type().(*types.Named); ok {
							if _, isInterface := named.Underlying().(*types.Interface); isInterface {
								embedded = append(embedded, map[string]any{
									"type": &TypeInfo{
										Kind: "basic",
										Name: named.Obj().Name(),
									},
								})
							}
						}
					} else {
						// Fallback to AST name if type info not available
						embedded = append(embedded, map[string]any{
							"type": &TypeInfo{
								Kind: "basic",
								Name: methodType.Name,
							},
						})
					}
				}
			}
		}
		node.SetAttribute("methods", methods)
		node.SetAttribute("embedded", embedded)

	case *goast.StructType:
		fields := make([]map[string]any, 0)
		if t.Fields != nil {
			for _, field := range t.Fields.List {
				fieldType := typeToTypeInfo(field.Type)
				if len(field.Names) == 0 {
					// Embedded field
					fields = append(fields, map[string]any{
						"name":     fieldType.Name,
						"type":     fieldType,
						"embedded": true,
					})
				} else {
					for _, name := range field.Names {
						fields = append(fields, map[string]any{
							"name":     name.Name,
							"type":     fieldType,
							"embedded": false,
						})
					}
				}
			}
		}
		node.SetAttribute("fields", fields)
		node.SetAttribute("underlying_type", "struct")

	default:
		node.SetAttribute("underlying_type", typeToTypeInfo(spec.Type))
	}

	return node
}
