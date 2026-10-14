#!/bin/sh
# Corra verifier v1: acceptance checks for the goal
# "implement per plan, design-aware, pushed to GitHub, CI auto-packaging".
# Run from the repo root. Environment: PATH includes Go, GOPROXY set.
cd "$(dirname "$0")/../.." || exit 1
FAIL=0
note() { echo "== $*"; }
fail() { echo "FAIL: $*"; FAIL=1; }

note "1. gofmt (excluding vendored deps)"
out=$(gofmt -l . 2>/dev/null | grep -v third_party)
[ -z "$out" ] || fail "unformatted: $out"

note "2. go vet"
go vet ./... > /tmp/vet.log 2>&1 || fail "vet: $(cat /tmp/vet.log)"

note "3. go build"
go build ./... > /tmp/build.log 2>&1 || fail "build: $(cat /tmp/build.log)"

note "4. go test"
go test ./... > /tmp/test.log 2>&1 || fail "test: $(tail -20 /tmp/test.log)"

note "5. no frontend artifacts (no HTML/JS frontend code)"
if find . -path ./third_party -prune -o -name "*.html" -print -o -name "*.ts" -print -o -name "*.js" -print | grep -v node_modules | grep . ; then
  fail "frontend files found"
fi

note "6. required files"
for f in go.mod main.go mygo.json resources/icon.png README.md \
         .github/workflows/release.yml .github/workflows/ci.yml \
         setup/workflows/release.yml setup/workflows/ci.yml setup/install-ci.sh \
         render/export.go render/diagram.go render/mathimg.go \
         app/hosting.go views/view.go menu.go; do
  [ -f "$f" ] || fail "missing $f"
done

note "7. native UI only (no webview window in main)"
grep -q "Content: *ui.View" main.go || fail "window is not native UI"

if [ $FAIL -eq 0 ]; then echo "VERIFIER v1: PASS"; else echo "VERIFIER v1: FAIL"; fi
exit $FAIL
