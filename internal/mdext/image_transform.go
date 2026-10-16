package mdext

import (
	"strconv"
	"strings"

	"github.com/yuin/goldmark/ast"
	gast "github.com/yuin/goldmark/extension/ast"
)

// TransformSizedImages gives every figure its size back as HTML attributes.
// goldmark only accepts a destination holding a space once it is wrapped in
// angle brackets, so "![a](p.png =300x200)" arrives as plain text and is turned
// back into the image it is. An image goldmark did parse keeps its suffix in the
// destination, which would otherwise be exported as part of the URL.
func TransformSizedImages(root ast.Node, src []byte) {
	var blocks []ast.Node
	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n.(type) {
		case *ast.Image:
			sizeImageNode(n)
			return ast.WalkSkipChildren, nil
		case *ast.Paragraph, *ast.TextBlock, *gast.TableCell:
			blocks = append(blocks, n)
		}
		return ast.WalkContinue, nil
	})
	// The walk is done, so replacing children here cannot disturb it.
	for _, block := range blocks {
		alt, url, title, spec, ok := imageParts(plainTextSource(block, src))
		if !ok {
			continue
		}
		link := ast.NewLink()
		link.Destination = []byte(url)
		if title != "" {
			link.Title = []byte(title)
		}
		img := ast.NewImage(link)
		img.AppendChild(img, ast.NewString([]byte(alt)))
		setImageSizeAttributes(img, spec)
		for child := block.FirstChild(); child != nil; {
			next := child.NextSibling()
			block.RemoveChild(block, child)
			child = next
		}
		block.AppendChild(block, img)
	}
}

// sizeImageNode moves a size suffix out of a destination goldmark accepted and
// into the attributes the HTML renderer already knows how to write.
func sizeImageNode(n ast.Node) {
	img, ok := n.(*ast.Image)
	if !ok {
		return
	}
	url, title, spec, ok := SplitImageDestination(string(img.Destination))
	if !ok || url == "" {
		return
	}
	img.Destination = []byte(url)
	img.Title = nil
	if title != "" {
		img.Title = []byte(strings.Trim(title, `"'`))
	}
	setImageSizeAttributes(img, spec)
}

// setImageSizeAttributes writes a size suffix as width and height, or as a
// percentage of the column.
func setImageSizeAttributes(img *ast.Image, spec string) {
	if spec == "" {
		return
	}
	if percent, ok := strings.CutSuffix(spec, "%"); ok {
		if v, err := strconv.Atoi(percent); err == nil && v > 0 && v <= 400 {
			img.SetAttribute([]byte("style"), []byte("width:"+percent+"%"))
		}
		return
	}
	width, height, _ := strings.Cut(spec, "x")
	if v, err := strconv.Atoi(width); err == nil && v > 0 {
		img.SetAttribute([]byte("width"), []byte(width))
	}
	if v, err := strconv.Atoi(height); err == nil && v > 0 {
		img.SetAttribute([]byte("height"), []byte(height))
	}
}

// plainTextSource joins a block's inline children, or returns "" when any of
// them is more than plain text: only then is the block's own source the whole
// story, and only then may it be replaced.
func plainTextSource(block ast.Node, src []byte) string {
	var text strings.Builder
	for child := block.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Text:
			text.Write(n.Segment.Value(src))
		case *ast.String:
			text.Write(n.Value)
		default:
			return ""
		}
	}
	return text.String()
}
