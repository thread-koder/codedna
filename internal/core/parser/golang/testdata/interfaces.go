package interfaces

type Reader interface {
	Read(p []byte) (n int, err error)
}

type Writer interface {
	Write(p []byte) (n int, err error)
}

type ReadWriter interface {
	Reader
	Writer
	Close() error
}

type Handler interface {
	Handle(req *Request) (*Response, error)
	Middleware(next Handler) Handler
	Validate(data interface{}) error
}

type Request struct {
	Method  string
	Path    string
	Headers map[string][]string
	Body    Reader
}

type Response struct {
	Status  int
	Headers map[string][]string
	Body    Writer
}

type BaseHandler struct {
	logger Logger
	cache  Cache
}

type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

type Cache interface {
	Get(key string) (interface{}, error)
	Set(key string, value interface{}) error
	Delete(key string) error
}
