# Bundled MathJax

`mathjax.bundle.js` contains MathJax 3.2.2, Copyright 2009-2022 The MathJax
Consortium, distributed under Apache License 2.0 (`LICENSE`). The source is
https://github.com/mathjax/MathJax-src/tree/3.2.2 and the npm package is
`mathjax-full@3.2.2`. The SVG TeX font path data is included in that distribution.
The bundled mhchem parser is by Martin Hensel, distributed with MathJax under
Apache License 2.0; its original copyright and license notice remains in the
bundle. No DOM implementation or network loader from npm is shipped separately.

`entry.js` selects MathJax's LiteDOM adaptor, TeX input, static SVG TeX fonts,
and the shipped TeX extensions including mhchem and physics. Error-suppressing
`noerrors` and `noundefined` packages are disabled. Dynamic loading (`require`
and `autoload`) is not enabled; unsupported packages return an editable-source
error. Version 3.2.2 is pinned to keep all font paths synchronous and offline.

Rebuild with `./build.sh`. Node and npm are build tools only. The script pins
MathJax 3.2.2 and esbuild 0.25.12 and produces an IIFE with no external imports.
