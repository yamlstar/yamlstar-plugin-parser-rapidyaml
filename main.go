// The Go implementation is retained as a differential test oracle.
// Release libraries are built from plugin.cpp and contain no Go runtime.
package main

import "encoding/json"

func quoteEDN(value string) string {
	text, _ := json.Marshal(value)
	return string(text)
}

func main() {}
