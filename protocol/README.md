# Device protocol sources

These protocol XML snapshots are committed so generation does not need system
Wayland XML or a sibling checkout. Bindings in `internal/protocols` are generated
by WLTurbo at `edb855fe490716032519a5660897092bda6913ae` (`go generate ./...`).
WLTurbo supplies the canonical core and xdg-shell bindings; local interfaces
are resolved before explicit `wl_*` cross-package mappings to `protocol/core`.

| XML | Pinned origin | SHA-256 |
| --- | --- | --- |
| `keyboard-shortcuts-inhibit-unstable-v1.xml` | wayland-protocols 1.49 | `9117d9e8ec02e9a3c3c55803b41e1227e76986f555f7c12eeb29f796fa63e69b` |
| `pointer-constraints-unstable-v1.xml` | installed wayland-protocols 1.49 | `f980fac900ba1dcfbbe97f588fc17b893926bd2b57624563653a1bfe4d035948` |
| `virtual-keyboard-unstable-v1.xml` | purego-libwayland `ad674f5256764b698efd3fe6a0389ac5cc6f81ef` | `7ad7870003ecd592cae47dc19d277a609b7f18fd7b7be012623cf3225a7294f5` |
| `wlr-output-management-unstable-v1.xml` | purego-libwayland `ad674f5256764b698efd3fe6a0389ac5cc6f81ef` | `65b0f82a6cf129bf1a1c31a2428795abd33886c15ddd5f3ad97e5922d7bdc3a7` |
| `wlr-virtual-pointer-unstable-v1.xml` | purego-libwayland `ad674f5256764b698efd3fe6a0389ac5cc6f81ef` | `3ff6d540be0bc5228195bf072bde42117ea17945a5c2061add5d3cf97d6bb524` |
