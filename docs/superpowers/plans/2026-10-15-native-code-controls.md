# Native code block controls

Continue the approved AppKit/macOS Typora editing design. Code blocks already
hide fences and highlight their language, but the native surface needs language
selection and the existing line-number preference must reach it.

- [x] Actual NSTextView regression cases for language changes, body/fence
  preservation, adjacent independent blocks, selection, undo/redo and IME.
- [x] Native context language chooser, including arbitrary lexer names, without
  requiring users to reveal or manually change code fences.
- [x] Native line-number gutter, resetting at each fence and counting logical
  lines rather than wrapped visual fragments; no numbers in saved/copied text.
- [x] Forward CodeLineNumbers from the app preference and verify rendering,
  zoom, code editing/indentation and source preservation.
- [x] Independent review, full test/build and visual evidence.

Worker owns native code helper headers and acceptance tests. Root owns bridge
integration and Go/view preference forwarding. Follow the parent plan's TDD
and focused subagent workflow. Do not commit or push without a user request.

Pre-commit verification: 14 focused native code tests and 484 full-suite tests
passed, alongside 226 race tests for richtext/mdext/views, the full build and
diff checks. Native screenshot: `/tmp/corra-native-code-controls.png`.

Known remaining limits: replacements crossing code-block boundaries are rejected
to protect hidden fences. A final empty code block following prose can inherit
the preceding paragraph's extra-line layout until the caret enters it; its
header/gutter geometry still needs a dedicated fix and acceptance test. Actual
custom-language alert interaction and toggling line numbers during marked IME
remain unverified. These limits do not represent full Typora parity.
