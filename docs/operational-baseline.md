# LibWL Devices — Failure Baseline (Task 1, Step 2)

Evidence ledger from `/home/brice/dev/projects/libwldevices-go`, recorded before any repair.
It is not a claim that the module works; nothing was fixed.

## 1. Environment

| Item | Value |
| --- | --- |
| Commit | `cc28e01cb21de5eb7f75e2c04a7415a4ff47a470` |
| Branch | `phase1/operational-health` |
| Go toolchain | `go version go1.27.1-X:nodwarf5 linux/amd64` |
| Run date | 2026-09-13 (08:09–08:10 +02:00) |
| Module | `github.com/bnema/libwldevices-go` (`go 1.24` directive in `go.mod`) |
| Worktree | `git status --short` empty before and after the run |

All commands below were run with the display removed (`env -u WAYLAND_DISPLAY -u XDG_RUNTIME_DIR`).
`go build ./...` exits `0`.

## 2. Package inventory

14 packages exist in the module.

| Package | Test file | Compiles | Test result w/o display |
| --- | --- | --- | --- |
| `.` (root, `doc.go`) | no | yes | `[no test files]` |
| `examples/monitors_ouput` | no | yes | `[no test files]` |
| `examples/pointer_constraints` | no | yes | `[build failed]` (vet stage only) |
| `examples/virtual_keyboard` | no | yes | `[no test files]` |
| `examples/virtual_pointer` | no | yes | `[no test files]` |
| `internal/client` | no | yes | `[no test files]` |
| `internal/protocols` | no | yes | `[no test files]` |
| `keyboard_shortcuts_inhibitor` | yes (9 tests) | yes | `ok` — 9 pass |
| `output_management` | yes (16 tests) | yes | `ok` — 16 pass |
| `pointer_constraints` | yes (9 tests) | yes | `ok` — 8 pass, 1 skip |
| `scanner` | no | yes | `[no test files]` |
| `scanner/cmd/wayland-scanner` | no | yes | `[no test files]` |
| `tools` | no | yes | `[no test files]` |
| `virtual_keyboard` | yes (9 tests) | yes | `ok` — 3 pass, 6 skip |
| `virtual_pointer` | yes (9 tests) | yes | `FAIL` — 2 pass, 7 fail |

`examples/pointer_constraints` compiles under `go build`; it fails only because `go test` runs a vet subset first.

## 3. Command evidence

### `env -u WAYLAND_DISPLAY -u XDG_RUNTIME_DIR go test ./...` → exit 1

```text
# github.com/bnema/libwldevices-go/examples/pointer_constraints
examples/pointer_constraints/main.go:324:2: fmt.Println call has possible Printf formatting directive %v
examples/pointer_constraints/main.go:324:2: fmt.Println arg list ends with redundant newline
FAIL	github.com/bnema/libwldevices-go/examples/pointer_constraints [build failed]
ok  	github.com/bnema/libwldevices-go/keyboard_shortcuts_inhibitor	(cached)
... (elided: identical cached "ok" lines for output_management, pointer_constraints, virtual_keyboard)
--- FAIL: TestNewVirtualPointerManager (0.00s)
    virtual_pointer_test.go:13: Failed to create virtual pointer manager: failed to create Wayland client: failed to connect to Wayland: XDG_RUNTIME_DIR not set
--- FAIL: TestVirtualPointerCreation / Motion / Buttons / Axis / Frame / Destroy — identical message
FAIL	github.com/bnema/libwldevices-go/virtual_pointer	0.002s
FAIL
```

The two vet diagnostics are on the multi-line raw-string `fmt.Println` that begins at
line 324 (verified against `examples/pointer_constraints/main.go:318-328`). Wording,
file and line match the expected reproduction exactly.

### `go vet ./...` → exit 1

```text
examples/pointer_constraints/main.go:324:2: fmt.Println call has possible Printf formatting directive %v
examples/pointer_constraints/main.go:324:2: fmt.Println arg list ends with redundant newline
```

Only these two diagnostics exist repo-wide; `go vet ./virtual_pointer/... ./keyboard_shortcuts_inhibitor/...` exits `0`.

### `gofmt -l .` → exit 0, 22 of 25 `.go` files listed

```text
doc.go  examples/monitors_ouput/main.go  examples/pointer_constraints/main.go  examples/virtual_keyboard/main.go
internal/client/client.go  internal/protocols/output_management.go  internal/protocols/virtual_pointer.go
keyboard_shortcuts_inhibitor/keyboard_shortcuts_inhibitor.go  keyboard_shortcuts_inhibitor/keyboard_shortcuts_inhibitor_test.go
output_management/generated.go  output_management/output_management.go  output_management/output_management_test.go
pointer_constraints/pointer_constraints.go  pointer_constraints/pointer_constraints_test.go  scanner/cmd/wayland-scanner/main.go
scanner/scanner.go  scanner/template.go  tools/generate.go  virtual_keyboard/virtual_keyboard.go
virtual_keyboard/virtual_keyboard_test.go  virtual_pointer/virtual_pointer.go  virtual_pointer/virtual_pointer_test.go
```

(wrapped for width; 22 paths. Unlisted: `examples/virtual_pointer/main.go`, `internal/protocols/pointer_constraints.go`, `internal/protocols/virtual_keyboard.go`. `gofmt -d` shows real deltas — struct-field alignment, tab-only blank lines — not CRLF.)

