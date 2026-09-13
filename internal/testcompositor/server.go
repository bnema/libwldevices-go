// Package testcompositor provides a minimal Wayland compositor for tests.
//
// It listens on a private Unix socket, answers the core handshake that a client
// needs to enumerate globals and complete roundtrips, records every request it
// receives, and lets a test push arbitrary events or protocol errors back to
// the client. Tests therefore exercise real framing and real protocol
// bookkeeping without a live compositor and without a display server.
package testcompositor

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

const (
	displayObjectID = 1

	// HeaderSize is the fixed Wayland message header size.
	HeaderSize = 8
)

// Global describes one wl_registry global the server announces.
type Global struct {
	Name      uint32
	Interface string
	Version   uint32
}

// Request is one recorded client request.
type Request struct {
	Object uint32
	Opcode uint16
	Body   []byte
	FDs    []int
}

// Uint32 decodes a uint32 argument at the given byte offset.
func (r Request) Uint32(offset int) uint32 {
	if offset+4 > len(r.Body) {
		return 0
	}
	return binary.LittleEndian.Uint32(r.Body[offset : offset+4])
}

// String decodes a Wayland string at the given byte offset and returns it with
// the number of bytes it occupied.
func (r Request) String(offset int) (string, int) {
	if offset+4 > len(r.Body) {
		return "", 0
	}
	length := int(binary.LittleEndian.Uint32(r.Body[offset : offset+4]))
	if length <= 0 || offset+4+length > len(r.Body) {
		return "", 0
	}
	value := string(r.Body[offset+4 : offset+4+length-1])
	consumed := 4 + length
	if pad := consumed % 4; pad != 0 {
		consumed += 4 - pad
	}
	return value, consumed
}

// Server is a minimal Wayland compositor serving one connection at a time.
type Server struct {
	runtimeDir string
	socketName string

	listener net.Listener

	mu       sync.Mutex
	conn     net.Conn
	requests []Request
	ids      map[string]uint32 // interface name -> bound object ID
	names    map[uint32]string // bound object ID -> interface name
	globals  []Global
	received []int // every descriptor the client sent, closed on shutdown
	registry uint32
	err      error
	closed   bool

	writeMu sync.Mutex
}

// Start launches a compositor that announces the given globals. The socket is
// removed and the listener closed when the test finishes.
func Start(t *testing.T, globals ...Global) *Server {
	t.Helper()

	runtimeDir := t.TempDir()
	socketName := "wayland-test"
	socketPath := filepath.Join(runtimeDir, socketName)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("testcompositor: listen %s: %v", socketPath, err)
	}

	s := &Server{
		runtimeDir: runtimeDir,
		socketName: socketName,
		listener:   listener,
		ids:        make(map[string]uint32),
		names:      make(map[uint32]string),
		globals:    globals,
	}

	go s.serve()

	t.Cleanup(s.Close)
	return s
}

// Env points the Wayland client library at this compositor for the duration of
// the test.
func (s *Server) Env(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_RUNTIME_DIR", s.runtimeDir)
	t.Setenv("WAYLAND_DISPLAY", s.socketName)
}

// SocketPath returns the compositor socket path.
func (s *Server) SocketPath() string {
	return filepath.Join(s.runtimeDir, s.socketName)
}

// Requests returns a copy of every request received so far.
func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Request, len(s.requests))
	copy(out, s.requests)
	return out
}

// WaitForRequests blocks until at least n requests have been recorded or the
// deadline expires, then returns everything recorded.
func (s *Server) WaitForRequests(t *testing.T, n int) []Request {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		requests := s.Requests()
		if len(requests) >= n {
			return requests
		}
		if time.Now().After(deadline) {
			t.Fatalf("testcompositor: waited for %d requests, saw %d: %+v", n, len(requests), requests)
		}
		time.Sleep(time.Millisecond)
	}
}

