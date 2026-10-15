# Native inline source editing

Continue the approved native Typora editing design using AppKit only. This is
one part of the full capability inventory, not a reduction of its scope.

Inline syntax must be editable where it is rendered. A plain click or keyboard
navigation into a styled cluster reveals its Markdown. Moving away, leaving the
editor or Cmd+Return parses the live source back into native attributed text.
Selections that cross a cluster remain selections; IME composition must not
trigger replacement. Reference links may normalize through the existing resolved
link policy when edited; unchanged paragraphs retain their original source.

- [x] Add a serializer-backed inline fragment and UTF-16 caret mapping helper.
- [x] Reveal contiguous overlapping inline formats while keeping paragraph and
  table identities; avoid source/code blocks and existing source spans.
- [x] Reuse source folding for non-math syntax. Rendering errors and unsupported
  inline models remain source; unmatched Markdown delimiters remain literal,
  editable text and can regain formatting when completed.
- [x] Route mouse and keyboard navigation through reveal without disturbing
  drag/word selections or command-driven formatting.
- [x] Exercise actual NSTextView edits, source delimiters, links, nested styles,
  Unicode, IME, table cells and undo/redo.
- [x] Independent review, full tests/build and accurate inventory update.

Go helper ownership: `richtext/inline_reveal.go` and its tests. Native integration
ownership: `nativerich/` bridge and new inline source acceptance tests. Work uses
the parent plan's test-first/subagent workflow; no automatic commit or push.

Original runs are retained with an expanded fragment until its live text changes.
This keeps merely revealing alternate delimiters/reference links from rewriting
the stored paragraph. Actual edits use live Markdown. Selected replacements that
cross a source boundary first fold and map the selection, including typing,
paste, backward/forward deletion and formatting commands.

The helper uses actual AST source segments, rather than matching visible text
inside Markdown containing URLs/titles. It checks lossless text/style projection
and supports source-to-rendered positions for the user's current spelling. It
returns an error if that projection cannot be proved; the native view retains
its current presentation. This milestone does not claim every possible syntax
combination or the full Typora inventory is complete.

Native visual evidence: `/tmp/corra-native-inline.png`, generated with
`CORRA_INLINE_SCREENSHOT=/tmp/corra-native-inline.png go test ./nativerich -run TestNativeInlineSource -count=1`.
The screenshot shows a live expanded CJK/emoji fragment among rendered styles
and an editable native table. Manual IME candidate selection and large-document
interactive latency remain unverified.

Pre-commit verification: `go test ./... -count=1 -timeout=180s` (484 tests),
`go test -race ./richtext ./internal/mdext ./views -count=1` (226 tests),
`go build ./...`, and `git diff --check` passed.
