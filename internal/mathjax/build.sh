#!/bin/sh
# Build-time Node/npm only; the application embeds the resulting self-contained JS.
set -eu
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
build_dir=$(mktemp -d "${TMPDIR:-/tmp}/corra-mathjax.XXXXXX")
trap 'rm -rf "$build_dir"' EXIT HUP INT TERM
npm install --prefix "$build_dir" --no-audit --no-fund --ignore-scripts --save-exact mathjax-full@3.2.2 esbuild@0.25.12
cp "$script_dir/entry.js" "$build_dir/entry.js"
"$build_dir/node_modules/.bin/esbuild" "$build_dir/entry.js" --bundle --platform=neutral --format=iife --target=es2018 --minify --legal-comments=inline --outfile="$script_dir/mathjax.bundle.js"
cp "$build_dir/node_modules/mathjax-full/LICENSE" "$script_dir/LICENSE"
