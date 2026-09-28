// Fixture window for the libwldevices-go consumer smoke test.
//
// The fixture is a real Wayland client living on the consumer's own
// connection: it maps an xdg_toplevel with a known title and app id, attaches
// an shm buffer and commits it, and then records every wl_pointer and
// wl_keyboard event the compositor delivers to it. Each observation is printed
// as one machine-readable FIXTURE_EVENT line so the harness output proves that
// injected input reached a client, not merely that requests were accepted.
//
// WLTurbo generated core bindings decode pointer and keyboard events.
package main

import (
	"fmt"
	"sync"
	"syscall"
	"time"

	"github.com/bnema/libwldevices-go/internal/client"
	"github.com/bnema/wlturbo/protocol/core"
	"github.com/bnema/wlturbo/protocol/xdgshell"
	"github.com/bnema/wlturbo/wl"
)

const (
	fixtureTitle = "libwldevices-fixture"
	fixtureAppID = "dev.bnema.libwldevices-go.fixture"

	// Buffer dimensions requested from the compositor are clamped to this
	// bound, which is also the pool capacity: two full-size buffers fit.
	fixtureMaxW = 1280
	fixtureMaxH = 800

	// A compositor may configure a toplevel with a zero size to let the client
	// choose. Sway does this before the first buffer, so the fixture commits
	// this fallback size and lets the compositor reconfigure it afterwards.
	fixtureFallbackW = 512
	fixtureFallbackH = 512

	fixtureWait = 10 * time.Second
	fixturePoll = 20 * time.Millisecond
)

// fixture observes the input events a mapped toplevel receives.
type fixture struct {
	mu  sync.Mutex
	ctx *wl.Context

	surface    *core.Surface
	xdgSurface *xdgshell.XdgSurface
	toplevel   *xdgshell.XdgToplevel
	seat       *core.Seat
	pointer    *core.Pointer
	keyboard   *core.Keyboard

	// shm pool backing the toplevel buffer.
	pool       *core.ShmPool
	poolData   []byte
	poolOffset int

	seatCaps uint32

	// Configure bookkeeping: an xdg_surface configure serial and a
	// xdg_toplevel size can arrive in either order, and a buffer may only be
	// committed once both are known.
	pendingSerial uint32
	serialPending bool
	configureW    int32
	configureH    int32

	buffer  *core.Buffer
	bufferW int32
	bufferH int32
	mapped  bool

	err error

	// Observations, all guarded by mu.
	pointerEnter   int
	pointerLeave   int
	pointerMotion  int
	pointerButtonP int
	pointerButtonR int
	pointerAxis    int
	keyboardEnter  int
	keyboardLeave  int
	keyboardKeyP   int
	keyboardKeyR   int
	keyboardMods   int
}

type observations struct {
	pointerEnter   int
	pointerLeave   int
	pointerMotion  int
	pointerButtonP int
	pointerButtonR int
	pointerAxis    int
	keyboardEnter  int
	keyboardLeave  int
	keyboardKeyP   int
	keyboardKeyR   int
	keyboardMods   int
}

