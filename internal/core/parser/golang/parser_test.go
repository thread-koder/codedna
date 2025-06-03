package goparser_test

import (
	"testing"

	"codedna/internal/core/parser/ast"
	goparser "codedna/internal/core/parser/golang"
)

// Helper function to find nodes of a specific type
func findNodes(root ast.Node, nodeType ast.NodeType) []ast.Node {
	var nodes []ast.Node
	var walk func(ast.Node)
	walk = func(n ast.Node) {
		if n.Type() == string(nodeType) {
			nodes = append(nodes, n)
		}
		for _, child := range n.Children() {
			walk(child)
		}
	}
	walk(root)
	return nodes
}

func TestBasicInfo(t *testing.T) {
	p := goparser.New()

	t.Run("Language", func(t *testing.T) {
		if lang := p.Language(); lang != "Go" {
			t.Errorf("Expected language 'Go', got '%s'", lang)
		}
	})

	t.Run("FileExtensions", func(t *testing.T) {
		exts := p.FileExtensions()
		if len(exts) != 1 || exts[0] != ".go" {
			t.Errorf("Expected ['.go'], got %v", exts)
		}
	})
}

func TestNodeCounts(t *testing.T) {
	p := goparser.New()
	node, err := p.ParseFile("testdata/sample.go")
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Verify root node type
	t.Run("RootNodeType", func(t *testing.T) {
		if node.Type() != string(ast.Module) {
			t.Errorf("Expected root node type %s, got %s", ast.Module, node.Type())
		}
	})

	// Count and verify all node types
	t.Run("NodeCounts", func(t *testing.T) {
		var (
			imports    int
			methods    int
			functions  int
			variables  int
			types      int
			interfaces int
		)

		// Count nodes recursively
		var countNodes func(ast.Node)
		countNodes = func(n ast.Node) {
			switch n.Type() {
			case string(ast.Import):
				imports++
			case string(ast.Method):
				methods++
			case string(ast.Function):
				functions++
			case string(ast.Variable):
				variables++
			case string(ast.Type):
				types++
			case string(ast.Interface):
				interfaces++
			}

			for _, child := range n.Children() {
				countNodes(child)
			}
		}

		countNodes(node)

		expectations := []struct {
			name     string
			got      int
			expected int
			desc     string
		}{
			{"imports", imports, 2, "fmt and strings imports"},
			{"methods", methods, 0, "no methods"},
			{"functions", functions, 2, "ProcessString and main functions"},
			{"variables", variables, 8, "sampleVar, isSample, pi, MaxRetries, Timeout, defaultRetries, singleVar, SingleConst"},
			{"types", types, 1, "User struct"},
			{"interfaces", interfaces, 0, "no interfaces"},
		}

		for _, exp := range expectations {
			t.Run(exp.name, func(t *testing.T) {
				if exp.got != exp.expected {
					t.Errorf("Expected %d %s (%s), got %d", exp.expected, exp.name, exp.desc, exp.got)
				}
			})
		}
	})
}

func TestInterfaces(t *testing.T) {
	p := goparser.New()
	node, err := p.ParseFile("testdata/interfaces.go")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	interfaces := findNodes(node, ast.Interface)
	if len(interfaces) != 6 {
		t.Errorf("Expected 6 interfaces (Reader, Writer, ReadWriter, Handler, Logger, Cache), got %d", len(interfaces))
	}

	// Test embedded interfaces
	for _, iface := range interfaces {
		attrs := iface.Attributes()
		name := attrs["name"].(string)
		if name == "ReadWriter" {
			embedded, ok := attrs["embedded"].([]map[string]any)
			if !ok {
				t.Fatal("Expected embedded to be []map[string]any")
			}
			if len(embedded) != 2 {
				t.Errorf("Expected 2 embedded interfaces, got %d", len(embedded))
			}
		}
	}

	// Test interface methods
	for _, iface := range interfaces {
		attrs := iface.Attributes()
		name := attrs["name"].(string)
		methods, ok := attrs["methods"].([]map[string]any)
		if !ok {
			t.Fatalf("Expected methods to be []map[string]any for interface %s", name)
		}

		switch name {
		case "Handler":
			if len(methods) != 3 {
				t.Errorf("Expected Handler to have 3 methods, got %d", len(methods))
			}
		case "Logger":
			if len(methods) != 3 {
				t.Errorf("Expected Logger to have 3 methods, got %d", len(methods))
			}
		case "Cache":
			if len(methods) != 3 {
				t.Errorf("Expected Cache to have 3 methods, got %d", len(methods))
			}
		}
	}
}

