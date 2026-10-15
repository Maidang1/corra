// Attachment saving: pasted and dropped images/files land in an assets
// folder next to the note (or upload to the configured image host).
package app

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AssetsDirName is the per-folder attachment directory.
const AssetsDirName = "assets"

var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".webp": true, ".svg": true, ".bmp": true, ".heic": true,
}

// IsImageFile reports whether the path looks like an image.
func IsImageFile(path string) bool {
	return imageExts[strings.ToLower(filepath.Ext(path))]
}

// SaveAttachment stores data as an attachment of the note and returns the
// Markdown reference. When image hosting is configured and the attachment
// is an image, it uploads instead of writing a local file.
func (a *App) SaveAttachment(name string, data []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		ext = ".png"
		name += ext
	}
	isImage := imageExts[ext]

	a.mu.Lock()
	cfg, notePath := a.Hosting, a.Path
	a.mu.Unlock()

	if isImage && cfg.Enabled && cfg.Token != "" {
		res, err := UploadImageToGitHub(cfg, name, data)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("![%s](%s)", attachmentAlt(name), res.URL), nil
	}

	if notePath == "" {
		return "", fmt.Errorf("先保存备忘录，再插入附件")
	}
	dir := filepath.Join(filepath.Dir(notePath), AssetsDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	stem := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	stem = strings.Map(func(r rune) rune {
		if strings.ContainsRune("/\\:*?\"<>| ", r) {
			return '-'
		}
		return r
	}, stem)
	target := filepath.Join(dir, fmt.Sprintf("%s-%s%s", time.Now().Format("20060102-150405"), stem, ext))
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return "", err
	}
	rel := AssetsDirName + "/" + filepath.Base(target)
	escaped := strings.ReplaceAll(url.PathEscape(rel), "%2F", "/")
	if isImage {
		return fmt.Sprintf("![%s](%s)", attachmentAlt(name), escaped), nil
	}
	return fmt.Sprintf("[%s](%s)", filepath.Base(name), escaped), nil
}

// ImportDroppedFile copies a dropped file into the note's assets and returns
// its Markdown reference (images as images, everything else as links).
func (a *App) ImportDroppedFile(path string) (string, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	return a.SaveAttachment(filepath.Base(path), data)
}

func attachmentAlt(name string) string {
	alt := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	return strings.ReplaceAll(alt, "]", "\\]")
}
