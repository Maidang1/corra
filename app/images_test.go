package app

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestImageLoaderConcurrentResult(t *testing.T) {
	for _, failed := range []bool{false, true} {
		name := "success"
		if failed {
			name = "failure"
		}
		t.Run(name, func(t *testing.T) {
			var data bytes.Buffer
			if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
				t.Fatal(err)
			}
			release := make(chan struct{})
			done := make(chan struct{})
			loader := NewImageLoader("")
			loader.client.Transport = imageRoundTripper(func(*http.Request) (*http.Response, error) {
				<-release
				if failed {
					return nil, errors.New("image unavailable")
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(data.Bytes()))}, nil
			})
			entry := loader.Get("https://example.test/image.png", "", func() { close(done) })
			pollImagePublication(t, release, done, func() {
				entry.Result()
				entry.Err()
			})
			bitmap, ready := entry.Result()
			if failed {
				if bitmap != nil || ready || entry.Err() == nil {
					t.Fatalf("failed image result = %v, %t, %v", bitmap, ready, entry.Err())
				}
			} else if bitmap == nil || !ready || entry.Err() != nil {
				t.Fatalf("loaded image result = %v, %t, %v", bitmap, ready, entry.Err())
			}
		})
	}
}

func TestAppBlockImageConcurrentResult(t *testing.T) {
	a := New()
	release := make(chan struct{})
	done := make(chan struct{})
	a.invalidate = func() { close(done) }
	want := image.NewRGBA(image.Rect(0, 0, 2, 2))
	render := func() (image.Image, error) {
		<-release
		return want, nil
	}
	if img, ready := a.BlockImage("key", render); img != nil || ready {
		t.Fatal("unrendered image is ready")
	}
	pollImagePublication(t, release, done, func() { a.BlockImage("key", render) })
	if img, ready := a.BlockImage("key", render); img != want || !ready {
		t.Fatalf("rendered image = %v, %t", img, ready)
	}
}

func TestAppInvalidateConcurrent(t *testing.T) {
	a := New()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range 200 {
			a.Invalidate(ui.Services{})
		}
	}()
	go func() {
		defer wg.Done()
		for range 200 {
			a.InvalidateUI()
		}
	}()
	wg.Wait()
}

func TestImageLoaderLocalBaseIsolation(t *testing.T) {
	loader := NewImageLoader("")
	var entries []*bitmapEntry
	for range 2 {
		root := t.TempDir()
		var data bytes.Buffer
		if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "same.png"), data.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		entry := loader.Get("same.png", root, func() { close(done) })
		if len(entries) > 0 && entry == entries[0] {
			t.Fatal("same URL in different note folders reused first bitmap")
		}
		entries = append(entries, entry)
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("image did not load")
		}
		if bmp, ready := entry.Result(); bmp == nil || !ready {
			t.Fatal("local image did not decode")
		}
	}
	t.Run("encoded filename resolves to local file", func(t *testing.T) {
		root := t.TempDir()
		var data bytes.Buffer
		if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "a (1).png"), data.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		entry := loader.Get("a%20%281%29.png", root, func() { close(done) })
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("encoded local image did not load")
		}
		if bitmap, ready := entry.Result(); bitmap == nil || !ready {
			t.Fatalf("encoded image failed: %v", entry.Err())
		}
	})
}

type imageRoundTripper func(*http.Request) (*http.Response, error)

func (f imageRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func pollImagePublication(t *testing.T, release chan struct{}, done <-chan struct{}, poll func()) {
	t.Helper()
	var readers, started sync.WaitGroup
	readers.Add(4)
	started.Add(4)
	for range 4 {
		go func() {
			defer readers.Done()
			poll()
			started.Done()
			for {
				select {
				case <-done:
					return
				default:
					poll()
					runtime.Gosched()
				}
			}
		}()
	}
	started.Wait()
	close(release)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("asynchronous image publication timed out")
	}
	readers.Wait()
}
