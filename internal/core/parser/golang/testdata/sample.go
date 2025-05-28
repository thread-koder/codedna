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