func TestTypes(t *testing.T) {
	p := goparser.New()
	node, err := p.ParseFile("testdata/types.go")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	types := findNodes(node, ast.Type)
	if len(types) != 2 {
		t.Errorf("Expected 2 types (User, UserSettings), got %d", len(types))
	}

	methods := findNodes(node, ast.Method)
	if len(methods) != 2 {
		t.Errorf("Expected 2 methods (UpdateSettings, SendMessage), got %d", len(methods))
	}

	// Test User type fields
	for _, typ := range types {
		attrs := typ.Attributes()
		name := attrs["name"].(string)
		if name == "User" {
			fields, ok := attrs["fields"].([]map[string]any)
			if !ok {
				t.Fatal("Expected fields to be []map[string]any")
			}
			if len(fields) != 9 {
				t.Errorf("Expected User to have 9 fields, got %d", len(fields))
			}

			// Test specific field types
			fieldTypes := map[string]string{
				"ID":       "basic",
				"Name":     "basic",
				"Email":    "basic",
				"Active":   "basic",
				"Metadata": "map",
				"Tags":     "slice",
				"Friends":  "slice",
				"Settings": "pointer",
				"Messages": "chan",
			}

			for _, field := range fields {
				name := field["name"].(string)
				fieldType := field["type"].(*goparser.TypeInfo)
				if expectedKind, ok := fieldTypes[name]; ok {
					if fieldType.Kind != expectedKind {
						t.Errorf("Field %s: expected kind %s, got %s", name, expectedKind, fieldType.Kind)
					}
				}
			}
		}
	}
}

