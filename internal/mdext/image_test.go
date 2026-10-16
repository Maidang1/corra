package mdext

import "testing"

func TestImageSizeValid(t *testing.T) {
	for _, tc := range []struct {
		spec string
		want bool
	}{
		{"", false},
		{"300x200", true},
		{"300x", true},
		{"x200", true},
		{"300", true},
		{"50%", true},
		{"x", false},
		{"1x2x3", false},
		{"-300x200", false},
		{"300x200px", false},
		{"tall", false},
		{"300 x 200", false},
		{"123456x", false},
	} {
		if got := ImageSizeValid(tc.spec); got != tc.want {
			t.Errorf("ImageSizeValid(%q) = %v, want %v", tc.spec, got, tc.want)
		}
	}
}

func TestSizedImage(t *testing.T) {
	for _, tc := range []struct {
		text, alt, url, spec string
		want                 bool
	}{
		{"![alt](a.png)", "alt", "a.png", "", true},
		{"![alt](a.png =300x200)", "alt", "a.png", "300x200", true},
		{"![alt](a.png =50%)", "alt", "a.png", "50%", true},
		{"![alt](a.png \"title\" =300x)", "alt", "a.png", "300x", true},
		{"![alt](<a b.png> =10x10)", "alt", "a b.png", "10x10", true},
		{"![](a.png =10x10)", "", "a.png", "10x10", true},
		{"![a](a.png =300x200) tail", "", "", "", false},
		{"![a](a.png =300x200", "", "", "", false},
		{"![a]( =300x200)", "", "", "", false},
		{"![a](a.png =)", "", "", "", false},
		{"![a](a.png =300x200px)", "", "", "", false},
		{"![a[b]](a.png =10x10)", "", "", "", false},
		{"before ![a](a.png =10x10)", "", "", "", false},
		{"![a](a.png) text", "", "", "", false},
		{"[a](a.png)", "", "", "", false},
		{"![a](a.png \"unclosed =10x10)", "", "", "", false},
	} {
		alt, url, spec, ok := SizedImage(tc.text)
		if ok != tc.want || alt != tc.alt || url != tc.url || spec != tc.spec {
			t.Errorf("SizedImage(%q) = %q, %q, %q, %v; want %q, %q, %q, %v",
				tc.text, alt, url, spec, ok, tc.alt, tc.url, tc.spec, tc.want)
		}
	}
}
