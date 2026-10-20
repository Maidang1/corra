package views

import (
	"testing"
)

// The export panel lists the built-in formats and runs their callbacks,
// closing itself before the native save dialog appears.
func TestExportPanelRunsBuiltinFormats(t *testing.T) {
	v, tt := notesFixture(t)
	var pdf, html bool
	v.OnExportPDF = func() { pdf = true }
	v.OnExportHTML = func() { html = true }
	v.PandocPath = func() string { return "/usr/local/bin/pandoc" }

	v.OpenExport()
	tt.Frame()
	if !tt.HasText("导出文档") || !tt.HasText("PDF 文档 (.pdf)") || !tt.HasText("HTML 网页 (.html)") || !tt.HasText("Word 文档 (.docx)") {
		t.Fatalf("format cards missing: %v", tt.Texts())
	}
	click(t, tt, "导出 PDF 文档 (.pdf)")
	tt.Frame()
	if !pdf || v.ExportOpen {
		t.Fatalf("PDF export: called=%v open=%v", pdf, v.ExportOpen)
	}
	v.OpenExport()
	tt.Frame()
	click(t, tt, "导出 HTML 网页 (.html)")
	tt.Frame()
	if !html || v.ExportOpen {
		t.Fatalf("HTML export: called=%v open=%v", html, v.ExportOpen)
	}
	v.Close()
}

// Pandoc cards forward the format's label, extension and converter argument.
func TestExportPanelRunsPandocFormat(t *testing.T) {
	v, tt := notesFixture(t)
	var label, ext, arg string
	v.OnExportPandoc = func(l, e, a string) { label, ext, arg = l, e, a }
	v.PandocPath = func() string { return "/usr/local/bin/pandoc" }

	v.OpenExport()
	tt.Frame()
	click(t, tt, "导出 Word 文档 (.docx)")
	tt.Frame()
	if label != "Word 文档 (.docx)" || ext != "docx" || arg != "docx" {
		t.Fatalf("pandoc export: %q %q %q", label, ext, arg)
	}
	if v.ExportOpen {
		t.Fatal("panel stayed open after pandoc export")
	}
	v.Close()
}

// Without pandoc the conversion cards stay disabled and the panel explains
// how to enable them; clicking a disabled card runs nothing.
func TestExportPanelWithoutPandoc(t *testing.T) {
	v, tt := notesFixture(t)
	called := false
	v.OnExportPandoc = func(string, string, string) { called = true }
	v.PandocPath = func() string { return "" }

	v.OpenExport()
	tt.Frame()
	if !tt.HasText("安装 pandoc") {
		t.Fatalf("pandoc hint missing: %v", tt.Texts())
	}
	click(t, tt, "导出 Word 文档 (.docx)")
	tt.Frame()
	if called {
		t.Fatal("disabled pandoc card ran its export")
	}
	if !v.ExportOpen {
		t.Fatal("disabled card closed the panel")
	}
	click(t, tt, "取消")
	tt.Frame()
	if v.ExportOpen {
		t.Fatal("cancel did not close the panel")
	}
	v.Close()
}

// The panel needs an open document, like the header's share button.
func TestExportPanelNeedsDocument(t *testing.T) {
	v, _ := notesFixture(t)
	v.App.Path = ""
	v.OpenExport()
	if v.ExportOpen {
		t.Fatal("panel opened without a document")
	}
	v.Close()
}
