package goparser_test

import (
	"os"
	"path/filepath"
	"slices"
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

const testFileContent = `
package example

import (
	"fmt"
	custom "github.com/example/pkg"
)

// A struct type
type Users struct {
	Name string
}

// An interface type
type Handler interface {
	Handle(msg string) error
}

func (u *Users) SayHello() {
	fmt.Println("Hello")
}

func DoSomething() error {
	return nil
}

// Grouped declarations
const (
	MaxRetries = 3
	Timeout    = 30
)

var (
	defaultRetries = 3
)

// Ungrouped declarations
const SingleConst = 42
var singleVar = "test"

// Multiple variables in single declaration
var two, three int

// Variables with inferred types
var (
	str = "hello"              // string
	num = 42                   // int
	pi = 3.14                  // float64
	slice = make([]int, 10)    // []int
	ptr = &str                 // *string
)
`

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
	// Setup test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte(testFileContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	p := goparser.New()
	node, err := p.ParseFile(testFile)
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
			{"imports", imports, 2, "fmt and custom imports"},
			{"methods", methods, 1, "SayHello method"},
			{"functions", functions, 1, "DoSomething function"},
			{"variables", variables, 12, "MaxRetries, Timeout, SingleConst, defaultRetries, singleVar, two, three, str, num, pi, slice, ptr"},
			{"types", types, 1, "Users struct"},
			{"interfaces", interfaces, 1, "Handler interface"},
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

func TestNodeAttributes(t *testing.T) {
	src := `
	package main

	import (
		"fmt"
		custom "github.com/example/pkg"
	)

	var (
		name = "John"
		age = 30
		pi = 3.14
		isValid = true
		ptr = &age
		MaxCount = 100  // Exported variable
	)

	var slice = make([]string, 0)
	var m = make(map[string]int)
	var ch = make(chan int)

	type Person struct {
		Name string
		Age  int
	}

	func main() {
		fmt.Println("Hello")
	}

	func add(a, b int) int {
		return a + b
	}

	type Handler interface {
		Handle(msg string) error
	}

	type MyHandler struct{}

	func (h *MyHandler) Handle(msg string) error {
		return nil
	}
	`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte(src), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	p := goparser.New()
	root, err := p.ParseFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	t.Run("Package", func(t *testing.T) {
		if pkg := root.Attributes()["package_name"]; pkg != "main" {
			t.Errorf("Expected package name 'main', got '%v'", pkg)
		}
	})

	t.Run("Imports", func(t *testing.T) {
		imports := findNodes(root, ast.Import)
		if len(imports) != 2 {
			t.Errorf("Expected 2 imports, got %d", len(imports))
		}

		for _, imp := range imports {
			attrs := imp.Attributes()
			path := attrs["path"].(string)
			isStdLib := attrs["is_std_lib"].(bool)

			switch path {
			case "fmt":
				if !isStdLib {
					t.Error("Expected 'fmt' to be a standard library import")
				}
			case "github.com/example/pkg":
				if isStdLib {
					t.Error("Expected 'github.com/example/pkg' to not be a standard library import")
				}
				if alias, ok := attrs["alias"].(string); !ok || alias != "custom" {
					t.Errorf("Expected alias 'custom', got %v", alias)
				}
			default:
				t.Errorf("Unexpected import path: %s", path)
			}
		}

		// Check dependencies
		if deps, ok := root.Attributes()["dependencies"].([]string); !ok {
			t.Error("Expected dependencies attribute to be []string")
		} else if len(deps) != 2 {
			t.Errorf("Expected 2 dependencies, got %d", len(deps))
		} else {
			expected := []string{"fmt", "github.com/example/pkg"}
			for _, exp := range expected {
				found := slices.Contains(deps, exp)
				if !found {
					t.Errorf("Expected dependency '%s' not found", exp)
				}
			}
		}
	})

	t.Run("Variables", func(t *testing.T) {
		variables := findNodes(root, ast.Variable)
		if len(variables) != 9 {
			t.Errorf("Expected 9 variables, got %d", len(variables))
		}

		for _, v := range variables {
			attrs := v.Attributes()
			name := attrs["name"].(string)
			typeInfo, ok := attrs["type"].(*goparser.TypeInfo)
			if !ok {
				t.Errorf("For variable %s: Expected type to be *TypeInfo, got %T", name, attrs["type"])
				continue
			}

			switch name {
			case "MaxCount":
				if typeInfo.Kind != "basic" || typeInfo.Name != "int" {
					t.Errorf("Expected type={Kind: basic, Name: int}, got %+v", typeInfo)
				}
				if exp := attrs["is_exported"].(bool); !exp {
					t.Error("Expected 'MaxCount' to be exported")
				}
			case "name":
				if typeInfo.Kind != "basic" || typeInfo.Name != "string" {
					t.Errorf("Expected type={Kind: basic, Name: string}, got %+v", typeInfo)
				}
				if exp := attrs["is_exported"].(bool); exp {
					t.Error("Expected 'name' to not be exported")
				}
			case "age":
				if typeInfo.Kind != "basic" || typeInfo.Name != "int" {
					t.Errorf("Expected type={Kind: basic, Name: int}, got %+v", typeInfo)
				}
				if exp := attrs["is_exported"].(bool); exp {
					t.Error("Expected 'age' to not be exported")
				}
			case "pi":
				if typeInfo.Kind != "basic" || typeInfo.Name != "float" {
					t.Errorf("Expected type={Kind: basic, Name: float}, got %+v", typeInfo)
				}
				if exp := attrs["is_exported"].(bool); exp {
					t.Error("Expected 'pi' to not be exported")
				}
			case "isValid":
				if typeInfo.Kind != "basic" || typeInfo.Name != "bool" {
					t.Errorf("Expected type={Kind: basic, Name: bool}, got %+v", typeInfo)
				}
				if exp := attrs["is_exported"].(bool); exp {
					t.Error("Expected 'isValid' to not be exported")
				}
			case "ptr":
				if typeInfo.Kind != "pointer" || typeInfo.ElemType == nil || typeInfo.ElemType.Name != "int" {
					t.Errorf("Expected type={Kind: pointer, ElemType: {Kind: basic, Name: int}}, got %+v", typeInfo)
				}
				if exp := attrs["is_exported"].(bool); exp {
					t.Error("Expected 'ptr' to not be exported")
				}
			case "slice":
				if typeInfo.Kind != "slice" || typeInfo.ElemType == nil || typeInfo.ElemType.Name != "string" {
					t.Errorf("Expected type={Kind: slice, ElemType: {Kind: basic, Name: string}}, got %+v", typeInfo)
				}
				if exp := attrs["is_exported"].(bool); exp {
					t.Error("Expected 'slice' to not be exported")
				}
			case "m":
				if typeInfo.Kind != "map" || typeInfo.KeyType == nil || typeInfo.KeyType.Name != "string" || typeInfo.ValueType == nil || typeInfo.ValueType.Name != "int" {
					t.Errorf("Expected type={Kind: map, KeyType: {Kind: basic, Name: string}, ValueType: {Kind: basic, Name: int}}, got %+v", typeInfo)
				}
				if exp := attrs["is_exported"].(bool); exp {
					t.Error("Expected 'm' to not be exported")
				}
			case "ch":
				if typeInfo.Kind != "chan" || typeInfo.ElemType == nil || typeInfo.ElemType.Name != "int" {
					t.Errorf("Expected type={Kind: chan, ElemType: {Kind: basic, Name: int}}, got %+v", typeInfo)
				}
				if exp := attrs["is_exported"].(bool); exp {
					t.Error("Expected 'ch' to not be exported")
				}
			}
		}
	})

	t.Run("Functions", func(t *testing.T) {
		functions := findNodes(root, ast.Function)
		if len(functions) != 2 {
			t.Errorf("Expected 2 functions, got %d", len(functions))
		}

		for _, f := range functions {
			attrs := f.Attributes()
			name := attrs["name"].(string)
			sig, ok := attrs["signature"].(map[string]any)
			if !ok {
				t.Errorf("For function %s: Expected signature to be a map[string]any, got %T", name, attrs["signature"])
				continue
			}

			switch name {
			case "main":
				params, ok := sig["params"].([]*goparser.TypeInfo)
				if !ok || len(params) != 0 {
					t.Errorf("Expected main params=[], got %v", sig["params"])
				}
				returns, ok := sig["returns"].([]*goparser.TypeInfo)
				if !ok || len(returns) != 0 {
					t.Errorf("Expected main returns=[], got %v", sig["returns"])
				}
			case "add":
				params, ok := sig["params"].([]*goparser.TypeInfo)
				if !ok || len(params) != 2 {
					t.Errorf("Expected add params to have 2 elements, got %v", sig["params"])
				} else {
					for _, param := range params {
						if param.Kind != "basic" || param.Name != "int" {
							t.Errorf("Expected param type={Kind: basic, Name: int}, got %+v", param)
						}
					}
				}
				returns, ok := sig["returns"].([]*goparser.TypeInfo)
				if !ok || len(returns) != 1 {
					t.Errorf("Expected add returns to have 1 element, got %v", sig["returns"])
				} else {
					ret := returns[0]
					if ret.Kind != "basic" || ret.Name != "int" {
						t.Errorf("Expected return type={Kind: basic, Name: int}, got %+v", ret)
					}
				}
			}
		}
	})

	t.Run("Methods", func(t *testing.T) {
		methods := findNodes(root, ast.Method)
		if len(methods) != 1 {
			t.Errorf("Expected 1 method, got %d", len(methods))
		}

		if len(methods) > 0 {
			m := methods[0]
			attrs := m.Attributes()
			if name := attrs["name"]; name != "Handle" {
				t.Errorf("Expected name=Handle, got %v", name)
			}

			recvType, ok := attrs["receiver_type"].(*goparser.TypeInfo)
			if !ok {
				t.Errorf("Expected receiver_type to be *TypeInfo, got %T", attrs["receiver_type"])
			} else if recvType.Kind != "pointer" || recvType.ElemType == nil || recvType.ElemType.Name != "MyHandler" {
				t.Errorf("Expected receiver_type={Kind: pointer, ElemType: {Kind: basic, Name: MyHandler}}, got %+v", recvType)
			}

			sig, ok := attrs["signature"].(map[string]any)
			if !ok {
				t.Errorf("Expected signature to be a map[string]any, got %T", attrs["signature"])
				return
			}

			params, ok := sig["params"].([]*goparser.TypeInfo)
			if !ok || len(params) != 1 {
				t.Errorf("Expected params to have 1 element, got %v", sig["params"])
			} else {
				param := params[0]
				if param.Kind != "basic" || param.Name != "string" {
					t.Errorf("Expected param type={Kind: basic, Name: string}, got %+v", param)
				}
			}

			returns, ok := sig["returns"].([]*goparser.TypeInfo)
			if !ok || len(returns) != 1 {
				t.Errorf("Expected returns to have 1 element, got %v", sig["returns"])
			} else {
				ret := returns[0]
				if ret.Kind != "basic" || ret.Name != "error" {
					t.Errorf("Expected return type={Kind: basic, Name: error}, got %+v", ret)
				}
			}
		}
	})

	t.Run("Types", func(t *testing.T) {
		types := findNodes(root, ast.Type)
		if len(types) != 2 {
			t.Errorf("Expected 2 types, got %d", len(types))
		}

		for _, typ := range types {
			attrs := typ.Attributes()
			name := attrs["name"].(string)
			switch name {
			case "Person":
				if exp := attrs["is_exported"].(bool); !exp {
					t.Error("Expected Person to be exported")
				}
			case "MyHandler":
				if exp := attrs["is_exported"].(bool); !exp {
					t.Error("Expected MyHandler to be exported")
				}
			}
		}
	})

	t.Run("Interfaces", func(t *testing.T) {
		interfaces := findNodes(root, ast.Interface)
		if len(interfaces) != 1 {
			t.Errorf("Expected 1 interface, got %d", len(interfaces))
		}

		if len(interfaces) > 0 {
			i := interfaces[0]
			attrs := i.Attributes()
			if name := attrs["name"]; name != "Handler" {
				t.Errorf("Expected name=Handler, got %v", name)
			}
			if exp := attrs["is_exported"].(bool); !exp {
				t.Error("Expected Handler to be exported")
			}

			// Verify interface methods
			methods, ok := attrs["methods"].([]map[string]any)
			if !ok {
				t.Errorf("Expected methods to be []map[string]interface{}, got %T", attrs["methods"])
				return
			}
			if len(methods) != 1 {
				t.Errorf("Expected 1 method, got %d", len(methods))
				return
			}

			method := methods[0]
			if name := method["name"].(string); name != "Handle" {
				t.Errorf("Expected method name=Handle, got %v", name)
			}

			sig, ok := method["signature"].(map[string]any)
			if !ok {
				t.Errorf("Expected signature to be map[string]interface{}, got %T", method["signature"])
				return
			}

			params, ok := sig["params"].([]*goparser.TypeInfo)
			if !ok || len(params) != 1 {
				t.Errorf("Expected params to have 1 element, got %v", sig["params"])
			} else {
				param := params[0]
				if param.Kind != "basic" || param.Name != "string" {
					t.Errorf("Expected param type={Kind: basic, Name: string}, got %+v", param)
				}
			}

			returns, ok := sig["returns"].([]*goparser.TypeInfo)
			if !ok || len(returns) != 1 {
				t.Errorf("Expected returns to have 1 element, got %v", sig["returns"])
			} else {
				ret := returns[0]
				if ret.Kind != "basic" || ret.Name != "error" {
					t.Errorf("Expected return type={Kind: basic, Name: error}, got %+v", ret)
				}
			}
		}
	})
}

func TestDirectoryParsing(t *testing.T) {
	tmpDir := t.TempDir()

	testFile1 := filepath.Join(tmpDir, "file1.go")
	if err := os.WriteFile(testFile1, []byte(testFileContent), 0644); err != nil {
		t.Fatalf("Failed to create test file1: %v", err)
	}

	testFile2 := filepath.Join(tmpDir, "file2.go")
	secondFileContent := `
	package example

	func AnotherFunction() {
	}
	`
	if err := os.WriteFile(testFile2, []byte(secondFileContent), 0644); err != nil {
		t.Fatalf("Failed to create test file2: %v", err)
	}

	// Parse and verify
	p := goparser.New()
	nodes, err := p.ParseDir(tmpDir)
	if err != nil {
		t.Fatalf("ParseDir failed: %v", err)
	}

	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(nodes))
	}

	for i, node := range nodes {
		if node.Type() != string(ast.Module) {
			t.Errorf("Node %d: expected type %s, got %s", i, ast.Module, node.Type())
		}
	}
}

func BenchmarkParseFile(b *testing.B) {
	parser := goparser.New()
	filename := "testdata/sample.go"

	for b.Loop() {
		_, err := parser.ParseFile(filename)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseFileWithTypes(b *testing.B) {
	parser := goparser.New()
	filename := "testdata/types.go"

	for b.Loop() {
		_, err := parser.ParseFile(filename)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseFileWithInterfaces(b *testing.B) {
	parser := goparser.New()
	filename := "testdata/interfaces.go"

	for b.Loop() {
		_, err := parser.ParseFile(filename)
		if err != nil {
			b.Fatal(err)
		}
	}
}
