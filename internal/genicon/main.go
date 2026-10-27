// Command genicon writes the app icon to resources/icon.png. The picture is
// an original drawing of the app's namesake: pointed ears, yellow eyes, a
// magenta cravat, and a crow on his shoulder. It is not a frame from the
// show. Run before packaging: go run ./internal/genicon
package main

import (
	_ "embed"
	"os"
)

//go:embed icon.png
var iconPNG []byte

func main() {
	if err := os.MkdirAll("resources", 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile("resources/icon.png", iconPNG, 0o644); err != nil {
		panic(err)
	}
}
