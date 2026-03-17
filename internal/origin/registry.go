package origin

import "fmt"

// New creates an Origin from the given config.
func New(cfg Config) (Origin, error) {
	switch cfg.Type {
	case TypeLocalHTTP:
		return NewLocalHTTP(cfg)
	case TypeLocalCommand:
		return NewLocalCommand(cfg)
	case TypeDockerContainer:
		return NewDockerContainer(cfg)
	case TypeDockerCompose:
		return NewDockerCompose(cfg)
	case TypeBuiltinStatic:
		return NewBuiltinStatic(cfg)
	case TypeBuiltinFileBrowser:
		return NewBuiltinFileBrowser(cfg)
	default:
		return nil, fmt.Errorf("unknown origin type: %q", cfg.Type)
	}
}
