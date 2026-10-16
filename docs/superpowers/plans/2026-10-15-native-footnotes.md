# Native footnote editing

Continue the approved macOS/AppKit Typora editor architecture. No WebView.
User authorization is the instruction to implement the entire parity inventory.

Footnote references become native inline attachments, numbered in reference
order, with native hover help containing the current definition. Definitions
retain their source position and identifier while their content uses the same
editable paragraphs, tables and inline styles as the rest of the document.
Click/arrow source reveal edits reference syntax; context and keyboard commands
navigate between a reference and its definition. Unresolved references remain
visible and editable. Insertion is one native undo transaction rather than an
autosave/reload operation.

The Go model retains exact reference tokens and definition container metadata.
Source-position-preserving parsers avoid moving definitions to the end of the
document. Unchanged Markdown, duplicate definitions and CRLF round-trip exactly;
edited containers regenerate valid indentation without swallowing neighbors.
Presentation numbers and tooltips are derived from the live native document and
never enter stored Markdown.

- [x] Go parser/serializer and exact inline caret maps, with failing tests first.
- [ ] Actual NSTextView reference rendering, hover help, definition edits and
  source expansion/folding, including Unicode, tables, CRLF and undo/redo.
  Block/code/newline container retention is green; the remaining gap is the
  final-code-exit and no-blank-neighbor formula fold fixed on 2026-10-15
  (see "Native footnote regressions" below).
- [ ] Native reference/definition navigation and atomic insertion.
- [ ] Preserve container identity through newline, formatting, IME and clipboard
  operations; test structural boundaries and deletion of the first paragraph.
- [ ] Independent specification and correctness reviews, full tests/race/build,
  native visual inspection and an accurate parity inventory update.

## Native footnote regressions (fixed 2026-10-15)

Two focused failures were reproduced and fixed:

1. `TestNativeFootnoteBlockNewlineKeepsContainer` (code-exit case). Enter on the
   final empty code line of a footnote definition and typing produced the right
   live document, but `Snapshot` returned the pre-edit source and dropped the
   new body paragraph. Root cause was not the serializer: `code_editing_darwin.h`
   boxed `@(i>0 || [first[@"footnoteContinuation"] boolValue])`, and C's `||`
   yields `int`, so Objective-C emitted `"footnoteContinuation":0`. Go's `bool`
   field rejected the number, `Editor.Snapshot` failed to decode and fell back to
   the stale `e.source`. Fixed with an explicit `@((BOOL)(...))` cast.
2. `TestNativeFootnoteFormulaWithoutBlankNeighbors`. Revealing a footnote formula
   and folding it with Cmd+Return inserted blank lines around the definition.
   `renderSource` re-parses the revealed source and rebuilt the paragraph without
   its `id`/`nextId`, so `serializeDocument`'s adjacency check treated the
   neighbors as changed and widened both separators. `renderSource` now restores
   the revealed paragraph's identity when the fold is one-to-one.

Verification: `go test ./nativerich -run '^TestNativeFootnote'` 20 passed;
`go test ./...` 517 passed; `go test -race ./richtext ./internal/mdext ./render
./views` 285 passed; `go build ./...` and `git diff --check` clean.

Ownership: model worker owns richtext footnote helpers and codec/reveal hooks;
root owns native bridge, view insertion, acceptance tests and documentation.
Use focused subagents for implementation/review under the parent plan. Do not
commit or push without a new user request.
