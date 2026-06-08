package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/apidoc"
)

func main() {
	serverURL := flag.String("server-url", "http://localhost:8080", "server URL to place in the OpenAPI document")
	outDir := flag.String("out", "docs/swagger", "output directory for generated docs")
	flag.Parse()

	payload, err := apidoc.OpenAPIJSON(*serverURL)
	if err != nil {
		fatal("build OpenAPI JSON: %v", err)
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal("create output directory: %v", err)
	}

	for _, name := range []string{"openapi.json", "swagger.json"} {
		path := filepath.Join(*outDir, name)
		if err := os.WriteFile(path, append(payload, '\n'), 0o644); err != nil {
			fatal("write %s: %v", path, err)
		}
		fmt.Printf("wrote %s\n", path)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
