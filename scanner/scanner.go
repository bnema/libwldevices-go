// Package scanner provides a Wayland protocol scanner that generates Go bindings from XML protocol files.
//
// This scanner parses Wayland protocol XML files and generates idiomatic Go code with:
// - Type-safe interfaces and implementations
// - Proper marshalling/unmarshalling of Wayland wire format
// - Connection handling and message dispatching
// - Error handling and validation
package scanner

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"go/format"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

// Protocol represents a Wayland protocol specification
type Protocol struct {
	XMLName     xml.Name     `xml:"protocol"`
	Name        string       `xml:"name,attr"`
	Copyright   string       `xml:"copyright"`
	Description *Description `xml:"description"`
	Interfaces  []Interface  `xml:"interface"`
}

// Interface represents a Wayland interface in the protocol
type Interface struct {
	Name        string       `xml:"name,attr"`
	Version     int          `xml:"version,attr"`
	Description *Description `xml:"description"`
	Requests    []Request    `xml:"request"`
	Events      []Event      `xml:"event"`
	Enums       []Enum       `xml:"enum"`
}

// Request represents a client-to-server message
type Request struct {
	Name        string       `xml:"name,attr"`
	Type        string       `xml:"type,attr"` // "destructor" or empty
	Since       int          `xml:"since,attr"`
	Description *Description `xml:"description"`
	Args        []Arg        `xml:"arg"`
}

// Event represents a server-to-client message
type Event struct {
	Name        string       `xml:"name,attr"`
	Since       int          `xml:"since,attr"`
	Description *Description `xml:"description"`
	Args        []Arg        `xml:"arg"`
}

// Arg represents a message argument
type Arg struct {
	Name      string `xml:"name,attr"`
	Type      string `xml:"type,attr"`
	Summary   string `xml:"summary,attr"`
	Interface string `xml:"interface,attr"`
	AllowNull bool   `xml:"allow-null,attr"`
	Enum      string `xml:"enum,attr"`
}

// Enum represents an enumeration type
type Enum struct {
	Name        string       `xml:"name,attr"`
	Since       int          `xml:"since,attr"`
	Bitfield    bool         `xml:"bitfield,attr"`
	Description *Description `xml:"description"`
	Entries     []Entry      `xml:"entry"`
}

// Entry represents an enum value
type Entry struct {
	Name    string `xml:"name,attr"`
	Value   string `xml:"value,attr"`
	Summary string `xml:"summary,attr"`
	Since   int    `xml:"since,attr"`
}

// Description represents documentation
type Description struct {
	Summary string `xml:"summary,attr"`
	Text    string `xml:",chardata"`
}

// Scanner generates Go bindings from Wayland protocol XML
type Scanner struct {
	protocol *Protocol
	imports  map[string]bool
}

// NewScanner creates a new protocol scanner
func NewScanner() *Scanner {
	return &Scanner{
		imports: make(map[string]bool),
	}
}

// ParseXML parses a Wayland protocol XML file
func (s *Scanner) ParseXML(path string) error {
	// Basic path validation to prevent path traversal
	if strings.Contains(path, "..") {
		return fmt.Errorf("invalid path: path traversal not allowed")
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open XML file: %w", err)
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read XML file: %w", err)
	}

	var protocol Protocol
	if err := xml.Unmarshal(data, &protocol); err != nil {
		return fmt.Errorf("failed to parse XML: %w", err)
	}

	s.protocol = &protocol
	return nil
}

// Generate creates Go source code for the parsed protocol
func (s *Scanner) Generate(packageName string) ([]byte, error) {
	if s.protocol == nil {
		return nil, fmt.Errorf("no protocol parsed")
	}

	// Reset imports
	s.imports = map[string]bool{
		"fmt":     true,
		"errors":  true,
		"time":    true,
		"sync":    true,
		"unsafe":  true,
		"syscall": true,
	}

	// Generate code
	var buf bytes.Buffer
	data := s.prepareTemplateData(packageName)

	tmpl := template.Must(template.New("protocol").Funcs(s.templateFuncs()).Parse(protocolTemplate))
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Return unformatted code for debugging
		return buf.Bytes(), fmt.Errorf("failed to format generated code: %w", err)
	}

	return formatted, nil
}

