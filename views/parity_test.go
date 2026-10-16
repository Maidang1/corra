package views

import (
	"testing"
)

func TestCountWords(t *testing.T) {
	wc := countWords("你好 world\n第二行 test")
	if wc.Words != 5 { // 你 好 world 第 二 行 test -> 2 + 1 + 3 + 1 = 7? count: CJK 5 + latin 2 = 7
		_ = wc
	}
	if wc.Words != 7 {
		t.Fatalf("words: %+v", wc)
	}
	if wc.Lines != 2 {
		t.Fatalf("lines: %+v", wc)
	}
	if wc.Minutes != 1 {
		t.Fatalf("minutes: %+v", wc)
	}
	if countWords("").Words != 0 {
		t.Fatal("empty")
	}
}
