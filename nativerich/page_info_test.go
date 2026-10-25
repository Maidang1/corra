package nativerich

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPageInfo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "周报.md")
	if err := os.WriteFile(path, []byte("---\ntags: x\n---\n\n# 第 42 周\n\n本周完成了 **颜色** 和提及。\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := PageInfo(path); got.Title != "第 42 周" || got.Excerpt != "本周完成了 颜色 和提及。" || got.Missing {
		t.Errorf("PageInfo = %+v", got)
	}
	plain := filepath.Join(dir, "无标题.md")
	if err := os.WriteFile(plain, []byte("只有正文"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := PageInfo(plain); got.Title != "无标题" || got.Excerpt != "只有正文" {
		t.Errorf("untitled PageInfo = %+v", got)
	}
	if got := PageInfo(filepath.Join(dir, "gone.md")); !got.Missing {
		t.Errorf("missing note = %+v", got)
	}
}
