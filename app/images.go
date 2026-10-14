package app

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"image"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/render"
)

// Invalidate connects the app to the window's redraw mechanism.
func (a *App) Invalidate(svc ui.Services) {
	a.services = svc
	a.invalidate = svc.Invalidate
}

func (a *App) InvalidateUI() {
	if a.invalidate != nil {
		a.invalidate()
	}
}

// BlockImage returns the rendered image for a heavy block (math, diagram,
// chart), rendering it asynchronously on first request. The second return
// is false while the render is pending or failed.
func (a *App) BlockImage(key string, fn func() (image.Image, error)) (image.Image, bool) {
	a.mu.Lock()
	e, ok := a.images[key]
	if !ok {
		e = &imageEntry{}
		a.images[key] = e
		a.mu.Unlock()
		go func() {
			img, err := fn()
			a.mu.Lock()
			e.img, e.err, e.ready = img, err, true
			a.mu.Unlock()
			a.InvalidateUI()
		}()
		return nil, false
	}
	a.mu.Unlock()
	return e.img, e.ready && e.err == nil
}

// Cache key for a block's rendered image.
func ImageKey(prefix string, b *render.Block, zoom float32, dark bool) string {
	h := sha1.Sum([]byte(b.Source))
	return fmt.Sprintf("%s-%s-%.2f-%v", prefix, hex.EncodeToString(h[:6]), zoom, dark)
}

// ---- Markdown / document image loading (local paths and remote URLs) ----

// ImageLoader fetches and decodes document images, with a small disk cache
// for remote ones.
type ImageLoader struct {
	cacheDir string
	client   *http.Client

	mu      sync.Mutex
	bitmaps map[string]*bitmapEntry
}

type bitmapEntry struct {
	ready bool
	bmp   *ui.Bitmap
	err   error
}

// Result returns the decoded bitmap and whether it is ready.
func (e *bitmapEntry) Result() (*ui.Bitmap, bool) { return e.bmp, e.ready && e.err == nil }

// Err returns the load error, if any.
func (e *bitmapEntry) Err() error { return e.err }

// NewImageLoader creates a loader; cacheDir may be "" (no disk cache).
func NewImageLoader(cacheDir string) *ImageLoader {
	return &ImageLoader{
		cacheDir: cacheDir,
		client:   &http.Client{Timeout: 15 * time.Second},
		bitmaps:  map[string]*bitmapEntry{},
	}
}

// Get returns the bitmap for a markdown image URL, loading it in the
// background on first use. Local paths resolve relative to baseDir.
func (l *ImageLoader) Get(url, baseDir string, invalidate func()) *bitmapEntry {
	l.mu.Lock()
	e, ok := l.bitmaps[url]
	if ok {
		l.mu.Unlock()
		return e
	}
	e = &bitmapEntry{}
	l.bitmaps[url] = e
	l.mu.Unlock()
	go func() {
		bmp, err := l.load(url, baseDir)
		l.mu.Lock()
		e.bmp, e.err, e.ready = bmp, err, true
		l.mu.Unlock()
		if invalidate != nil {
			invalidate()
		}
	}()
	return e
}

func (l *ImageLoader) load(url, baseDir string) (*ui.Bitmap, error) {
	var data []byte
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		data = l.loadRemote(url)
	} else {
		p := url
		if baseDir != "" && !filepath.IsAbs(p) {
			p = filepath.Join(baseDir, p)
		}
		var err error
		data, err = os.ReadFile(filepath.Clean(p))
		if err != nil {
			return nil, err
		}
	}
	if data == nil {
		return nil, fmt.Errorf("image: could not load %s", url)
	}
	return ui.DecodeBitmap(data)
}

func (l *ImageLoader) loadRemote(url string) []byte {
	// Disk cache first.
	if l.cacheDir != "" {
		sum := sha1.Sum([]byte(url))
		p := filepath.Join(l.cacheDir, hex.EncodeToString(sum[:])+".img")
		if data, err := os.ReadFile(p); err == nil {
			return data
		}
	}
	resp, err := l.client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var buf strings.Builder
	// read in chunks
	data := make([]byte, 0, 64*1024)
	tmp := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(tmp)
		data = append(data, tmp[:n]...)
		if err != nil {
			break
		}
	}
	_ = buf
	if l.cacheDir != "" && len(data) > 0 {
		sum := sha1.Sum([]byte(url))
		p := filepath.Join(l.cacheDir, hex.EncodeToString(sum[:])+".img")
		_ = os.WriteFile(p, data, 0o644)
	}
	return data
}
