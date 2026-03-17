package origin

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestStaticServesFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>Hello</h1>"), 0644)

	o, err := NewBuiltinStatic(Config{Type: TypeBuiltinStatic, Path: dir})
	if err != nil {
		t.Fatal(err)
	}

	url, err := o.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer o.Stop(context.Background())

	resp, err := http.Get(url + "/index.html")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "<h1>Hello</h1>" {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestStaticSPAFallback(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("SPA Root"), 0644)

	o, err := NewBuiltinStatic(Config{Type: TypeBuiltinStatic, Path: dir, SPA: true})
	if err != nil {
		t.Fatal(err)
	}

	url, err := o.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer o.Stop(context.Background())

	resp, err := http.Get(url + "/nonexistent-route")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "SPA Root" {
		t.Errorf("SPA fallback expected 'SPA Root', got: %s", body)
	}
}

func TestStaticCacheControl(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "style.css"), []byte("body{}"), 0644)

	o, err := NewBuiltinStatic(Config{
		Type:         TypeBuiltinStatic,
		Path:         dir,
		CacheControl: "public, max-age=3600",
	})
	if err != nil {
		t.Fatal(err)
	}

	url, err := o.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer o.Stop(context.Background())

	resp, err := http.Get(url + "/style.css")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	cc := resp.Header.Get("Cache-Control")
	if cc != "public, max-age=3600" {
		t.Errorf("expected Cache-Control 'public, max-age=3600', got %q", cc)
	}
}

func TestStaticNonDirectory(t *testing.T) {
	f := filepath.Join(t.TempDir(), "file.txt")
	os.WriteFile(f, []byte("not a dir"), 0644)

	_, err := NewBuiltinStatic(Config{Type: TypeBuiltinStatic, Path: f})
	if err == nil {
		t.Error("expected error for non-directory path")
	}
}
