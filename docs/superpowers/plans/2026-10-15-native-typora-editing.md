# Native macOS Typora Editing Implementation Plan

> For agentic workers: use superpowers:subagent-driven-development with focused ownership and test-first implementation. Do not commit or push without a user request.

**Goal:** Improve the macOS native editor toward the Typora editing acceptance inventory without WebView.

**Architecture:** Extend the existing Go Markdown model and AppKit NSTextView bridge. Structural operations preserve source for untouched blocks, edit real native text storage, retain UTF-16 selections and undo as one transaction.

**Tech stack:** Go, Goldmark, Objective-C/AppKit, MyGo v0.5.1.

## Tasks and status

- [x] Review the approved architecture and executable acceptance scope.
- [x] Extend `richtext/codec.go` with nested-list depth, exact no-edit round trips, edited-item serialization and mixed nested types. Add corpus tests under `richtext/` first; run `go test ./richtext`.
- [x] Extend `nativerich/bridge_darwin.m` with native list indent/outdent, descendant movement, depth-aware numbering, Return/Backspace behavior, selection and undo. Add native harness acceptance tests first.
- [x] Add native table structural commands, context menu entries, keyboard row insertion/deletion, alignment and row/column movement. Add native harness tests first, including adjacent prose and undo/redo.
- [x] Audit/fix structured editing edge cases: empty items/cells, selection boundaries, IME, CRLF, source metadata and persistence.
- [x] Independently review spec compliance, then review code correctness; fix findings.
- [x] Run `go test ./...` and `go build ./...`; document verified improvements and unresolved Typora gaps accurately.

## Acceptance specifics

Lists load nested Markdown as editable items rather than source lines. Tab
indents an item beneath a preceding sibling, including its descendants;
Shift+Tab outdents. First siblings cannot create orphan indentation. Enter
continues at the current depth, empty nested items outdent, and root empty items
exit. Ordered numbering is independent at each depth. Each structural command
preserves selection and supports undo/redo without discarding formatting.

Tables expose insert/delete row/column, move row/column and per-column alignment
through the native editor. The table header and at least one body row/column
remain valid; deleting the final column has an explicit safe behavior. Cells
retain inline formatting and escaped content; operations use current text storage
rather than stale load metadata. Cursor lands in the intended cell after each
operation. Prose before/after the table remains unchanged.

These changes establish structural editing. Remaining advanced math, HTML/media,
footnote editing and full source-reveal parity stay tracked against the parent
design. Do not represent this milestone as full Typora parity.

## Verified result

- `go test ./... -count=1 -timeout=120s`: 340 passing cases in 12 packages.
- `go build ./...`: successful on macOS arm64 with cgo.
- Native AppKit screenshot inspected: `/tmp/corra-native-structure.png`.
- Independent spec and code review completed; empty-table extraction,
  independent nested numbering, and copied subtree normalization findings fixed.
- Additional delivered behavior: extended inline formatting input, titled/resolved
  reference links, code selection indentation, and image click/source editing.
- Manual Chinese IME candidate selection and large-document latency have not been
  interactively validated. Full Typora parity remains unfinished as listed in
  `docs/typora-parity.md`.
