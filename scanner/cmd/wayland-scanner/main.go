// Command wayland-scanner generates Go bindings from Wayland protocol XML files.
//
// Usage:
//   wayland-scanner [flags] protocol.xml
//
// Flags:
//   -o, --output    Output file path (default: <protocol>_generated.go)
//   -p, --package   Go package name (default: derived from protocol name)
//   -h, --help      Show help
//
// Examples:
//   # Generate bindings for virtual pointer protocol
//   wayland-scanner -o virtual_pointer.go ../wlr-protocols/unstable/wlr-virtual-pointer-unstable-v1.xml
//
//   # Generate with custom package name
//   wayland-scanner -p vpointer -o vpointer.go protocol.xml
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bnema/libwldevices-go/scanner"
)

func main() {
	var (
		outputPath  string
		packageName string
		showHelp    bool
	)

	flag.StringVar(&outputPath, "o", "", "Output file path")
	flag.StringVar(&outputPath, "output", "", "Output file path")
	flag.StringVar(&packageName, "p", "", "Go package name")
	flag.StringVar(&packageName, "package", "", "Go package name")
	flag.BoolVar(&showHelp, "h", false, "Show help")
	flag.BoolVar(&showHelp, "help", false, "Show help")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] protocol.xml\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generates Go bindings from Wayland protocol XML files.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Generate bindings for virtual pointer protocol\n")
		fmt.Fprintf(os.Stderr, "  %s -o virtual_pointer.go wlr-virtual-pointer-unstable-v1.xml\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Generate with custom package name\n")
		fmt.Fprintf(os.Stderr, "  %s -p vpointer -o vpointer.go protocol.xml\n", os.Args[0])
	}

	flag.Parse()

	if showHelp || flag.NArg() == 0 {
		flag.Usage()
		if showHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}

	xmlPath := flag.Arg(0)

	// Derive output path if not specified
	if outputPath == "" {
		base := filepath.Base(xmlPath)
		base = strings.TrimSuffix(base, filepath.Ext(base))
		outputPath = base + "_generated.go"
	}

	// Derive package name if not specified
	if packageName == "" {
		base := filepath.Base(xmlPath)
		base = strings.TrimSuffix(base, filepath.Ext(base))
		
		// Clean up common prefixes/suffixes
		base = strings.TrimPrefix(base, "wlr-")
		base = strings.TrimPrefix(base, "zwlr-")
		base = strings.TrimSuffix(base, "-unstable-v1")
		base = strings.TrimSuffix(base, "-unstable-v2")
		base = strings.TrimSuffix(base, "-stable-v1")
		
		// Convert to valid Go package name
		packageName = strings.ReplaceAll(base, "-", "_")
	}

	// Create scanner
	s := scanner.NewScanner()

	// Parse XML
	if err := s.ParseXML(xmlPath); err != nil {
		log.Fatalf("Failed to parse XML: %v", err)
	}

	// Generate code
	code, err := s.Generate(packageName)
	if err != nil {
		log.Fatalf("Failed to generate code: %v", err)
	}

	// Write output with secure permissions
	if err := os.WriteFile(outputPath, code, 0600); err != nil {
		log.Fatalf("Failed to write output: %v", err)
	}

	fmt.Printf("Generated %s (package %s)\n", outputPath, packageName)
}