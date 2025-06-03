package sample

import (
	"fmt"
	"strings"
)

// User represents a basic user type
type User struct {
	Name string
}

func ProcessString(s string) string {
	return strings.ToUpper(s)
}

func main() {
	result := ProcessString("hello")
	fmt.Println(result)
}

var (
	sampleVar = "sample"
	isSample  = true
	pi        = 3.14
)

const (
	MaxRetries = 3
	Timeout    = 30
)

var (
	defaultRetries = 3
)

const SingleConst = 42

var singleVar = "test"
