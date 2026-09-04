package context

import (
	"{{MODULE_PATH}}/internal/ports"

	hexenv "github.com/king-glitch/hexag/framework/env"
	"github.com/pkg/errors"
)

// LoadConfigFromEnv populates ports.Config from its `env:"..."` struct tags —
// the tags on ports.Config are the single source of truth for env var names.
func LoadConfigFromEnv() (ports.Config, error) {
	config, err := hexenv.LoadConfigFromEnv[ports.Config]()
	if err != nil {
		return ports.Config{}, errors.Wrap(err, "failed to load config from env")
	}

	return config, nil
}
