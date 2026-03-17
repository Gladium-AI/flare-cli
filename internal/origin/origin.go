package origin

import (
	"context"
	"io"
)

// Type enumerates supported origin backends.
type Type string

const (
	TypeLocalHTTP          Type = "local:http"
	TypeLocalCommand       Type = "local:command"
	TypeDockerContainer    Type = "docker:container"
	TypeDockerCompose      Type = "docker:compose-service"
	TypeBuiltinStatic      Type = "builtin:static"
	TypeBuiltinFileBrowser Type = "builtin:file-browser"
)

// ValidTypes returns all supported origin type strings.
func ValidTypes() []Type {
	return []Type{
		TypeLocalHTTP,
		TypeLocalCommand,
		TypeDockerContainer,
		TypeDockerCompose,
		TypeBuiltinStatic,
		TypeBuiltinFileBrowser,
	}
}

// Config holds origin-specific configuration derived from CLI flags.
type Config struct {
	Type Type

	// local:http
	URL       string // Upstream URL (e.g., http://127.0.0.1:3000)
	HealthURL string // Override health check URL

	// local:command
	Command    string
	Args       []string
	Dir        string
	Env        map[string]string
	Port       int
	HealthPath string
	Shell      bool
	KillSignal string

	// builtin:static / builtin:file-browser
	Path         string // Root directory
	Index        string // Index file (static)
	SPA          bool   // SPA mode (static)
	CacheControl string // Cache-Control header (static)

	// builtin:file-browser
	AllowUpload bool
	AllowDelete bool
	AllowRename bool
	ShowHidden  bool
	Download    bool
	ReadOnly    bool

	// docker:container
	Image         string
	ContainerPort int
	PublishPort   string // e.g., "127.0.0.1:38080"
	Entrypoint    string
	Network       string
	Remove        bool
	DockerBin     string

	// docker:compose-service
	ComposeFile string
	ServiceName string
	ProjectName string
	Build       bool
	UpDetached  bool

	// Shared
	WaitForReady string // Duration string for startup timeout
}

// Origin represents a runnable local application backend.
type Origin interface {
	// Type returns the origin type identifier.
	Type() Type

	// Start launches the origin and returns the loopback URL (http://host:port)
	// it is listening on. Must bind to 127.0.0.1 only.
	// Blocks until the origin is ready to accept connections.
	Start(ctx context.Context) (loopbackURL string, err error)

	// Stop gracefully shuts down the origin.
	Stop(ctx context.Context) error

	// Logs returns a reader that streams the origin's combined output.
	// Returns nil if the origin has no log stream (e.g., local:http proxy).
	Logs() io.ReadCloser

	// Healthy returns nil if the origin is responsive.
	Healthy(ctx context.Context) error
}
