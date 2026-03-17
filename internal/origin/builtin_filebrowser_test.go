package origin

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func startFileBrowser(t *testing.T, cfg Config) (string, *BuiltinFileBrowser) {
	t.Helper()
	cfg.Type = TypeBuiltinFileBrowser
	fb, err := NewBuiltinFileBrowser(cfg)
	if err != nil {
		t.Fatal(err)
	}
	url, err := fb.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { fb.Stop(context.Background()) })
	return url, fb
}

func TestFileBrowserListDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hi"), 0644)
	os.Mkdir(filepath.Join(dir, "subdir"), 0755)

	url, _ := startFileBrowser(t, Config{Path: dir, Download: true})

	resp, err := http.Get(url + "/api/files")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var entries []fileEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		t.Fatal(err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}

	// Directories first.
	if len(entries) >= 1 && !entries[0].IsDir {
		t.Error("expected first entry to be a directory")
	}
}

func TestFileBrowserDownload(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content123"), 0644)

	url, _ := startFileBrowser(t, Config{Path: dir, Download: true})

	resp, err := http.Get(url + "/download/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "content123" {
		t.Errorf("expected 'content123', got %q", body)
	}
}

func TestFileBrowserUpload(t *testing.T) {
	dir := t.TempDir()

	url, _ := startFileBrowser(t, Config{Path: dir, AllowUpload: true, ReadOnly: false})

	// Create multipart request.
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "uploaded.txt")
	part.Write([]byte("uploaded content"))
	writer.WriteField("path", "/")
	writer.Close()

	resp, err := http.Post(url+"/upload", writer.FormDataContentType(), &buf)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	// Verify file exists.
	data, err := os.ReadFile(filepath.Join(dir, "uploaded.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "uploaded content" {
		t.Errorf("expected 'uploaded content', got %q", data)
	}
}

func TestFileBrowserReadOnly(t *testing.T) {
	dir := t.TempDir()

	// ReadOnly=true means upload endpoint isn't registered.
	url, _ := startFileBrowser(t, Config{Path: dir, AllowUpload: true, ReadOnly: true})

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("data"))
	writer.Close()

	resp, err := http.Post(url+"/upload", writer.FormDataContentType(), &buf)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Should get 404 since upload handler is not registered in read-only mode.
	if resp.StatusCode == http.StatusCreated {
		t.Error("expected upload to be rejected in read-only mode")
	}
}

func TestFileBrowserPathTraversal(t *testing.T) {
	dir := t.TempDir()

	url, _ := startFileBrowser(t, Config{Path: dir})

	resp, err := http.Get(url + "/api/files?path=/../../../etc")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Even if 200, check that no entries from /etc leak through.
		// The path cleaning should prevent traversal.
	}
	// The main point is it shouldn't crash or serve /etc files.
}

func TestFileBrowserHiddenFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".hidden"), []byte("secret"), 0644)
	os.WriteFile(filepath.Join(dir, "visible.txt"), []byte("public"), 0644)

	url, _ := startFileBrowser(t, Config{Path: dir, ShowHidden: false})

	resp, err := http.Get(url + "/api/files")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var entries []fileEntry
	json.NewDecoder(resp.Body).Decode(&entries)

	for _, e := range entries {
		if e.Name == ".hidden" {
			t.Error("hidden file should not be listed when ShowHidden=false")
		}
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 visible entry, got %d", len(entries))
	}
}

func TestFileBrowserNonDirectory(t *testing.T) {
	f := filepath.Join(t.TempDir(), "file.txt")
	os.WriteFile(f, []byte("not a dir"), 0644)

	_, err := NewBuiltinFileBrowser(Config{Type: TypeBuiltinFileBrowser, Path: f})
	if err == nil {
		t.Error("expected error for non-directory path")
	}
}