// startFixture builds and maps the fixture toplevel on the consumer's
// connection and returns it once the compositor has accepted a buffer.
func startFixture(c *client.Client) (*fixture, error) {
	registry := c.GetRegistry()
	ctx := c.GetContext()

	f := &fixture{ctx: ctx}

	compositor := core.NewCompositor(ctx)
	if _, err := registry.BindNegotiated("wl_compositor", 6, compositor); err != nil {
		return nil, fmt.Errorf("bind wl_compositor: %w", err)
	}

	surface, err := compositor.CreateSurface()
	if err != nil {
		return nil, fmt.Errorf("create wl_surface: %w", err)
	}
	f.surface = surface

	wmBase := xdgshell.NewXdgWmBase(ctx)
	if _, err := registry.BindNegotiated("xdg_wm_base", 6, wmBase); err != nil {
		return nil, fmt.Errorf("bind xdg_wm_base: %w", err)
	}
	wmBase.OnPing(func(serial uint32) {
		// Ignoring a ping lets the compositor declare the client unresponsive.
		_ = wmBase.Pong(serial)
	})

	xdgSurface, err := wmBase.GetXdgSurface(surface)
	if err != nil {
		return nil, fmt.Errorf("get xdg_surface: %w", err)
	}
	f.xdgSurface = xdgSurface

	toplevel, err := xdgSurface.GetToplevel()
	if err != nil {
		return nil, fmt.Errorf("get xdg_toplevel: %w", err)
	}
	f.toplevel = toplevel

	if err := toplevel.SetTitle(fixtureTitle); err != nil {
		return nil, fmt.Errorf("set title: %w", err)
	}
	if err := toplevel.SetAppId(fixtureAppID); err != nil {
		return nil, fmt.Errorf("set app id: %w", err)
	}

	xdgSurface.OnConfigure(f.noteXdgConfigure)
	toplevel.OnConfigure(f.noteToplevelConfigure)

	// The seat is bound separately so the generated core bindings can be used
	// to create a wl_pointer/wl_keyboard whose dispatch decodes events.
	seat := core.NewSeat(ctx)
	if _, err := registry.BindNegotiated("wl_seat", 7, seat); err != nil {
		return nil, fmt.Errorf("bind wl_seat: %w", err)
	}
	seat.OnCapabilities(func(caps uint32) {
		f.mu.Lock()
		f.seatCaps = caps
		f.emitLocked(fmt.Sprintf("seat_capabilities caps=%#x", caps))
		f.mu.Unlock()
	})
	f.seat = seat

	// WLTurbo's generated shm binding owns the concrete pool and its FD transfer.
	shm := core.NewShm(ctx)
	if _, err := registry.BindNegotiated("wl_shm", 1, shm); err != nil {
		return nil, fmt.Errorf("bind wl_shm: %w", err)
	}

	poolSize := fixtureMaxW * fixtureMaxH * 4 * 2
	fd, err := wl.CreateAnonymousFile(int64(poolSize))
	if err != nil {
		return nil, fmt.Errorf("create shm file: %w", err)
	}
	data, err := wl.MapMemory(fd, poolSize)
	if err != nil {
		_ = syscall.Close(fd)
		return nil, fmt.Errorf("map shm file: %w", err)
	}
	f.poolData = data

	pool, err := shm.CreatePool(fd, int32(poolSize))
	if err != nil {
		_ = wl.UnmapMemory(data)
		_ = syscall.Close(fd)
		return nil, fmt.Errorf("create shm pool: %w", err)
	}

	f.pool = pool

	// Commit an empty surface to receive the first configure, then a buffer.
	if err := surface.Commit(); err != nil {
		return nil, fmt.Errorf("initial commit: %w", err)
	}

	if !f.pumpUntil(c, "window mapped", func() bool { return f.isMapped() }) {
		return f, fmt.Errorf("fixture window did not map: %v", f.err)
	}
	fmt.Printf("FIXTURE_NOTE mapped title=%q app_id=%q buffer=%dx%d\n",
		fixtureTitle, fixtureAppID, f.bufferW, f.bufferH)
	return f, nil
}

func (f *fixture) isMapped() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.mapped
}

func (f *fixture) snapshot() observations {
	f.mu.Lock()
	defer f.mu.Unlock()
	return observations{
		pointerEnter:   f.pointerEnter,
		pointerMotion:  f.pointerMotion,
		pointerButtonP: f.pointerButtonP,
		pointerButtonR: f.pointerButtonR,
		pointerAxis:    f.pointerAxis,
		keyboardEnter:  f.keyboardEnter,
		keyboardLeave:  f.keyboardLeave,
		keyboardKeyP:   f.keyboardKeyP,
		keyboardKeyR:   f.keyboardKeyR,
		keyboardMods:   f.keyboardMods,
	}
}

// err returns the first recorded error, if any.
func (f *fixture) fixtureErr() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.err
}

