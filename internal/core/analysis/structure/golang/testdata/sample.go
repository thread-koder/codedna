package testdata

// Writer represents a basic writer interface
type Writer interface {
	Write(data []byte) (int, error)
}

// Document represents a document with content
type Document struct {
	content []byte
	writer  Writer
}

// NewDocument creates a new document
func NewDocument(w Writer) *Document {
	return &Document{writer: w}
}

// Write writes document content
func (d *Document) Write(data []byte) (int, error) {
	d.content = append(d.content, data...)
	return d.writer.Write(data)
}

// JSONDocument represents a JSON document
type JSONDocument struct {
	Document // composition
	format   string
}

// Validator checks document validity
type Validator interface {
	Validate() error
}

// ValidatingDocument combines document with validation
type ValidatingDocument struct {
	*Document
	validator Validator
}

// Constants for document types
const (
	TypeText = "text"
	TypeJSON = "json"
)

// GetContent returns the document content
func (d *Document) GetContent() []byte {
	return d.content
}
