#!/bin/sh
# Corra verifier v2: v1 plus a real packaging check (`go tool mygo build`),
# proving the same command the CI release workflow runs works locally.
cd "$(dirname "$0")/../.." || exit 1
sh verifier/v1/check.sh || exit 1
echo "== 8. icon generation (go run ./internal/genicon)"
go run ./internal/genicon > /tmp/icon.log 2>&1 || { echo "FAIL: genicon: $(cat /tmp/icon.log)"; exit 1; }
[ -f resources/icon.png ] || { echo "FAIL: icon.png missing"; exit 1; }

echo "== 9. mygo packaging (go tool mygo build)"
rm -rf /tmp/corra-verify-build
if go tool mygo build -o /tmp/corra-verify-build > /tmp/pkg.log 2>&1; then
  # Linux/Windows package a plain binary named after the module; macOS
  # packages a signed .app bundle whose executable is named after the app.
  bin=$(find /tmp/corra-verify-build -type f \( -name corra -o -name Corra \) | head -1)
  if [ -n "$bin" ]; then
    sz=$(du -h "$bin" | cut -f1)
    echo "packaged binary: $bin ($sz)"
  else
    echo "FAIL: no binary in build output"; exit 1
  fi
else
  echo "FAIL: mygo build: $(tail -10 /tmp/pkg.log)"; exit 1
fi
echo "VERIFIER v2: PASS"
