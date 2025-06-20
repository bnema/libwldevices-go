// Package main provides a code generator for Wayland virtual input protocol bindings.
//
// This tool parses Wayland protocol XML files and generates Go bindings for them.
// It supports generating bindings for virtual pointer, virtual keyboard, pointer constraints,
// and output management protocols.
//
// Usage:
//   go run tools/generate.go -protocol=virtual_pointer -xml=path/to/protocol.xml -output=path/to/output.go
//   go run tools/generate.go -protocol=virtual_keyboard -xml=path/to/protocol.xml -output=path/to/output.go
//   go run tools/generate.go -protocol=output_management -xml=path/to/protocol.xml -output=path/to/output.go
//
// The generator creates Go interfaces and implementation stubs based on the protocol
// specification, making it easier to maintain consistency between the protocol and
// the Go bindings.
package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// ProtocolXML represents the root element of a Wayland protocol XML file
type ProtocolXML struct {
	XMLName     xml.Name      `xml:"protocol"`
	Name        string        `xml:"name,attr"`
	Copyright   string        `xml:"copyright"`
	Description Description   `xml:"description"`
	Interfaces  []Interface  `xml:"interface"`
}

// Interface represents a Wayland protocol interface
type Interface struct {
	Name        string        `xml:"name,attr"`
	Version     string        `xml:"version,attr"`
	Description Description   `xml:"description"`
	Requests    []Request     `xml:"request"`
	Events      []Event       `xml:"event"`
	Enums       []Enum        `xml:"enum"`
}

// Description represents a description element
type Description struct {
	Summary string `xml:"summary,attr"`
	Text    string `xml:",chardata"`
}

// Request represents a protocol request
type Request struct {
	Name        string      `xml:"name,attr"`
	Type        string      `xml:"type,attr"`
	Since       string      `xml:"since,attr"`
	Description Description `xml:"description"`
	Args        []Arg       `xml:"arg"`
}

// Event represents a protocol event
type Event struct {
	Name        string      `xml:"name,attr"`
	Since       string      `xml:"since,attr"`
	Description Description `xml:"description"`
	Args        []Arg       `xml:"arg"`
}

// Arg represents an argument to a request or event
type Arg struct {
	Name        string `xml:"name,attr"`
	Type        string `xml:"type,attr"`
	Summary     string `xml:"summary,attr"`
	Interface   string `xml:"interface,attr"`
	AllowNull   string `xml:"allow-null,attr"`
	Enum        string `xml:"enum,attr"`
}

// Enum represents an enum definition
type Enum struct {
	Name        string      `xml:"name,attr"`
	Since       string      `xml:"since,attr"`
	Description Description `xml:"description"`
	Entries     []Entry     `xml:"entry"`
}

// Entry represents an enum entry
type Entry struct {
	Name    string `xml:"name,attr"`
	Value   string `xml:"value,attr"`
	Summary string `xml:"summary,attr"`
	Since   string `xml:"since,attr"`
}

// TemplateData contains all the data needed for code generation
type TemplateData struct {
	PackageName string
	Protocol    ProtocolXML
	Interfaces  []InterfaceData
	Constants   []ConstantData
}

// InterfaceData contains processed interface data for templates
type InterfaceData struct {
	Name         string
	GoName       string
	Description  string
	Requests     []RequestData
	Events       []EventData
	IsManager    bool
}

// RequestData contains processed request data for templates
type RequestData struct {
	Name        string
	GoName      string
	Description string
	Args        []ArgData
	IsDestructor bool
}

// EventData contains processed event data for templates
type EventData struct {
	Name        string
	GoName      string
	Description string
	Args        []ArgData
}

// ArgData contains processed argument data for templates
type ArgData struct {
	Name     string
	GoName   string
	GoType   string
	Summary  string
}

// ConstantData contains constant definitions
type ConstantData struct {
	Name        string
	Value       string
	Comment     string
}

var (
	protocolFlag = flag.String("protocol", "", "Protocol type (virtual_pointer or virtual_keyboard)")
	xmlFlag      = flag.String("xml", "", "Path to protocol XML file")
	outputFlag   = flag.String("output", "", "Output Go file path")
	packageFlag  = flag.String("package", "", "Go package name (auto-detected if not provided)")
	helpFlag     = flag.Bool("help", false, "Show help")
)