// pumpUntil dispatches events until cond is true or the timeout expires. Each
// iteration performs a roundtrip, which gives the compositor a chance to process
// the injected input on its own (separate) connection and deliver it here.
func (f *fixture) pumpUntil(c *client.Client, what string, cond func() bool) bool {
	deadline := time.Now().Add(fixtureWait)
	for {
		if err := f.fixtureErr(); err != nil {
			return false
		}
		if cond() {
			return true
		}
		if !time.Now().Before(deadline) {
			return false
		}
		if err := c.GetDisplay().Roundtrip(); err != nil {
			f.mu.Lock()
			if f.err == nil {
				f.err = fmt.Errorf("roundtrip while waiting for %s: %w", what, err)
			}
			f.mu.Unlock()
			return false
		}
		time.Sleep(fixturePoll)
	}
}

// waitPointerCapability waits until the seat advertises the pointer capability.
func (f *fixture) waitPointerCapability(c *client.Client) bool {
	return f.pumpUntil(c, "wl_pointer capability", func() bool {
		f.mu.Lock()
		defer f.mu.Unlock()
		return f.seatCaps&core.CAPABILITY_POINTER != 0
	})
}

// waitKeyboardCapability waits until the seat advertises the keyboard capability.
func (f *fixture) waitKeyboardCapability(c *client.Client) bool {
	return f.pumpUntil(c, "wl_keyboard capability", func() bool {
		f.mu.Lock()
		defer f.mu.Unlock()
		return f.seatCaps&core.CAPABILITY_KEYBOARD != 0
	})
}

// bindPointer creates a wl_pointer object and registers its event handlers.
func (f *fixture) bindPointer() error {
	pointer, err := f.seat.GetPointer()
	if err != nil {
		return fmt.Errorf("wl_seat.get_pointer: %w", err)
	}

	pointer.OnEnter(func(serial uint32, surfaceID uint32, x wl.Fixed, y wl.Fixed) {
		f.mu.Lock()
		f.pointerEnter++
		f.emitLocked(fmt.Sprintf("pointer_enter serial=%d surface=%d x=%.1f y=%.1f", serial, surfaceID, x.Float64(), y.Float64()))
		f.mu.Unlock()
	})
	pointer.OnLeave(func(serial uint32, surfaceID uint32) {
		f.mu.Lock()
		f.pointerLeave++
		f.emitLocked(fmt.Sprintf("pointer_leave serial=%d surface=%d", serial, surfaceID))
		f.mu.Unlock()
	})
	pointer.OnMotion(func(timeMS uint32, x wl.Fixed, y wl.Fixed) {
		f.mu.Lock()
		f.pointerMotion++
		f.emitLocked(fmt.Sprintf("pointer_motion time=%d x=%.1f y=%.1f", timeMS, x.Float64(), y.Float64()))
		f.mu.Unlock()
	})
	pointer.OnButton(func(serial uint32, timeMS uint32, button uint32, state uint32) {
		f.mu.Lock()
		if state == 1 {
			f.pointerButtonP++
		} else {
			f.pointerButtonR++
		}
		f.emitLocked(fmt.Sprintf("pointer_button button=%d state=%s", button, buttonState(state)))
		f.mu.Unlock()
	})
	pointer.OnAxis(func(timeMS uint32, axis uint32, value wl.Fixed) {
		f.mu.Lock()
		f.pointerAxis++
		f.emitLocked(fmt.Sprintf("pointer_axis axis=%d value=%.2f", axis, value.Float64()))
		f.mu.Unlock()
	})

	f.pointer = pointer
	return nil
}