// templateData holds data for template generation
type templateData struct {
	Package     string
	Imports     []string
	Protocol    string
	Interfaces  []interfaceData
	Constants   []constantData
	WireFormats []wireFormatData
	RequestOps  []opData
	EventOps    []opData
}

type interfaceData struct {
	Name         string
	GoName       string
	Version      int
	Description  string
	Requests     []messageData
	Events       []messageData
	RequestOps   []opData
	EventOps     []opData
	IsManager    bool
	ManagedType  string
}

type messageData struct {
	Name        string
	GoName      string
	Opcode      int
	Description string
	Args        []argData
	Since       int
	IsDestructor bool
}

type argData struct {
	Name       string
	GoName     string
	Type       string
	GoType     string
	WireType   string
	Interface  string
	AllowNull  bool
	IsNewID    bool
	IsEnum     bool
	EnumType   string
}

type constantData struct {
	Name    string
	Value   string
	Type    string
	Comment string
}

type opData struct {
	Interface string
	Name      string
	Value     int
}

type wireFormatData struct {
	Name   string
	GoType string
	Size   int
}

func (s *Scanner) prepareTemplateData(packageName string) templateData {
	data := templateData{
		Package:  packageName,
		Protocol: s.protocol.Name,
	}

	// Collect imports
	for imp := range s.imports {
		data.Imports = append(data.Imports, imp)
	}
	sort.Strings(data.Imports)

	// Process interfaces
	for _, iface := range s.protocol.Interfaces {
		ifaceData := s.processInterface(iface)
		data.Interfaces = append(data.Interfaces, ifaceData)

		// Collect opcodes for dispatch tables
		for _, req := range ifaceData.Requests {
			data.RequestOps = append(data.RequestOps, opData{
				Interface: ifaceData.Name,
				Name:      req.Name,
				Value:     req.Opcode,
			})
		}
		for _, event := range ifaceData.Events {
			data.EventOps = append(data.EventOps, opData{
				Interface: ifaceData.Name,
				Name:      event.Name,
				Value:     event.Opcode,
			})
		}

		// Process enums as constants
		for _, enum := range iface.Enums {
			for _, entry := range enum.Entries {
				constName := s.toConstantName(iface.Name, enum.Name, entry.Name)
				data.Constants = append(data.Constants, constantData{
					Name:    constName,
					Value:   entry.Value,
					Type:    "int32",
					Comment: entry.Summary,
				})
			}
		}
	}

	// Add wire format types
	data.WireFormats = []wireFormatData{
		{Name: "Int", GoType: "int32", Size: 4},
		{Name: "Uint", GoType: "uint32", Size: 4},
		{Name: "Fixed", GoType: "Fixed", Size: 4},
		{Name: "String", GoType: "string", Size: -1},
		{Name: "Object", GoType: "uint32", Size: 4},
		{Name: "NewID", GoType: "uint32", Size: 4},
		{Name: "Array", GoType: "[]byte", Size: -1},
		{Name: "FD", GoType: "int32", Size: 4},
	}

	return data
}

func (s *Scanner) processInterface(iface Interface) interfaceData {
	data := interfaceData{
		Name:        iface.Name,
		GoName:      s.toGoName(iface.Name),
		Version:     iface.Version,
		Description: s.formatDescription(iface.Description),
		IsManager:   strings.Contains(iface.Name, "manager"),
	}

	// Determine managed type for managers
	if data.IsManager {
		// Extract managed type name (e.g., zwlr_virtual_pointer_manager_v1 -> zwlr_virtual_pointer_v1)
		parts := strings.Split(iface.Name, "_")
		for i, part := range parts {
			if part == "manager" && i > 0 {
				parts = append(parts[:i], parts[i+1:]...)
				break
			}
		}
		data.ManagedType = strings.Join(parts, "_")
	}

	// Process requests
	for i, req := range iface.Requests {
		msgData := messageData{
			Name:         req.Name,
			GoName:       s.toGoName(req.Name),
			Opcode:       i,
			Description:  s.formatDescription(req.Description),
			Since:        req.Since,
			IsDestructor: req.Type == "destructor",
		}

		for _, arg := range req.Args {
			msgData.Args = append(msgData.Args, s.processArg(arg))
		}

		data.Requests = append(data.Requests, msgData)
	}

	// Process events
	for i, event := range iface.Events {
		msgData := messageData{
			Name:        event.Name,
			GoName:      s.toGoName(event.Name),
			Opcode:      i,
			Description: s.formatDescription(event.Description),
			Since:       event.Since,
		}

		for _, arg := range event.Args {
			msgData.Args = append(msgData.Args, s.processArg(arg))
		}

		data.Events = append(data.Events, msgData)
	}

	return data
}