func main() {
	flag.Parse()

	if *helpFlag {
		showHelp()
		return
	}

	if *protocolFlag == "" || *xmlFlag == "" || *outputFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: -protocol, -xml, and -output flags are required\n\n")
		showHelp()
		os.Exit(1)
	}

	if *protocolFlag != "virtual_pointer" && *protocolFlag != "virtual_keyboard" && *protocolFlag != "pointer_constraints" && *protocolFlag != "output_management" {
		fmt.Fprintf(os.Stderr, "Error: -protocol must be 'virtual_pointer', 'virtual_keyboard', 'pointer_constraints', or 'output_management'\n")
		os.Exit(1)
	}

	// Generate the bindings
	if err := generateBindings(*protocolFlag, *xmlFlag, *outputFlag, *packageFlag); err != nil {
		log.Fatalf("Failed to generate bindings: %v", err)
	}

	fmt.Printf("Successfully generated %s bindings: %s\n", *protocolFlag, *outputFlag)
}

func showHelp() {
	fmt.Println("Wayland Virtual Input Protocol Binding Generator")
	fmt.Println("===============================================")
	fmt.Println()
	fmt.Println("This tool generates Go bindings for Wayland virtual input protocols.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run tools/generate.go [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Generate virtual pointer bindings")
	fmt.Println("  go run tools/generate.go \\")
	fmt.Println("    -protocol=virtual_pointer \\")
	fmt.Println("    -xml=../wlr-protocols/unstable/wlr-virtual-pointer-unstable-v1.xml \\")
	fmt.Println("    -output=virtual_pointer/generated.go")
	fmt.Println()
	fmt.Println("  # Generate virtual keyboard bindings")
	fmt.Println("  go run tools/generate.go \\")
	fmt.Println("    -protocol=virtual_keyboard \\")
	fmt.Println("    -xml=path/to/virtual-keyboard-unstable-v1.xml \\")
	fmt.Println("    -output=virtual_keyboard/generated.go")
	fmt.Println()
	fmt.Println("  # Generate output management bindings")
	fmt.Println("  go run tools/generate.go \\")
	fmt.Println("    -protocol=output_management \\")
	fmt.Println("    -xml=path/to/wlr-output-management-unstable-v1.xml \\")
	fmt.Println("    -output=output_management/generated.go")
}

