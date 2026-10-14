# Verifier index

Append-only record of acceptance verifiers. One entry per version: when it
was created, what it measures, how it differs from the prior version. Every
run appends a timestamped record under `runs/` with command, exit code and
produced values.

## v1 (2026-10-14)

Created: first acceptance gate for the goal "implement per plan v3.5,
design-aware, push to GitHub, CI auto-packaging". Measures:

1. gofmt clean (vendored `third_party/` excluded)
2. `go vet ./...` clean
3. `go build ./...` succeeds
4. `go test ./...` passes (render pipeline, editor core, headless view)
5. no frontend artifacts in the tree (no .html/.ts/.js) — the "native UI,
   no frontend code" requirement
6. required deliverable files exist (mygo.json, icon, README, workflows,
   feature sources)
7. the main window is created with `ui.View` (native UI), not a webview URL

Run: `sh verifier/v1/check.sh`

## v2 (2026-10-14)

Extends v1 with check 8 (icon generation via `go run ./internal/genicon`) and check 9: `go tool mygo build` produces a packaged app
(binary + desktop entry + archive + deb) — the same command the release
workflow runs per platform. v1 checks unchanged.

2026-10-14 amendment: check 9 accepts a macOS `.app` bundle executable
(`Corra.app/Contents/MacOS/Corra`) in addition to the plain `corra`
binary Linux/Windows package — the gate now passes on macOS hosts too.

Run: `sh verifier/v2/check.sh`
