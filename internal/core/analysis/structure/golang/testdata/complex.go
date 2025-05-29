package testdata

// Reader is a basic reader interface
type Reader interface {
	Read() ([]byte, error)
}

// ReadWriter combines Reader and Writer interfaces
type ReadWriter interface {
	Reader
	Writer
}

// Storage represents a data storage interface
type Storage interface {
	Store(data []byte) error
	Load() ([]byte, error)
}

// BaseStorage provides basic storage functionality
type BaseStorage struct {
	data []byte
}

// Store implements Storage.Store
func (b *BaseStorage) Store(data []byte) error {
	b.data = make([]byte, len(data))
	copy(b.data, data)
	return nil
}

// Load implements Storage.Load
func (b *BaseStorage) Load() ([]byte, error) {
	return b.data, nil
}

// MemoryDocument combines Document with storage capabilities
type MemoryDocument struct {
	*Document
	BaseStorage
	cache map[string][]byte
}

// NewMemoryDocument creates a new memory document
func NewMemoryDocument(w Writer) *MemoryDocument {
	return &MemoryDocument{
		Document: NewDocument(w),
		cache:    make(map[string][]byte),
	}
}

// Write overrides Document.Write to add caching
func (m *MemoryDocument) Write(data []byte) (int, error) {
	// Store in cache before writing
	m.cache["latest"] = data
	// Call parent implementation
	return m.Document.Write(data)
}

// Read implements Reader interface
func (m *MemoryDocument) Read() ([]byte, error) {
	return m.Document.GetContent(), nil
}

// CacheStats provides cache statistics
type CacheStats interface {
	CacheSize() int
	CacheKeys() []string
}

// GetCacheSize returns the number of cached items
func (m *MemoryDocument) CacheSize() int {
	return len(m.cache)
}

// GetCacheKeys returns all cache keys
func (m *MemoryDocument) CacheKeys() []string {
	keys := make([]string, 0, len(m.cache))
	for k := range m.cache {
		keys = append(keys, k)
	}
	return keys
}

// DocumentProcessor represents a document processor
type DocumentProcessor interface {
	Process(doc ReadWriter) error
}

// SimpleProcessor is a basic document processor
type SimpleProcessor struct {
	format string
}

// Process implements DocumentProcessor.Process
func (p *SimpleProcessor) Process(doc ReadWriter) error {
	_, err := doc.Read()
	if err != nil {
		return err
	}
	// Process and write back
	return nil
}
