// Language-agnostic AST representations
package ast

import "fmt"

// represents a position in source code
type Position struct {
	Line   int
	Column int
	Offset int
}

// AST node
type Node interface {
	Position() Position

	Type() string

	Children() []Node

	Attributes() map[string]any
}

type NodeType string

const (
	Function  NodeType = "Function"
	Method    NodeType = "Method"
	Import    NodeType = "Import"
	Variable  NodeType = "Variable"
	Type      NodeType = "Type"
	Interface NodeType = "Interface"
	Block     NodeType = "Block"
	Module    NodeType = "Module"
)

// provides a basic implementation of Node
type BaseNode struct {
	pos        Position
	nodeType   NodeType
	children   []Node
	attributes map[string]any
}

// creates a new BaseNode
func NewBaseNode(nodeType NodeType, pos Position) *BaseNode {
	return &BaseNode{
		pos:        pos,
		nodeType:   nodeType,
		children:   make([]Node, 0),
		attributes: make(map[string]any),
	}
}

func (n *BaseNode) Position() Position         { return n.pos }
func (n *BaseNode) Type() string               { return string(n.nodeType) }
func (n *BaseNode) Children() []Node           { return n.children }
func (n *BaseNode) Attributes() map[string]any { return n.attributes }

// adds a child node
func (n *BaseNode) AddChild(child Node) {
	n.children = append(n.children, child)
}

// sets a node attribute
func (n *BaseNode) SetAttribute(key string, value any) {
	n.attributes[key] = value
}

// provides a debug representation of the node
func (n *BaseNode) String() string {
	return fmt.Sprintf("%s at line %d", n.nodeType, n.pos.Line)
}
