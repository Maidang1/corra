# Native Rich Editor Implementation Plan

> **For agentic workers:** Use superpowers:subagent-driven-development for bounded native/model implementation with independent review.

**Goal:** Native WYSIWYG default on macOS while keeping Markdown source and reliable saves.
**Architecture:** Go rich document codec, AppKit editor behind platform bridge, shared views command/save routing.
**Tech Stack:** Go, cgo/Objective-C AppKit, Goldmark, MyGo.

## Progress
- Task 1 implemented and spec-reviewed: exact-source codec, blank paragraphs,
  UTF-16, protected blocks, adjacency and partial-formatting regression coverage.
- Task 2 implemented and reviewed: real AppKit harness green; clipboard/focus
  followups fixed. No Accessibility or real IME candidate automation.
- Task 3 implemented: rich/source/reading modes, native save routing, modal
  visibility, command forwarding and cgo-enabled make/release configuration.
- Task 4: independent spec and quality reviews passed; light/dark native
  captures inspected. Full verification rerun after invalid-byte/link fixes:
  213 tests, 188 race cases, 205 fallback cases; vet/build/signature check passed.
  Native dev build running with cgo; interactive IME/full-window acceptance
  remains explicitly manual (Accessibility/screen capture not available).

Existing uncommitted Notes UI work is retained. Work continues on
`codex/native-rich-editor`, without commits or resetting the user's workspace.

### Task 1: Rich document codec
Files richtext/*.go. Tests first for Markdown parsing/serialization and UTF16 spans.
Opaque constructs preserve exact source; unchanged documents round-trip exactly.
Implement exported JSON-friendly document model and codec. Run go test ./richtext.

### Task 2: Native editor bridge
Files nativerich/* only. Implement NSTextView rich attributes, list return/checklist,
native undo, IME-safe callbacks, layout/visibility and opaque source preservation.
Add platform stubs and native deterministic harness tests. Run package tests.

### Task 3: View/app integration
Files views/*, app/state.go, main.go, menu.go, mygo.json if build needed.
Interface-driven native routing; default rich mode only when attached; preserve
source/preview fallback. Shared toolbar formatting/insert, save sync, mode switching,
hide over dialogs. Add failing view integration tests then implement.

### Task 4: Verify
Run go test ./..., race changed packages, go vet/build, packaged native build.
Native harness verifies actual NSTextView operations (not merely fake callbacks).
Update docs and running dev app; explicitly distinguish manual IME verification.