func generateBindings(protocol, xmlPath, outputPath, packageName string) error {
	// Basic path validation to prevent path traversal
	if strings.Contains(xmlPath, "..") || strings.Contains(outputPath, "..") {
		return fmt.Errorf("invalid path: path traversal not allowed")
	}
	
	// Read and parse the XML file
	xmlFile, err := os.Open(xmlPath)
	if err != nil {
		return fmt.Errorf("failed to open XML file: %v", err)
	}
	defer func() { _ = xmlFile.Close() }()

	xmlData, err := io.ReadAll(xmlFile)
	if err != nil {
		return fmt.Errorf("failed to read XML file: %v", err)
	}

	var protocolXML ProtocolXML
	if err := xml.Unmarshal(xmlData, &protocolXML); err != nil {
		return fmt.Errorf("failed to parse XML: %v", err)
	}

	// Auto-detect package name if not provided
	if packageName == "" {
		packageName = protocol
	}

	// Process the protocol data
	templateData := processProtocolData(protocolXML, packageName)

	// Create output directory if it doesn't exist with secure permissions
	if err := os.MkdirAll(filepath.Dir(outputPath), 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Generate the Go code
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer func() { _ = outputFile.Close() }()

	tmpl := template.Must(template.New("bindings").Funcs(templateFuncs).Parse(bindingTemplate))
	if err := tmpl.Execute(outputFile, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	return nil
}

func processProtocolData(protocol ProtocolXML, packageName string) TemplateData {
	data := TemplateData{
		PackageName: packageName,
		Protocol:    protocol,
		Interfaces:  make([]InterfaceData, 0, len(protocol.Interfaces)),
		Constants:   make([]ConstantData, 0),
	}

	// Process interfaces
	for _, iface := range protocol.Interfaces {
		ifaceData := InterfaceData{
			Name:        iface.Name,
			GoName:      toGoName(iface.Name),
			Description: strings.TrimSpace(iface.Description.Text),
			IsManager:   strings.Contains(iface.Name, "manager"),
		}

		// Process requests
		for _, req := range iface.Requests {
			reqData := RequestData{
				Name:        req.Name,
				GoName:      toGoName(req.Name),
				Description: strings.TrimSpace(req.Description.Text),
				IsDestructor: req.Type == "destructor",
			}

			// Process arguments
			for _, arg := range req.Args {
				argData := ArgData{
					Name:    arg.Name,
					GoName:  toGoName(arg.Name),
					GoType:  waylandTypeToGo(arg.Type),
					Summary: arg.Summary,
				}
				reqData.Args = append(reqData.Args, argData)
			}

			ifaceData.Requests = append(ifaceData.Requests, reqData)
		}

		// Process events
		for _, event := range iface.Events {
			eventData := EventData{
				Name:        event.Name,
				GoName:      toGoName(event.Name),
				Description: strings.TrimSpace(event.Description.Text),
			}

			// Process arguments
			for _, arg := range event.Args {
				argData := ArgData{
					Name:    arg.Name,
					GoName:  toGoName(arg.Name),
					GoType:  waylandTypeToGo(arg.Type),
					Summary: arg.Summary,
				}
				eventData.Args = append(eventData.Args, argData)
			}

			ifaceData.Events = append(ifaceData.Events, eventData)
		}

		data.Interfaces = append(data.Interfaces, ifaceData)

		// Process enums as constants
		for _, enum := range iface.Enums {
			for _, entry := range enum.Entries {
				constData := ConstantData{
					Name:    toConstantName(iface.Name, enum.Name, entry.Name),
					Value:   entry.Value,
					Comment: entry.Summary,
				}
				data.Constants = append(data.Constants, constData)
			}
		}
	}

	// Sort constants by name for consistent output
	sort.Slice(data.Constants, func(i, j int) bool {
		return data.Constants[i].Name < data.Constants[j].Name
	})

	return data
}

func toGoName(name string) string {
	// Convert snake_case to PascalCase
	parts := strings.Split(name, "_")
	result := make([]string, len(parts))
	for i, part := range parts {
		if part != "" {
			result[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(result, "")
}

func toConstantName(interfaceName, enumName, entryName string) string {
	// Create a constant name like ERROR_INVALID_AXIS
	parts := []string{}
	if enumName != "" {
		parts = append(parts, strings.ToUpper(enumName))
	}
	parts = append(parts, strings.ToUpper(entryName))
	return strings.Join(parts, "_")
}

func waylandTypeToGo(waylandType string) string {
	switch waylandType {
	case "int":
		return "int32"
	case "uint":
		return "uint32"
	case "fixed":
		return "float64"
	case "string":
		return "string"
	case "object":
		return "interface{}"
	case "new_id":
		return "interface{}"
	case "array":
		return "[]byte"
	case "fd":
		return "*os.File"
	default:
		return "interface{}"
	}
}

var templateFuncs = template.FuncMap{
	"title": func(s string) string {
		if s == "" {
			return s
		}
		return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
	},
	"lower": strings.ToLower,
	"upper": strings.ToUpper,
	"formatComment": formatComment,
	"firstLine": firstLine,
}

// firstLine returns the first line of a string
func firstLine(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return s
}

// formatComment formats a multi-line string for use in Go comments
func formatComment(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	result := make([]string, len(lines))
	for i, line := range lines {
		result[i] = "// " + strings.TrimSpace(line)
	}
	return strings.Join(result, "\n")
}

const bindingTemplate = `// Code generated by tools/generate.go. DO NOT EDIT.
// Source: {{.Protocol.Name}}

// Package {{.PackageName}} provides Go bindings for the {{.Protocol.Name}} Wayland protocol.{{if .Protocol.Description.Text}}
//
// {{.Protocol.Description.Summary}}{{end}}
package {{.PackageName}}

import (
	"context"
	"fmt"
	"os"
	"time"
)

{{if .Constants}}// Constants
{{range .Constants}}const {{.Name}} = {{.Value}}{{if .Comment}} // {{.Comment}}{{end}}
{{end}}
{{end}}

{{range .Interfaces}}
// {{.GoName}} represents the {{.Name}} interface.{{if .Description}}
{{formatComment .Description}}{{end}}
type {{.GoName}} interface {
{{range .Requests}}	// {{.GoName}}{{if .Description}} - {{firstLine .Description}}{{end}}
	{{.GoName}}({{range $i, $arg := .Args}}{{if $i}}, {{end}}{{$arg.GoName}} {{$arg.GoType}}{{end}}) error
{{end}}
}
{{end}}

// Error represents errors that can occur with virtual input operations.
type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("{{.PackageName}} error %d: %s", e.Code, e.Message)
}

{{range .Interfaces}}
{{$ifaceName := .GoName}}
// {{lower .GoName}} is the concrete implementation of {{.GoName}}.
type {{lower .GoName}} struct {
	active bool
}

// New{{.GoName}} creates a new {{.GoName}}.
func New{{.GoName}}(ctx context.Context) ({{.GoName}}, error) {
	// This is a stub implementation - in reality, this would:
	// 1. Connect to the Wayland display
	// 2. Get the registry
	// 3. Bind to {{.Name}}
	// 4. Return the object
	
	return &{{lower .GoName}}{
		active: true,
	}, nil
}

{{range .Requests}}
func (obj *{{lower $ifaceName}}) {{.GoName}}({{range $i, $arg := .Args}}{{if $i}}, {{end}}{{$arg.GoName}} {{$arg.GoType}}{{end}}) error {
	if !obj.active {
		return &Error{
			Code:    -1,
			Message: "object not active",
		}
	}

	// This would send the actual {{.Name}} request to the Wayland compositor
	return nil
}
{{end}}
{{end}}
`