// Package golang provides Go-specific code structure analysis
package gostructure

import (
	"codedna/internal/core/parser/ast"
)

// Node wraps an AST node with Go-specific functionality
type Node struct {
	ast.Node
}

// Returns the programming language of this node
func (n *Node) Language() string {
	return "go"
}

// Creates a new Go node
func NewNode(node ast.Node) *Node {
	return &Node{Node: node}
}

// The type of a code element
type ElementType string

const (
	ElementPackage   ElementType = "package"
	ElementInterface ElementType = "interface"
	ElementTypeDecl  ElementType = "type"
	ElementFunction  ElementType = "function"
	ElementMethod    ElementType = "method"
	ElementVariable  ElementType = "variable"
)

// The type of relationship between elements
type RelationType string

const (
	RelationContains        RelationType = "contains"
	RelationImplements      RelationType = "implements"
	RelationEmbeds          RelationType = "embeds"
	RelationInterfaceEmbeds RelationType = "interface_embeds"
	RelationMethodReceiver  RelationType = "method_receiver"
	RelationCalls           RelationType = "calls" // function/method calls
	RelationReferences      RelationType = "references"
)

// A code element in the structure
type Element struct {
	Type       ElementType
	Name       string
	Attributes map[string]any
}

// A relationship between two elements
type Relationship struct {
	Type   RelationType
	Source *Element
	Target *Element
}

// The analyzed code structure
type Structure struct {
	Elements      []*Element
	Relationships []*Relationship
}

// The results of Go code structure analysis
type Analysis struct {
	language  string
	Structure *Structure
}

// Creates a new analysis result
func NewAnalysis() *Analysis {
	return &Analysis{
		language: "go",
		Structure: &Structure{
			Elements:      make([]*Element, 0),
			Relationships: make([]*Relationship, 0),
		},
	}
}

// Returns the programming language that was analyzed
func (a *Analysis) Language() string {
	return a.language
}