// WaitForRequest blocks until a request for the named interface and opcode has
// been received, then returns it.
func (s *Server) WaitForRequest(t *testing.T, iface string, opcode uint16) Request {
	t.Helper()

	id := s.ObjectID(iface)
	if id == 0 {
		t.Fatalf("testcompositor: interface %q was never bound", iface)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		for _, req := range s.Requests() {
			if req.Object == id && req.Opcode == opcode {
				return req
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("testcompositor: no request object=%d opcode=%d for %q: %+v", id, opcode, iface, s.Requests())
		}
		time.Sleep(time.Millisecond)
	}
}

// ObjectID returns the object ID bound for an interface, or 0 when unknown.
func (s *Server) ObjectID(iface string) uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ids[iface]
}

// RegistryID returns the object ID the client used in get_registry, or 0.
func (s *Server) RegistryID() uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.registry
}

// InterfaceFor returns the interface bound to an object ID, or "".
func (s *Server) InterfaceFor(id uint32) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.names[id]
}

// SendEvent pushes one event to the client.
func (s *Server) SendEvent(object uint32, opcode uint16, args ...any) error {
	body, err := marshal(args...)
	if err != nil {
		return err
	}
	return s.write(message(object, opcode, body))
}

// SendDisplayError reports a protocol error to the client.
func (s *Server) SendDisplayError(object uint32, code uint32, text string) error {
	return s.SendEvent(displayObjectID, 0, object, code, text)
}

// Close stops the compositor and removes its socket.
func (s *Server) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	conn := s.conn
	s.mu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}
	_ = s.listener.Close()
	_ = os.Remove(s.SocketPath())

	s.mu.Lock()
	for _, fd := range s.received {
		_ = unix.Close(fd)
	}
	s.received = nil
	s.mu.Unlock()
}

// Err reports the first unexpected server-side error.
func (s *Server) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

func (s *Server) serve() {
	conn, err := s.listener.Accept()
	if err != nil {
		return // listener closed during cleanup
	}

	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()

	for {
		object, opcode, body, fds, err := readMessage(conn)
		if err != nil {
			return // client closed, or the test finished
		}

		s.mu.Lock()
		s.requests = append(s.requests, Request{Object: object, Opcode: opcode, Body: body, FDs: fds})
		s.received = append(s.received, fds...)
		registry := s.registry
		s.mu.Unlock()

		if err := s.handle(conn, object, opcode, body, registry); err != nil {
			return
		}
	}
}

func (s *Server) handle(conn net.Conn, object uint32, opcode uint16, body []byte, registry uint32) error {
	switch {
	case object == displayObjectID && opcode == 0: // sync
		callback := readUint32(body, 0)
		if err := s.write(message(callback, 0, marshalRaw(uint32(1)))); err != nil {
			return err
		}
		return s.write(message(displayObjectID, 1, marshalRaw(callback)))

	case object == displayObjectID && opcode == 1: // get_registry
		id := readUint32(body, 0)
		s.mu.Lock()
		s.registry = id
		globals := append([]Global(nil), s.globals...)
		s.mu.Unlock()

		for _, global := range globals {
			args, err := marshal(global.Name, global.Interface, global.Version)
			if err != nil {
				return err
			}
			if err := s.write(message(id, 0, args)); err != nil {
				return err
			}
		}
		return nil

	case registry != 0 && object == registry && opcode == 0: // wl_registry.bind
		name := readUint32(body, 0)
		iface, consumed := (Request{Body: body}).String(4)
		version := readUint32(body, 4+consumed)
		id := readUint32(body, 8+consumed)

		s.mu.Lock()
		s.ids[iface] = id
		s.names[id] = iface
		s.mu.Unlock()

		_ = name
		_ = version
		return nil
	}

	return nil
}

func (s *Server) write(msg []byte) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()

	if conn == nil {
		return nil
	}
	_, err := conn.Write(msg)
	return err
}