## 4. Live-compositor tests vs. genuine unit tests

- **Compositor required, fail hard (`virtual_pointer`):** `TestNewVirtualPointerManager`, `...Creation`,
  `...Motion`, `...Buttons`, `...Axis`, `...Frame`, `...Destroy`; **no** skip guard, aborts on `XDG_RUNTIME_DIR not set`.
- **Compositor required, skip gracefully:** `TestNewPointerConstraintsManager` (`t.Skipf("Cannot test without
  Wayland: %v", err)`) and `virtual_keyboard`'s `TestNewVirtualKeyboardManager`, `...Creation`, `...Keys`,
  `...Modifiers`, `...Close`, `TestTypeString` (`t.Skipf("Skipping test - virtual keyboard manager not available: %v", err)`).
- **Genuine unit tests (pass with no display):** all 9 `keyboard_shortcuts_inhibitor`, all 16 `output_management`,
  8 of 9 `pointer_constraints` (constants, error types, nil/invalid args, close), 3 `virtual_keyboard` constant
  tests, `TestButtonConstants`/`TestAxisConstants` in `virtual_pointer`.
- Net with no display: 1 package `FAIL` (7 tests), 1 `[build failed]`, 7 tests skipped across 2 packages, 2 fully green.

## 5. Generator situation

Two unrelated generators exist, and neither is wired into `go generate`:

- `scanner/scanner.go` (495 lines) — XML protocol scanner emitting Go bindings (`go/format` +
  `text/template`); `scanner/template.go` (394 lines) — its `protocolTemplate`, header
  `// Code generated by wayland-scanner. DO NOT EDIT.`; `scanner/cmd/wayland-scanner/main.go`
  (110 lines) — CLI wrapper (`-o/--output`, `-p/--package`).
- `tools/generate.go` — older generator (`-protocol -xml -output -package`); its inline
  template emits `// Code generated by tools/generate.go. DO NOT EDIT.` and **stub** bodies
  (`// This is a stub implementation - in reality, this would: ...`).
- `internal/protocols/*.go` — low-level bindings, no `Code generated` marker; `output_management.go` and
  `virtual_pointer.go` import `github.com/bnema/wlturbo/wl`.
- `output_management/generated.go` — header `// Code generated by tools/generate.go. DO NOT EDIT.` /
  `// Source: wlr_output_management_unstable_v1`, i.e. produced by the stub template.

Regeneration commands:

```text
make generate-protocols            # only chains to generate-pointer-constraints
make generate-pointer-constraints  # requires /usr/share/wayland-protocols/... XML
go run tools/generate.go -protocol=pointer_constraints -xml=<xml> -output=<out.go> -package=protocols
go run ./scanner/cmd/wayland-scanner -o <out.go> <protocol.xml>
```

Observed run: `go generate ./...` printed **nothing**, exited `0`, and left `git status --short` and
`git diff --stat` empty. `grep -rn "go:generate" --include='*.go' .` returns nothing — the repo has
**no `//go:generate` directives**, so `go generate ./...` succeeds *vacuously* and regenerates nothing.
The real generators depend on system-installed protocol XML, so reproducibility of
`output_management/generated.go` and `internal/protocols/*.go` is **unverified** here. The tree was
restored with `git checkout -- .`; `git status --short` is empty.

## 6. WLTurbo dependency

```text
require github.com/bnema/wlturbo v0.1.0
require golang.org/x/sys v0.33.0 // indirect
```

`go list -m` confirms `github.com/bnema/wlturbo v0.1.0`; `grep -n "replace" go.mod` finds nothing — **no `replace` directive exists**, so v0.1.0 resolves from the module proxy/cache.

## 7. Legacy module path note

`grep -rn "wayland-virtual-input-go" --exclude-dir=.git .` → **zero occurrences** (exit `1`).
`github.com/bnema/wayland-virtual-input-go` appears nowhere here, so it is a different module from
`github.com/bnema/libwldevices-go` and cannot serve as a compatibility gate for this codebase.
What Waymon imports can only be checked in the Waymon repo (out of scope, not done).

## 8. Classification

| Finding | Status |
| --- | --- |
| `go vet ./...` fails at `examples/pointer_constraints/main.go:324`; `go test ./...` exits 1 | **Reproduced** |
| Compositor tests: `virtual_pointer` fails (no skip guard); `pointer_constraints`/`virtual_keyboard` skip | **Reproduced** |
| `keyboard_shortcuts_inhibitor`, `output_management` tests are display-independent | **Reproduced** |
| `examples/pointer_constraints` still compiles (`go build ./...` exit 0) | **Verified** |
| `gofmt -l .` reports 22 of 25 files unformatted | **Reproduced** |
| No `//go:generate` directives; `go generate ./...` is a no-op | **Reproduced** |
| Reproducibility of checked-in generated files | **Unverified** (generators not exercised) |
| `output_management/generated.go` came from the `tools/generate.go` stub template | **Verified** |
| `wlturbo v0.1.0`, no `replace` directive | **Verified** |
| `wayland-virtual-input-go` absent from this repo | **Verified** (0 grep hits) |
| Waymon's import path and any compatibility implication | **Suspected** (not verifiable here) |
| Root cause(s) of the failures | **Suspected** — symptoms only, no cause analysis done |
