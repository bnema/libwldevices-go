package virtual_keyboard

import (
	"fmt"
	"github.com/bnema/wlturbo/wl"
	"syscall"
)

// createDefaultKeymap creates a minimal XKB keymap file descriptor
func createDefaultKeymap() (int, uint32, error) {
	// Minimal XKB keymap
	keymap := `xkb_keymap {
	xkb_keycodes  { include "evdev+aliases(qwerty)"	};
	xkb_types     { include "complete"	};
	xkb_compat    { include "complete"	};
	xkb_symbols   { include "pc+us+inet(evdev)"	};
	xkb_geometry  { include "pc(pc105)"	};
};`

	// Create anonymous shared memory file
	size := len(keymap) + 1 // +1 for null terminator
	fd, err := wl.CreateAnonymousFile(int64(size))
	if err != nil {
		return -1, 0, err
	}

	// Map the memory
	data, err := wl.MapMemory(fd, size)
	if err != nil {
		_ = syscall.Close(fd)
		return -1, 0, err
	}
	defer func() { _ = wl.UnmapMemory(data) }()

	// Copy keymap to shared memory
	copy(data, keymap)
	data[len(keymap)] = 0 // null terminator

	// Seek to beginning for compositor to read
	_, err = syscall.Seek(fd, 0, 0)
	if err != nil {
		_ = syscall.Close(fd)
		return -1, 0, err
	}

	// Verify the fd is readable
	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil {
		_ = syscall.Close(fd)
		return -1, 0, fmt.Errorf("fstat failed: %w", err)
	}

	// Return the keymap size INCLUDING null terminator
	// Wayland expects the full mmap size including the null byte
	// Safe conversion: size is controlled and small
	if size < 0 || size > 0x7FFFFFFF {
		_ = syscall.Close(fd)
		return -1, 0, fmt.Errorf("invalid keymap size: %d", size)
	}
	return fd, uint32(size), nil
}
