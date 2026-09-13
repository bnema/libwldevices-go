// Package scanner generates WLTurbo-backed Go bindings from Wayland protocol
// XML files.
//
// The generated code owns protocol semantics only: it embeds wl.BaseProxy,
// allocates and registers child objects through the wl.Context, sends requests
// through that context, and dispatches typed events to registered handlers.
// Message framing, descriptor passing and connection lifetime stay in the
// transport, so a generated binding never opens a socket or writes a header.
package scanner

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// TransportImport is the import path generated bindings use for the Wayland
// transport.
const TransportImport = "github.com/bnema/wlturbo/wl"

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
	source   string
}

// NewScanner creates a new protocol scanner
func NewScanner() *Scanner {
	return &Scanner{}
}

// ParseXML parses a Wayland protocol XML file
func (s *Scanner) ParseXML(path string) error {
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
	// Only the base name is recorded: generated output must not depend on the
	// path the XML happened to be read from.
	s.source = filepath.Base(path)
	return nil
}

// Generate creates Go source code for the parsed protocol
func (s *Scanner) Generate(packageName string) ([]byte, error) {
	if s.protocol == nil {
		return nil, fmt.Errorf("no protocol parsed")
	}

	data, err := s.prepareTemplateData(packageName)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	tmpl := template.Must(template.New("protocol").Parse(protocolTemplate))
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return buf.Bytes(), fmt.Errorf("failed to format generated code: %w", err)
	}
	return formatted, nil
}

// templateData holds data for template generation
type templateData struct {
	Package    string
	Protocol   string
	Source     string
	Imports    []string
	Constants  []constantData
	Interfaces []interfaceData
}

type interfaceData struct {
	Name     string
	GoName   string
	Version  int
	Requests []requestData
	Events   []eventData
}

type requestData struct {
	Name            string
	GoName          string
	Doc             string
	Opcode          int
	Destructor      bool
	Params          string
	Results         string
	ArgPreparations []string
	ArgExprs        []string
	CreatesChild    bool
	ChildVar        string
	ChildType       string
	ErrorReturn     string
	HasFDs          bool
	SendCall        string
}

type eventData struct {
	Name          string
	GoName        string
	Opcode        int
	HandlerField  string
	HandlerParams string
	HandlerArgs   string
	DecodeLines   []string
}

type constantData struct {
	Name  string
	Value string
	Type  string
	Enum  string
	Entry string
}

// knownTransportTypes are the wl_ interfaces the transport ships typed
// wrappers for. Other wl_ interfaces are handled as generic objects.
var knownTransportTypes = map[string]bool{
	"Compositor": true,
	"Output":     true,
	"Pointer":    true,
	"Keyboard":   true,
	"Region":     true,
	"Seat":       true,
	"Surface":    true,
	"Touch":      true,
}

// goKeywords are avoided as parameter names because generated identifiers
// must not collide with the language.
var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

func (s *Scanner) prepareTemplateData(packageName string) (templateData, error) {
	data := templateData{
		Package:  packageName,
		Protocol: s.protocol.Name,
		Source:   s.source,
		Imports:  []string{TransportImport},
	}

	for _, iface := range s.protocol.Interfaces {
		ifaceData, err := s.processInterface(iface)
		if err != nil {
			return data, err
		}
		data.Interfaces = append(data.Interfaces, ifaceData)

		for _, enum := range iface.Enums {
			for _, entry := range enum.Entries {
				data.Constants = append(data.Constants, constantData{
					Name:  s.toConstantName(iface.Name, enum.Name, entry.Name),
					Value: entry.Value,
					Type:  "int32",
					Enum:  enum.Name,
					Entry: entry.Name,
				})
			}
		}
	}

	return data, nil
}

func (s *Scanner) processInterface(iface Interface) (interfaceData, error) {
	data := interfaceData{
		Name:    iface.Name,
		GoName:  s.toGoName(iface.Name),
		Version: iface.Version,
	}

	for i, req := range iface.Requests {
		request, err := s.processRequest(iface, req, i)
		if err != nil {
			return data, err
		}
		data.Requests = append(data.Requests, request)
	}

	for i, event := range iface.Events {
		eventData, err := s.processEvent(event, i)
		if err != nil {
			return data, err
		}
		data.Events = append(data.Events, eventData)
	}

	return data, nil
}

