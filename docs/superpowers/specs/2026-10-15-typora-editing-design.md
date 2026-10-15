# Typora editing parity: design proposal

Status: architecture approved in chat on 2026-10-15: pure native, macOS only; no WebView. Proceed with the AppKit editor and plain Markdown storage.

## Goal and boundary

Upgrade the document editing area to support Typora's documented editing
capabilities. A rendered preview alone does not satisfy an editing requirement.
Every capability needs an input path, selection/caret behavior, undo/redo,
Markdown persistence, and a repeatable acceptance scenario.

The target includes Markdown live editing, tables, images, math, diagrams,
references, HTML/media blocks, editing commands, writing modes, and editor
preferences. Library management, account services, and release packaging are
separate concerns. Export correctness remains a regression requirement for
documents edited through the new surface.

## Verified starting point before implementation

- MyGo is pinned to v0.5.1; the native shell and AppKit editor remain in use.
- `main.go` currently attaches an AppKit editor through `nativerich.New`.
  Native rich editing is selected on macOS with cgo; other builds use the
  source/preview path.
- `richtext/codec.go` models a flat paragraph list. `simpleList` rejects nested
  and multi-block items. Reference links and links with titles fall back to source.
- `nativerich/bridge_darwin.m` makes Tab inert in list items and converts a list
  item to body text on Shift+Tab. This is not nested-list indentation.
- Tables use NSTextTable and already support cell editing/navigation. Structure
  editing, row/column movement and alignment require a separate acceptance audit.
- Math/images/diagrams use attachments with source reveal. Current math uses the
  existing Go renderer, whose compatibility differs from Typora's MathJax stack.
- The existing `docs/typora-parity.md` mixes preview, export and editing support;
  it must be revised against executable scenarios before making parity claims.

## Selected architecture

Keep MyGo's native shell and the AppKit NSTextView editor. No WebView or web
frontend will be introduced. macOS with cgo is the only target for this upgrade.

Extend the existing source-preserving paragraph model with list depth and
container metadata. Structural commands operate as a single undo transaction,
with stable UTF-16 selections and typing attributes. Keep complex unsupported
constructs editable as source until their native interaction is implemented.

Native table editing uses NSTextTable plus contextual commands to insert,
delete, move and align rows/columns. The native editor owns selection/history;
Go owns Markdown parsing, serialization and existing save/conflict handling.

Full Typora parity is the target, not an unverified completion claim. Record
remaining gaps explicitly, including advanced math, HTML/media and drag handles.

## Capability and acceptance inventory

| Area | Required behavior and representative scenario |
| --- | --- |
| Live Markdown | Type headings, inline styles, links and delimiters; reveal syntax at the active range; edit it and leave the range without caret jumps or duplicate markers. |
| Text input | Chinese IME composition, emoji/UTF-16, dead keys, CJK punctuation, multiline selection, pair insertion/deletion and plain-text paste preserve intended text. |
| Lists and quotes | Mixed nested ordered/unordered/task lists, multi-paragraph items and nested quotes; Tab/Shift+Tab move the selected subtree; Enter continues/exits; Backspace demotes/merges; numbering stays correct. |
| Tables | Create and edit tables; navigate both directions; add/delete/reorder rows and columns; set alignment; paste rectangular data; preserve escaped pipes and inline formatting. |
| Code | Create fences by typing, choose language, highlight, indent/outdent selections, show line numbers, and enter/exit blocks by keyboard. |
| Images | Insert/paste/drop at the selection; edit URL, alt, title and dimensions; resolve relative paths/base paths; preserve existing hosting and local attachments. |
| Mathematics | Inline/display edit-preview transitions, MathJax-compatible expressions, macros, chemistry/physics, labels, references and numbering; errors preserve editable source. |
| Diagrams | Edit supported diagram source beside/in place of rendered output; show useful errors; support the documented Typora diagram families and preserve Corra-specific charts. |
| References and metadata | Edit footnotes/reference links, inspect their definitions, update TOC, edit YAML metadata and GitHub-style alerts without flattening source. |
| HTML and media | Edit supported HTML, video and embedded content; document renderer behavior and isolate document content from privileged application bindings. |
| Commands | Find/replace, next/previous match, select word/line/style/cell/row, paragraph formatting, format clearing and configurable/conflict-free shortcut routing. |
| Writing modes | Focus/typewriter modes, spell checking, smart punctuation, emoji completion, zoom, reading width, line spacing and theme settings. |
| History and storage | Each semantic operation has predictable undo/redo; switching notes/modes and autosaving must preserve history, selection and pending composition. |

The matrix is the initial inventory. Expand it from the official documentation
before calling the target complete; unverified items stay explicitly unverified.

## Data flow and integration contract

Markdown on disk remains the authoritative interchange format. The editing
surface holds the active transaction/history state. A document ID and monotonically
increasing revision accompany load, change and save acknowledgements so late
events cannot overwrite another note. App-driven reloads are distinguishable from
user edits and cannot create change echo loops.

Composition blocks destructive mode/file transitions until committed or explicitly
cancelled. Save/open/close/export operate on a current snapshot and surface errors
without discarding edits. Reuse the existing autosave and external-conflict flow.

Use native NSTextView, NSTextTable and native context menus. No HTML/JavaScript
frontend or WebView is introduced. Existing Go math/diagram renderers produce
native image attachments. Rendering errors preserve editable source.

Source preservation is a release gate: preserve untouched blocks and
unsupported syntax exactly; normalize edited constructs only through a documented
serializer policy. Untouched reference links retain exact spelling; editing a
paragraph containing a resolved reference may emit inline links with the same
destination and title, preserving its definition block. Include CRLF, reference definitions, front matter, link titles,
HTML and custom Corra blocks in the corpus. Do not silently drop extensions.

## Delivery sequence

1. Audit: turn the inventory into input/output fixtures and identify verified
   existing behavior, partial behavior and missing features.
2. Native model: extend list hierarchy and container identity; preserve source,
   selection, IME, history and existing Go save synchronization.
3. Editing foundation: extend the existing surface, persistence, command
   routing, composition, note switching and recovery. Keep a recoverable source
   editing path throughout migration.
4. Structured editing: nested lists/quotes, tables, code, links and images.
5. Extended content: math, diagrams, footnotes, TOC, metadata, HTML/media.
6. Writing preferences and full capability audit; update user documentation using
   verified results rather than blanket parity claims.

Each milestone must produce usable behavior with its own acceptance evidence.
Implementing one milestone does not satisfy the entire inventory.

## Verification

- Unit tests for transformations, serializer stability and bridge revision rules.
- Native AppKit harness tests for actual key sequences, selection, clipboard,
  context menus and undo/redo.
- Composition event tests plus manual Chinese IME candidate-selection checks.
- Open/edit/autosave/reopen and external-change conflict tests using isolated notes.
- Offline startup and local image/math/diagram checks; failed rendering preserves
  editable input. Inspect screenshots from the real AppKit surface.
- Existing `go build ./...` and `go test ./...` remain required regression checks.
- Only macOS rich editing is in scope. Manual Chinese IME candidate selection and
  large-document typing latency require separate interactive validation.

## Primary references

- https://support.typora.io/Markdown-Reference/
- https://support.typora.io/Shortcut-Keys/
- https://support.typora.io/Table-Editing/
- https://support.typora.io/Math/
- https://support.typora.io/Auto-Pair/
- Local implementation: `richtext/`, `nativerich/`, `views/rich_editor.go`.