// bindKeyboard creates a wl_keyboard object and registers its event handlers.
func (f *fixture) bindKeyboard() error {
	keyboard, err := f.seat.GetKeyboard()
	if err != nil {
		return fmt.Errorf("wl_seat.get_keyboard: %w", err)
	}
	keyboard.OnKeymap(func(_ uint32, fd *wl.OwnedFD, _ uint32) { _ = fd.Close() })

	keyboard.OnEnter(func(serial uint32, surfaceID uint32, keys []byte) {
		f.mu.Lock()
		f.keyboardEnter++
		f.emitLocked(fmt.Sprintf("keyboard_enter serial=%d surface=%d keys=%d", serial, surfaceID, len(keys)))
		f.mu.Unlock()
	})
	keyboard.OnLeave(func(serial uint32, surfaceID uint32) {
		f.mu.Lock()
		f.keyboardLeave++
		f.emitLocked(fmt.Sprintf("keyboard_leave serial=%d surface=%d", serial, surfaceID))
		f.mu.Unlock()
	})
	keyboard.OnKey(func(serial uint32, timeMS uint32, key uint32, state uint32) {
		f.mu.Lock()
		if state == 1 {
			f.keyboardKeyP++
		} else {
			f.keyboardKeyR++
		}
		f.emitLocked(fmt.Sprintf("keyboard_key key=%d state=%s", key, buttonState(state)))
		f.mu.Unlock()
	})
	keyboard.OnModifiers(func(serial uint32, depressed uint32, latched uint32, locked uint32, group uint32) {
		f.mu.Lock()
		f.keyboardMods++
		f.emitLocked(fmt.Sprintf("keyboard_modifiers depressed=%#x locked=%#x group=%d", depressed, locked, group))
		f.mu.Unlock()
	})

	f.keyboard = keyboard
	return nil
}

// emitLocked prints one machine-readable observation line. mu must be held.
func (f *fixture) emitLocked(line string) {
	fmt.Printf("FIXTURE_EVENT %s\n", line)
}

func (f *fixture) noteXdgConfigure(serial uint32) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pendingSerial = serial
	f.serialPending = true
	f.applyConfigureLocked()
}

func (f *fixture) noteToplevelConfigure(width int32, height int32, states []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if width > 0 && height > 0 {
		f.configureW, f.configureH = width, height
	}
	f.applyConfigureLocked()
}

// applyConfigureLocked acks the pending configure and commits a buffer so the
// compositor maps the surface. A zero configure size means the client picks,
// so the fixture falls back to a fixed size and lets a later configure resize
// it. mu must be held.
func (f *fixture) applyConfigureLocked() {
	if f.err != nil || !f.serialPending {
		return
	}
	width, height := f.configureW, f.configureH
	if width <= 0 {
		width = fixtureFallbackW
	}
	if height <= 0 {
		height = fixtureFallbackH
	}
	if width > fixtureMaxW {
		width = fixtureMaxW
	}
	if height > fixtureMaxH {
		height = fixtureMaxH
	}
	if f.buffer == nil || f.bufferW != width || f.bufferH != height {
		if err := f.createBufferLocked(width, height); err != nil {
			f.err = err
			return
		}
	}
	if err := f.xdgSurface.AckConfigure(f.pendingSerial); err != nil {
		f.err = fmt.Errorf("ack configure: %w", err)
		return
	}
	f.serialPending = false
	if err := f.surface.Attach(f.buffer, 0, 0); err != nil {
		f.err = fmt.Errorf("attach buffer: %w", err)
		return
	}
	if err := f.surface.Damage(0, 0, width, height); err != nil {
		f.err = fmt.Errorf("damage buffer: %w", err)
		return
	}
	if err := f.surface.Commit(); err != nil {
		f.err = fmt.Errorf("commit buffer: %w", err)
		return
	}
	f.mapped = true
}

func (f *fixture) createBufferLocked(width, height int32) error {
	stride := width * 4
	size := int(stride) * int(height)
	if f.poolOffset+size > len(f.poolData) {
		return fmt.Errorf("shm pool exhausted: need %d bytes at offset %d of %d", size, f.poolOffset, len(f.poolData))
	}
	buffer, err := f.pool.CreateBuffer(int32(f.poolOffset), width, height, stride, core.FORMAT_XRGB8888)
	if err != nil {
		return fmt.Errorf("create wl_buffer: %w", err)
	}
	// Fill the buffer with opaque mid-grey so a page flip commits real content.
	for i := f.poolOffset; i < f.poolOffset+size; i += 4 {
		f.poolData[i] = 0x80
		f.poolData[i+1] = 0x80
		f.poolData[i+2] = 0x80
		f.poolData[i+3] = 0xff
	}
	f.poolOffset += size
	f.buffer = buffer
	f.bufferW, f.bufferH = width, height
	return nil
}

func buttonState(state uint32) string {
	if state == 1 {
		return "press"
	}
	return "release"
}