func (s *Scanner) processRequest(iface Interface, req Request, opcode int) (requestData, error) {
	data := requestData{
		Name:       req.Name,
		GoName:     s.toGoName(req.Name),
		Doc:        s.formatDescription(req.Description),
		Opcode:     opcode,
		Destructor: req.Type == "destructor",
		ChildVar:   "child",
	}

	var (
		params   []string
		argExprs []string
		fdExprs  []string
		newIDArg *Arg
	)

	for i, arg := range req.Args {
		if arg.Type == "new_id" {
			if newIDArg != nil {
				return data, fmt.Errorf("%s.%s: more than one new_id argument is not supported", iface.Name, req.Name)
			}
			newIDArg = &arg
			continue
		}

		name := s.paramName(arg.Name, i)
		goType, err := s.goTypeForArg(arg)
		if err != nil {
			return data, fmt.Errorf("%s.%s: %w", iface.Name, req.Name, err)
		}

		switch arg.Type {
		case "object":
			// Objects are pointer types, and travel as wl.Object so a nil
			// pointer is never sent as a non-nil interface holding a typed nil.
			params = append(params, name+" *"+goType)
			holder := fmt.Sprintf("arg%d", i)
			data.ArgPreparations = append(data.ArgPreparations,
				fmt.Sprintf("var %s wl.Object", holder),
				fmt.Sprintf("if %s != nil {", name),
				fmt.Sprintf("\t%s = %s", holder, name),
				"}")
			argExprs = append(argExprs, holder)

		case "fd":
			// Descriptors travel out of band: the body carries no value for
			// them, and the descriptor list is attached to the request.
			params = append(params, name+" int")
			fdExprs = append(fdExprs, name)
			argExprs = append(argExprs, "uintptr("+name+")")

		default:
			params = append(params, name+" "+goType)
			argExprs = append(argExprs, name)
		}
	}

	if newIDArg != nil {
		childType, err := s.goTypeForArg(*newIDArg)
		if err != nil {
			return data, fmt.Errorf("%s.%s: %w", iface.Name, req.Name, err)
		}
		data.CreatesChild = true
		data.ChildType = childType
		argExprs = append(argExprs, data.ChildVar)
	}

	data.Params = strings.Join(params, ", ")
	data.ArgExprs = argExprs
	args := ""
	if len(argExprs) > 0 {
		args = ", " + strings.Join(argExprs, ", ")
	}
	if len(fdExprs) > 0 {
		data.HasFDs = true
		data.SendCall = fmt.Sprintf("SendRequestWithFDs(o, %d, []int{%s}%s)", opcode, strings.Join(fdExprs, ", "), args)
	} else {
		data.SendCall = fmt.Sprintf("SendRequest(o, %d%s)", opcode, args)
	}
	if data.CreatesChild {
		data.Results = "(*" + data.ChildType + ", error)"
		data.ErrorReturn = "nil, err"
	} else {
		data.Results = "error"
		data.ErrorReturn = "err"
	}

	return data, nil
}

func (s *Scanner) processEvent(event Event, opcode int) (eventData, error) {
	data := eventData{
		Name:         event.Name,
		GoName:       s.toGoName(event.Name),
		Opcode:       opcode,
		HandlerField: "on" + s.toGoName(event.Name),
	}

	var params, args []string

	for i, arg := range event.Args {
		name := s.paramName(arg.Name, i)

		switch arg.Type {
		case "new_id":
			childType, err := s.goTypeForArg(arg)
			if err != nil {
				return data, fmt.Errorf("%s.%s: %w", event.Name, arg.Name, err)
			}
			child := name + "Object"
			data.DecodeLines = append(data.DecodeLines,
				fmt.Sprintf("%sID := event.Uint32()", name),
				fmt.Sprintf("%s := &%s{}", child, childType),
				fmt.Sprintf("%s.SetContext(o.Context())", child),
				fmt.Sprintf("%s.SetID(%sID)", child, name),
				fmt.Sprintf("o.Context().Register(%s)", child),
			)
			params = append(params, child+" *"+childType)
			args = append(args, child)

		case "object":
			// Object references are reported as raw object IDs: the generated
			// layer does not own a registry of foreign proxies.
			data.DecodeLines = append(data.DecodeLines, fmt.Sprintf("%sID := event.Uint32()", name))
			params = append(params, name+"ID uint32")
			args = append(args, name+"ID")

		default:
			goType, decoder, err := s.scalarDecoder(arg)
			if err != nil {
				return data, fmt.Errorf("%s.%s: %w", event.Name, arg.Name, err)
			}
			data.DecodeLines = append(data.DecodeLines, fmt.Sprintf("%s := event.%s()", name, decoder))
			params = append(params, name+" "+goType)
			args = append(args, name)
		}
	}

	data.HandlerParams = strings.Join(params, ", ")
	data.HandlerArgs = strings.Join(args, ", ")
	return data, nil
}

