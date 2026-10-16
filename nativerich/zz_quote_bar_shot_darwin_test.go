//go:build darwin && cgo

package nativerich

import (
	"os"
	"testing"
)

// Renders consecutive quote paragraphs (including a wrapped one, a nested
// quote and a callout) to a PNG so the connected quote bar can be checked
// visually. Run: CORRA_TYPO_SHOT=/tmp/quote.png go test ./nativerich -run TestQuoteBarShot
const quoteShotDoc = `# 引用竖线

> hello

> 11

> hh1k

> 中文混排 Mixed

> fewfe

普通段落，打断上面的引用连通。

> vea

> ca

> 这一行故意写得特别长，用来观察软换行之后竖线是否覆盖整段：引用块应该有一条左侧竖线与更安静的文本颜色，折行之后的第二行也应该被竖线覆盖住才对。

> > 嵌套引用的一层

> 回到外层

> [!NOTE]
> callout 的第一行
> callout 的第二行

文末段落。
`

func TestQuoteBarShot(t *testing.T) {
	path := os.Getenv("CORRA_TYPO_SHOT")
	if path == "" {
		t.Skip("set CORRA_TYPO_SHOT=path.png")
	}
	nativeTestMain(func() {
		window := nativeTestWindow()
		defer nativeTestCloseWindow(window)
		e := New(window, func() {}, func() {})
		if e == nil {
			t.Error("native editor unavailable")
			return
		}
		defer e.Close()
		e.SetFrame(0, 0, 900, 1100)
		e.SetVisible(true)
		e.Load(quoteShotDoc)
		nativeTestSelect(e, 0, 0)
		e.SetStyle(1, false, true)
		if !nativeTestScreenshot(e, path) {
			t.Errorf("screenshot failed: %s", path)
		}
	})
}