// readMessage reads one complete Wayland message from conn, collecting any
// ancillary file descriptors sent with it.
func readMessage(conn net.Conn) (object uint32, opcode uint16, body []byte, fds []int, err error) {
	var hdr [HeaderSize]byte

	if unixConn, ok := conn.(*net.UnixConn); ok {
		oob := make([]byte, unix.CmsgSpace(16*4))
		n, oobn, _, _, rerr := unixConn.ReadMsgUnix(hdr[:], oob)
		if rerr != nil {
			return 0, 0, nil, nil, rerr
		}
		if n < HeaderSize {
			return 0, 0, nil, nil, io.ErrUnexpectedEOF
		}
		fds, err = parseFDs(oob[:oobn])
		if err != nil {
			return 0, 0, nil, fds, err
		}
	} else if _, err = io.ReadFull(conn, hdr[:]); err != nil {
		return 0, 0, nil, nil, err
	}

	object = binary.LittleEndian.Uint32(hdr[0:4])
	sizeOpcode := binary.LittleEndian.Uint32(hdr[4:8])
	size := sizeOpcode >> 16
	opcode = uint16(sizeOpcode & 0xffff)

	if size < HeaderSize || size%4 != 0 {
		return 0, 0, nil, fds, fmt.Errorf("testcompositor: malformed header: size=%d opcode=%d", size, opcode)
	}

	body = make([]byte, size-HeaderSize)
	if len(body) > 0 {
		if _, err := io.ReadFull(conn, body); err != nil {
			return 0, 0, nil, fds, err
		}
	}
	return object, opcode, body, fds, nil
}

// parseFDs extracts descriptors from socket control messages.
func parseFDs(oob []byte) ([]int, error) {
	if len(oob) == 0 {
		return nil, nil
	}

	scms, err := syscall.ParseSocketControlMessage(oob)
	if err != nil {
		return nil, fmt.Errorf("testcompositor: parse control message: %w", err)
	}

	var fds []int
	for i := range scms {
		scm := &scms[i]
		if scm.Header.Type != syscall.SCM_RIGHTS {
			continue
		}
		parsed, err := syscall.ParseUnixRights(scm)
		if err != nil {
			return fds, fmt.Errorf("testcompositor: parse unix rights: %w", err)
		}
		fds = append(fds, parsed...)
	}
	return fds, nil
}

// message builds a framed Wayland message.
func message(object uint32, opcode uint16, body []byte) []byte {
	msg := make([]byte, HeaderSize+len(body))
	binary.LittleEndian.PutUint32(msg[0:4], object)
	binary.LittleEndian.PutUint32(msg[4:8], (uint32(HeaderSize+len(body))<<16)|uint32(opcode))
	copy(msg[HeaderSize:], body)
	return msg
}

// marshalRaw wraps an already encoded body.
func marshalRaw(values ...any) []byte {
	body, err := marshal(values...)
	if err != nil {
		panic(err)
	}
	return body
}

// marshal encodes Wayland arguments the same way the client library does.
func marshal(args ...any) ([]byte, error) {
	var body []byte
	for _, arg := range args {
		switch v := arg.(type) {
		case uint32:
			body = binary.LittleEndian.AppendUint32(body, v)
		case int32:
			body = binary.LittleEndian.AppendUint32(body, uint32(v))
		case int:
			body = binary.LittleEndian.AppendUint32(body, uint32(v))
		case string:
			length := uint32(len(v) + 1)
			body = binary.LittleEndian.AppendUint32(body, length)
			body = append(body, v...)
			body = append(body, 0)
			for len(body)%4 != 0 {
				body = append(body, 0)
			}
		case []byte:
			body = binary.LittleEndian.AppendUint32(body, uint32(len(v)))
			body = append(body, v...)
			for len(body)%4 != 0 {
				body = append(body, 0)
			}
		default:
			return nil, fmt.Errorf("testcompositor: unsupported argument type %T", arg)
		}
	}
	return body, nil
}

func readUint32(body []byte, offset int) uint32 {
	if offset+4 > len(body) {
		return 0
	}
	return binary.LittleEndian.Uint32(body[offset : offset+4])
}
