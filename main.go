// main.go
package main

import (
	"fmt"
	"os"

	"ollama-inspector/cmd/blobs"

	"ollama-inspector/cmd/inspect"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: ollama-inspector <inspect|blobs> [flags]")
		os.Exit(1)
	}
	var err error
	switch os.Args[1] {
	case "inspect":
		err = inspect.Run(os.Args[2:])
	case "blobs":
		err = blobs.Run(os.Args[2:])

	default:
		fmt.Fprintf(os.Stderr, "unknown sub-command %q\n", os.Args[1])
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