func (s *Scanner) processArg(arg Arg) argData {
	data := argData{
		Name:      arg.Name,
		GoName:    s.toGoName(arg.Name),
		Type:      arg.Type,
		Interface: arg.Interface,
		AllowNull: arg.AllowNull,
		IsNewID:   arg.Type == "new_id",
	}

	// Determine Go type and wire type
	switch arg.Type {
	case "int":
		data.GoType = "int32"
		data.WireType = "Int"
	case "uint":
		data.GoType = "uint32"
		data.WireType = "Uint"
	case "fixed":
		data.GoType = "Fixed"
		data.WireType = "Fixed"
		s.imports["math"] = true
	case "string":
		data.GoType = "string"
		data.WireType = "String"
	case "object":
		if arg.Interface != "" {
			data.GoType = "*" + s.toGoName(arg.Interface)
		} else {
			data.GoType = "WaylandObject"
		}
		data.WireType = "Object"
	case "new_id":
		if arg.Interface != "" {
			data.GoType = "*" + s.toGoName(arg.Interface)
		} else {
			data.GoType = "WaylandObject"
		}
		data.WireType = "NewID"
	case "array":
		data.GoType = "[]byte"
		data.WireType = "Array"
	case "fd":
		data.GoType = "int"
		data.WireType = "FD"
		s.imports["os"] = true
	default:
		data.GoType = "interface{}"
		data.WireType = "Unknown"
	}

	// Handle enum types
	if arg.Enum != "" {
		data.IsEnum = true
		data.EnumType = arg.Enum
		// Keep numeric type for enums
		if data.GoType == "interface{}" {
			data.GoType = "uint32"
		}
	}

	return data
}

func (s *Scanner) toGoName(name string) string {
	// Remove protocol prefix and version suffix
	name = strings.TrimPrefix(name, "zwlr_")
	name = strings.TrimPrefix(name, "zwp_")
	name = strings.TrimPrefix(name, "wl_")
	name = strings.TrimSuffix(name, "_v1")
	name = strings.TrimSuffix(name, "_v2")

	// Convert snake_case to PascalCase
	parts := strings.Split(name, "_")
	for i, part := range parts {
		if part != "" {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

func (s *Scanner) toConstantName(iface, enum, entry string) string {
	// Build constant name
	parts := []string{}
	
	// Add enum name
	enumParts := strings.Split(enum, "_")
	for _, part := range enumParts {
		if part != "" {
			parts = append(parts, strings.ToUpper(part))
		}
	}
	
	// Add entry name
	entryParts := strings.Split(entry, "_")
	for _, part := range entryParts {
		if part != "" {
			parts = append(parts, strings.ToUpper(part))
		}
	}
	
	return strings.Join(parts, "_")
}

func (s *Scanner) formatDescription(desc *Description) string {
	if desc == nil {
		return ""
	}
	
	text := strings.TrimSpace(desc.Text)
	if text == "" {
		return desc.Summary
	}
	
	// Format multiline descriptions
	lines := strings.Split(text, "\n")
	formatted := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			formatted = append(formatted, line)
		}
	}
	
	return strings.Join(formatted, " ")
}

func (s *Scanner) templateFuncs() template.FuncMap {
	return template.FuncMap{
		"lower": strings.ToLower,
		"title": func(s string) string {
			if s == "" {
				return s
			}
			return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
		},
		"hasPrefix": strings.HasPrefix,
		"trimPrefix": strings.TrimPrefix,
		"quote": strconv.Quote,
	}
}