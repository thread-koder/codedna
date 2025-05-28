// Parses source code into ASTs
package parser

import (
	"slices"

	"codedna/internal/core/parser/ast"
)

// interface for language-specific parsers
type Parser interface {
	ParseFile(filename string) (ast.Node, error)

	ParseDir(dir string) ([]ast.Node, error)

	Language() string

	FileExtensions() []string
}

// maintains a map of available parsers
type Registry struct {
	parsers map[string]Parser // language -> parser
}

// creates a new parser registry
func NewRegistry() *Registry {
	return &Registry{
		parsers: make(map[string]Parser),
	}
}

// adds a parser to the registry
func (r *Registry) Register(p Parser) {
	r.parsers[p.Language()] = p
}

// returns a parser for the given language
func (r *Registry) Get(language string) (Parser, bool) {
	p, ok := r.parsers[language]
	return p, ok
}

// returns a parser that can handle the given file extension
func (r *Registry) GetByExtension(ext string) (Parser, bool) {
	for _, p := range r.parsers {
		if slices.Contains(p.FileExtensions(), ext) {
			return p, true
		}
	}
	return nil, false
}
