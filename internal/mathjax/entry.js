/* MathJax 3.2.2, Apache-2.0. See LICENSE and THIRD_PARTY_NOTICES.md. */
const {mathjax} = require('mathjax-full/js/mathjax.js');
const {TeX} = require('mathjax-full/js/input/tex.js');
const {SVG} = require('mathjax-full/js/output/svg.js');
const {liteAdaptor} = require('mathjax-full/js/adaptors/liteAdaptor.js');
const {RegisterHTMLHandler} = require('mathjax-full/js/handlers/html.js');
const {AllPackages} = require('mathjax-full/js/input/tex/AllPackages.js');
const adaptor = liteAdaptor();
RegisterHTMLHandler(adaptor);
const packages = AllPackages.filter(name => name !== 'noerrors' && name !== 'noundefined').concat(['physics']);

function textOf(node) {
  if (node.kind === 'text') return node.getText();
  return (node.childNodes || []).map(textOf).join('');
}
function numbersOf(node, result) {
  if (node.kind === 'mlabeledtr') {
    const tag = textOf(node.childNodes[0]);
    result.push(tag.replace(/^\((.*)\)$/, '$1'));
  }
  for (const child of node.childNodes || []) numbersOf(child, result);
}

// MathJax normally uses responsive nested SVG viewports for equation tags.
// Resolve those viewports now so a native SVG rasterizer needs no CSS layout.
function fixedSVG(root, svg) {
  if (adaptor.hasAttribute(root, 'viewBox')) return root;
  const ex = 8;
  const unitsPerEx = svg.font.params.x_height * 1000;
  const length = (value, reference) => {
    if (!value) return reference;
    if (value.endsWith('ex')) return parseFloat(value) * ex;
    if (value.endsWith('%')) return parseFloat(value) / 100 * reference;
    return parseFloat(value);
  };
  const width = length(adaptor.getStyle(root, 'min-width'), 1);
  const height = length(adaptor.getAttribute(root, 'height'), 1);
  if (!(width > 0 && height > 0 && isFinite(width) && isFinite(height))) throw Error('Invalid responsive math dimensions');
  const flatten = (parent, viewportWidth, viewportHeight) => {
    for (const child of [...adaptor.childNodes(parent)]) {
      if (adaptor.kind(child) !== 'svg') { if (adaptor.kind(child) !== '#text') flatten(child, viewportWidth, viewportHeight); continue; }
      const box = (adaptor.getAttribute(child, 'viewBox') || '').split(/\s+/).map(Number);
      if (box.length !== 4 || box[2] <= 0 || box[3] <= 0) throw Error('Invalid nested math SVG viewBox');
      const w = length(adaptor.getAttribute(child, 'width'), viewportWidth);
      const h = length(adaptor.getAttribute(child, 'height'), viewportHeight);
      const alignment = adaptor.getAttribute(child, 'preserveAspectRatio') || 'xMidYMid';
      const scale = Math.min(w / box[2], h / box[3]);
      const dx = alignment.includes('xMax') ? w - box[2] * scale : alignment.includes('xMin') ? 0 : (w - box[2] * scale) / 2;
      const dy = alignment.includes('YMax') ? h - box[3] * scale : alignment.includes('YMin') ? 0 : (h - box[3] * scale) / 2;
      flatten(child, w, h);
      const group = adaptor.node('g', {transform: `matrix(${scale} 0 0 ${scale} ${dx - box[0] * scale} ${dy - box[1] * scale})`}, adaptor.childNodes(child), 'http://www.w3.org/2000/svg');
      adaptor.replace(group, child);
    }
  };
  flatten(root, width, height);
  const scale = unitsPerEx / ex;
  const content = adaptor.node('g', {transform: `scale(${scale})`}, adaptor.childNodes(root), 'http://www.w3.org/2000/svg');
  adaptor.append(root, content);
  adaptor.setAttribute(root, 'viewBox', `0 0 ${width * scale} ${height * scale}`);
  adaptor.setAttribute(root, 'width', `${width / ex}ex`);
  return root;
}

// No filesystem, loader, browser, timers, or network API is exposed to TeX.
globalThis.corraMathJax = function (request) {
  const tex = new TeX({
    packages,
    macros: request.options.macros || {},
    tags: request.options.numbering || 'none',
    maxMacros: 1000,
    maxBuffer: 64 * 1024,
    formatError: (jax, error) => { throw error; }
  });
  const svg = new SVG({fontCache: 'none', internalSpeechTitles: false});
  const document = mathjax.document('', {InputJax: tex, OutputJax: svg});
  const items = request.expressions.map(expression => {
    const item = new document.options.MathItem(expression.tex, tex, expression.display);
    item.start.node = adaptor.body(document.document);
    item.setMetrics(16, 8, 640, 1000000, 1);
    document.math.push(item);
    return item;
  });
  // MathDocument.compile includes MathJax's reference-update pass, so forward
  // references resolve against the completed document without renumbering.
  document.compile();
  let total = 0;
  return items.map(item => {
    if (item.inputData.error) return {error: String(item.inputData.error.message || item.inputData.error)};
    try {
      item.typeset(document);
      const root = adaptor.firstChild(item.typesetRoot);
      const result = adaptor.outerHTML(fixedSVG(root, svg));
      if (!result.startsWith('<svg')) throw Error('MathJax did not produce SVG');
      total += result.length;
      if (result.length > 8 * 1024 * 1024 || total > 32 * 1024 * 1024) throw Error('MathJax SVG output limit exceeded');
      const numbers = [];
      numbersOf(item.root, numbers);
      return {svg: result, number: numbers.join(', ')};
    } catch (error) {
      return {error: String(error.message || error)};
    }
  });
};
