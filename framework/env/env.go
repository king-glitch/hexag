package env

import (
	"github.com/caarlos0/env/v11"
	"github.com/pkg/errors"
)

// LoadConfigFromEnv populates T from its `env:"..."` struct tags. T is the
// project's own Config struct — the framework never sees which env vars a
// given project needs.
func LoadConfigFromEnv[T any]() (T, error) {
	config, err := env.ParseAs[T]()
	if err != nil {
		var zero T
		return zero, errors.Wrap(err, "failed to load config from env")
	}

	return config, nil
}