func TestGenerics(t *testing.T) {
	p := goparser.New()
	node, err := p.ParseFile("testdata/generics.go")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	t.Run("Generic Types", func(t *testing.T) {
		types := findNodes(node, ast.Type)
		typesByName := make(map[string]ast.Node)
		for _, typ := range types {
			name := typ.Attributes()["name"].(string)
			typesByName[name] = typ
		}

		// Test List[T]
		if list, ok := typesByName["List"]; ok {
			attrs := list.Attributes()
			typeParams, ok := attrs["type_params"].([]*goparser.TypeInfo)
			if !ok {
				t.Fatal("Expected type_params to be []*TypeInfo")
			}
			if len(typeParams) != 1 {
				t.Errorf("Expected 1 type parameter, got %d", len(typeParams))
			} else {
				param := typeParams[0]
				if param.Name != "T" || !param.IsTypeParam {
					t.Errorf("Expected type parameter T, got %+v", param)
				}
				if len(param.Constraints) != 1 || param.Constraints[0].Name != "any" {
					t.Errorf("Expected constraint 'any', got %+v", param.Constraints)
				}
			}
		} else {
			t.Error("List type not found")
		}

		// Test Map[K, V]
		if m, ok := typesByName["Map"]; ok {
			attrs := m.Attributes()
			typeParams, ok := attrs["type_params"].([]*goparser.TypeInfo)
			if !ok {
				t.Fatal("Expected type_params to be []*TypeInfo")
			}
			if len(typeParams) != 2 {
				t.Errorf("Expected 2 type parameters, got %d", len(typeParams))
			} else {
				k, v := typeParams[0], typeParams[1]
				if k.Name != "K" || !k.IsTypeParam {
					t.Errorf("Expected type parameter K, got %+v", k)
				}
				if len(k.Constraints) != 1 || k.Constraints[0].Name != "comparable" {
					t.Errorf("Expected constraint 'comparable', got %+v", k.Constraints)
				}
				if v.Name != "V" || !v.IsTypeParam {
					t.Errorf("Expected type parameter V, got %+v", v)
				}
				if len(v.Constraints) != 1 || v.Constraints[0].Name != "any" {
					t.Errorf("Expected constraint 'any', got %+v", v.Constraints)
				}
			}
		} else {
			t.Error("Map type not found")
		}

		// Test Number[T]
		if num, ok := typesByName["Number"]; ok {
			attrs := num.Attributes()
			typeParams, ok := attrs["type_params"].([]*goparser.TypeInfo)
			if !ok {
				t.Fatal("Expected type_params to be []*TypeInfo")
			}
			if len(typeParams) != 1 {
				t.Errorf("Expected 1 type parameter, got %d", len(typeParams))
			} else {
				param := typeParams[0]
				if param.Name != "T" || !param.IsTypeParam {
					t.Errorf("Expected type parameter T, got %+v", param)
				}
				if len(param.Constraints) != 2 {
					t.Errorf("Expected 2 constraints, got %d", len(param.Constraints))
				}
			}
		} else {
			t.Error("Number type not found")
		}

		// Test Container interface
		interfaces := findNodes(node, ast.Interface)
		var container ast.Node
		for _, iface := range interfaces {
			if iface.Attributes()["name"] == "Container" {
				container = iface
				break
			}
		}

		if container == nil {
			t.Fatal("Container interface not found")
		}

		attrs := container.Attributes()
		typeParams, ok := attrs["type_params"].([]*goparser.TypeInfo)
		if !ok {
			t.Fatal("Expected type_params to be []*TypeInfo")
		}
		if len(typeParams) != 1 {
			t.Errorf("Expected 1 type parameter, got %d", len(typeParams))
		}

		methods, ok := attrs["methods"].([]map[string]any)
		if !ok {
			t.Fatal("Expected methods to be []map[string]any")
		}
		if len(methods) != 2 {
			t.Errorf("Expected 2 methods, got %d", len(methods))
		}
	})

	t.Run("Generic Function", func(t *testing.T) {
		functions := findNodes(node, ast.Function)
		var transform ast.Node
		for _, fn := range functions {
			if fn.Attributes()["name"] == "Transform" {
				transform = fn
				break
			}
		}

		if transform == nil {
			t.Fatal("Transform function not found")
		}

		attrs := transform.Attributes()
		typeParams, ok := attrs["type_params"].([]*goparser.TypeInfo)
		if !ok {
			t.Fatal("Expected type_params to be []*TypeInfo")
		}
		if len(typeParams) != 2 {
			t.Errorf("Expected 2 type parameters, got %d", len(typeParams))
		} else {
			tParam, uParam := typeParams[0], typeParams[1]
			if tParam.Name != "T" || !tParam.IsTypeParam {
				t.Errorf("Expected type parameter T, got %+v", tParam)
			}
			if uParam.Name != "U" || !uParam.IsTypeParam {
				t.Errorf("Expected type parameter U, got %+v", uParam)
			}
		}

		sig, ok := attrs["signature"].(map[string]any)
		if !ok {
			t.Fatal("Expected signature to be map[string]any")
		}
		params, ok := sig["params"].([]*goparser.TypeInfo)
		if !ok || len(params) != 2 {
			t.Errorf("Expected 2 parameters, got %+v", params)
		}
	})

	t.Run("Generic Method", func(t *testing.T) {
		methods := findNodes(node, ast.Method)
		var add ast.Node
		for _, m := range methods {
			if m.Attributes()["name"] == "Add" {
				add = m
				break
			}
		}

		if add == nil {
			t.Fatal("Add method not found")
		}

		attrs := add.Attributes()
		recvType, ok := attrs["receiver_type"].(*goparser.TypeInfo)
		if !ok {
			t.Fatal("Expected receiver_type to be *TypeInfo")
		}
		if recvType.Kind != "pointer" {
			t.Errorf("Expected pointer receiver, got %s", recvType.Kind)
		}
		if recvType.ElemType.Kind != "generic" {
			t.Errorf("Expected generic type receiver, got %s", recvType.ElemType.Kind)
		}

		sig, ok := attrs["signature"].(map[string]any)
		if !ok {
			t.Fatal("Expected signature to be map[string]any")
		}
		params, ok := sig["params"].([]*goparser.TypeInfo)
		if !ok || len(params) != 1 {
			t.Errorf("Expected 1 parameter, got %+v", params)
		}
	})
}

func TestDirectoryParsing(t *testing.T) {
	p := goparser.New()
	nodes, err := p.ParseDir("testdata")
	if err != nil {
		t.Fatalf("ParseDir failed: %v", err)
	}

	if len(nodes) != 4 {
		t.Errorf("Expected 4 nodes (sample.go, interfaces.go, types.go, generics.go), got %d", len(nodes))
	}

	for _, node := range nodes {
		if node.Type() != string(ast.Module) {
			t.Errorf("Expected type %s, got %s", ast.Module, node.Type())
		}
	}
}

func BenchmarkParser_ParseFile(b *testing.B) {
	parser := goparser.New()

	for b.Loop() {
		node, err := parser.ParseFile("testdata/sample.go")
		if err != nil {
			b.Fatalf("Failed to parse file: %v", err)
		}
		if node == nil {
			b.Fatal("Expected AST node")
		}
	}
}

func BenchmarkParser_ParseDir(b *testing.B) {
	parser := goparser.New()

	for b.Loop() {
		nodes, err := parser.ParseDir("testdata")
		if err != nil {
			b.Fatalf("Failed to parse directory: %v", err)
		}
		if len(nodes) == 0 {
			b.Fatal("Expected AST nodes")
		}
	}
}