// scalarDecoder maps a simple argument to its Go type and event decoder.
func (s *Scanner) scalarDecoder(arg Arg) (goType, decoder string, err error) {
	switch arg.Type {
	case "int":
		return "int32", "Int32", nil
	case "uint":
		return "uint32", "Uint32", nil
	case "fixed":
		return "wl.Fixed", "Fixed", nil
	case "string":
		return "string", "String", nil
	case "array":
		return "[]byte", "Array", nil
	case "fd":
		return "uintptr", "Fd", nil
	}
	return "", "", fmt.Errorf("unsupported event argument type %q", arg.Type)
}

// goTypeForArg returns the Go type of an object-like argument. Protocol-local
// interfaces map to generated types, known core interfaces to transport
// wrappers, and everything else to the generic object interface.
func (s *Scanner) goTypeForArg(arg Arg) (string, error) {
	switch arg.Type {
	case "int":
		return "int32", nil
	case "uint":
		return "uint32", nil
	case "fixed":
		return "wl.Fixed", nil
	case "string":
		return "string", nil
	case "array":
		return "[]byte", nil
	case "fd":
		return "int", nil
	case "object", "new_id":
		return s.goTypeForInterface(arg.Interface), nil
	}
	return "", fmt.Errorf("unsupported argument type %q", arg.Type)
}

func (s *Scanner) goTypeForInterface(iface string) string {
	if iface == "" {
		return "wl.BaseProxy"
	}
	if strings.HasPrefix(iface, "wl_") {
		goName := s.toGoName(iface)
		if knownTransportTypes[goName] {
			return "wl." + goName
		}
		return "wl.BaseProxy"
	}
	return s.toGoName(iface)
}

// paramName turns a protocol argument name into a Go parameter name.
func (s *Scanner) paramName(name string, index int) string {
	if name == "" {
		return fmt.Sprintf("arg%d", index)
	}

	parts := strings.Split(name, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i == 0 {
			parts[i] = strings.ToLower(part[:1]) + part[1:]
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	result := strings.Join(parts, "")
	if result == "" {
		return fmt.Sprintf("arg%d", index)
	}
	if goKeywords[result] {
		return result + "Arg"
	}
	return result
}

func (s *Scanner) toGoName(name string) string {
	name = strings.TrimPrefix(name, "zwlr_")
	name = strings.TrimPrefix(name, "zwp_")
	name = strings.TrimPrefix(name, "wl_")
	name = strings.TrimSuffix(name, "_v1")
	name = strings.TrimSuffix(name, "_v2")

	parts := strings.Split(name, "_")
	for i, part := range parts {
		if part != "" {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

func (s *Scanner) toConstantName(iface, enum, entry string) string {
	join := func(value string) []string {
		parts := strings.Split(value, "_")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			if part != "" {
				out = append(out, strings.ToUpper(part))
			}
		}
		return out
	}
	return strings.Join(append(join(enum), join(entry)...), "_")
}

// formatDescription renders an argument or request description as a single
// documentation line.
func (s *Scanner) formatDescription(desc *Description) string {
	if desc == nil {
		return ""
	}

	text := strings.TrimSpace(desc.Text)
	if text == "" {
		return desc.Summary
	}

	lines := strings.Split(text, "\n")
	formatted := make([]string, 0, len(lines))
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			formatted = append(formatted, line)
		}
	}
	return strings.Join(formatted, " ")
}
