package origin

import "fmt"

// New creates an Origin from the given config.
func New(cfg Config) (Origin, error) {
	switch cfg.Type {
	case TypeLocalHTTP:
		return NewLocalHTTP(cfg)
	case TypeLocalCommand:
		return nil, fmt.Errorf("origin type %q not yet implemented", cfg.Type)
	case TypeDockerContainer:
		return nil, fmt.Errorf("origin type %q not yet implemented", cfg.Type)
	case TypeDockerCompose:
		return nil, fmt.Errorf("origin type %q not yet implemented", cfg.Type)
	case TypeBuiltinStatic:
		return nil, fmt.Errorf("origin type %q not yet implemented", cfg.Type)
	case TypeBuiltinFileBrowser:
		return nil, fmt.Errorf("origin type %q not yet implemented", cfg.Type)
	default:
		return nil, fmt.Errorf("unknown origin type: %q", cfg.Type)
	}
}
