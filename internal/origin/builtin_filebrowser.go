package origin

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BuiltinFileBrowser serves a web-based file browser.
type BuiltinFileBrowser struct {
	cfg      Config
	server   *http.Server
	listener net.Listener
}

// NewBuiltinFileBrowser creates a file browser origin.
func NewBuiltinFileBrowser(cfg Config) (*BuiltinFileBrowser, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("--path is required for builtin:file-browser origin")
	}

	absPath, err := filepath.Abs(cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("checking path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", absPath)
	}

	cfg.Path = absPath
	return &BuiltinFileBrowser{cfg: cfg}, nil
}

func (fb *BuiltinFileBrowser) Type() Type {
	return TypeBuiltinFileBrowser
}

func (fb *BuiltinFileBrowser) Start(_ context.Context) (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("binding to loopback: %w", err)
	}
	fb.listener = listener

	mux := http.NewServeMux()
	mux.HandleFunc("/", fb.handleBrowse)
	mux.HandleFunc("/api/files", fb.handleAPI)
	if fb.cfg.Download {
		mux.HandleFunc("/download/", fb.handleDownload)
	}
	if fb.cfg.AllowUpload && !fb.cfg.ReadOnly {
		mux.HandleFunc("/upload", fb.handleUpload)
	}

	fb.server = &http.Server{Handler: mux}
	go fb.server.Serve(listener)

	addr := listener.Addr().String()
	return fmt.Sprintf("http://%s", addr), nil
}

func (fb *BuiltinFileBrowser) Stop(ctx context.Context) error {
	if fb.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return fb.server.Shutdown(shutdownCtx)
	}
	return nil
}

func (fb *BuiltinFileBrowser) Logs() io.ReadCloser {
	return nil
}

func (fb *BuiltinFileBrowser) Healthy(_ context.Context) error {
	if fb.listener == nil {
		return fmt.Errorf("server not started")
	}
	return nil
}

type fileEntry struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	IsDir   bool   `json:"is_dir"`
	ModTime string `json:"mod_time"`
}

func (fb *BuiltinFileBrowser) listDir(relPath string) ([]fileEntry, error) {
	absPath := filepath.Join(fb.cfg.Path, filepath.Clean(relPath))

	// Prevent traversal outside root.
	if !strings.HasPrefix(absPath, fb.cfg.Path) {
		return nil, fmt.Errorf("path outside root")
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}

	var files []fileEntry
	for _, e := range entries {
		if !fb.cfg.ShowHidden && strings.HasPrefix(e.Name(), ".") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fileEntry{
			Name:    e.Name(),
			Size:    info.Size(),
			IsDir:   e.IsDir(),
			ModTime: info.ModTime().Format(time.RFC3339),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return files[i].Name < files[j].Name
	})

	return files, nil
}

func (fb *BuiltinFileBrowser) handleAPI(w http.ResponseWriter, r *http.Request) {
	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		relPath = "/"
	}

	files, err := fb.listDir(relPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func (fb *BuiltinFileBrowser) handleDownload(w http.ResponseWriter, r *http.Request) {
	relPath := strings.TrimPrefix(r.URL.Path, "/download")
	absPath := filepath.Join(fb.cfg.Path, filepath.Clean(relPath))

	if !strings.HasPrefix(absPath, fb.cfg.Path) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	http.ServeFile(w, r, absPath)
}

func (fb *BuiltinFileBrowser) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if fb.cfg.ReadOnly {
		http.Error(w, "read-only mode", http.StatusForbidden)
		return
	}

	r.ParseMultipartForm(32 << 20) // 32MB max.
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	dir := r.FormValue("path")
	if dir == "" {
		dir = "/"
	}

	destDir := filepath.Join(fb.cfg.Path, filepath.Clean(dir))
	if !strings.HasPrefix(destDir, fb.cfg.Path) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	destPath := filepath.Join(destDir, filepath.Base(header.Filename))
	out, err := os.Create(destPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer out.Close()

	io.Copy(out, file)
	w.WriteHeader(http.StatusCreated)
}

func (fb *BuiltinFileBrowser) handleBrowse(w http.ResponseWriter, r *http.Request) {
	relPath := r.URL.Path
	if relPath == "" {
		relPath = "/"
	}

	files, err := fb.listDir(relPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Path     string
		Files    []fileEntry
		Download bool
		Upload   bool
		ReadOnly bool
	}{
		Path:     relPath,
		Files:    files,
		Download: fb.cfg.Download,
		Upload:   fb.cfg.AllowUpload && !fb.cfg.ReadOnly,
		ReadOnly: fb.cfg.ReadOnly,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fileBrowserTmpl.Execute(w, data)
}

var fileBrowserTmpl = template.Must(template.New("browser").Parse(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>File Browser - {{.Path}}</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
         max-width: 900px; margin: 0 auto; padding: 20px; color: #1a1a1a; background: #fafafa; }
  h1 { font-size: 1.2em; padding: 12px 0; border-bottom: 1px solid #e0e0e0; margin-bottom: 12px;
       word-break: break-all; }
  table { width: 100%; border-collapse: collapse; }
  th { text-align: left; padding: 8px 12px; border-bottom: 2px solid #e0e0e0; font-size: 0.85em;
       color: #666; text-transform: uppercase; }
  td { padding: 8px 12px; border-bottom: 1px solid #f0f0f0; }
  tr:hover { background: #f5f5f5; }
  a { color: #0066cc; text-decoration: none; }
  a:hover { text-decoration: underline; }
  .dir { font-weight: 600; }
  .size { color: #666; font-size: 0.9em; }
  .time { color: #999; font-size: 0.85em; }
  .actions { font-size: 0.85em; }
</style>
</head>
<body>
<h1>{{.Path}}</h1>
<table>
<thead><tr><th>Name</th><th>Size</th><th>Modified</th>{{if .Download}}<th>Actions</th>{{end}}</tr></thead>
<tbody>
{{if ne .Path "/"}}
<tr><td colspan="4"><a href="../">..</a></td></tr>
{{end}}
{{range .Files}}
<tr>
  <td>{{if .IsDir}}<a class="dir" href="{{.Name}}/">{{.Name}}/</a>
      {{else}}<a href="{{.Name}}">{{.Name}}</a>{{end}}</td>
  <td class="size">{{if .IsDir}}-{{else}}{{.Size}}{{end}}</td>
  <td class="time">{{.ModTime}}</td>
  {{if $.Download}}<td class="actions">{{if not .IsDir}}<a href="/download{{$.Path}}{{.Name}}">download</a>{{end}}</td>{{end}}
</tr>
{{end}}
</tbody>
</table>
</body>
</html>`))
