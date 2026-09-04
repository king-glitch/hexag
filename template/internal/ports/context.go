package ports

import hexports "github.com/king-glitch/hexag/framework/ports"

type Config struct {
	Port string `env:"APP_PORT"`

	MongoURI      string `env:"APP_DATABASE_MONGO_URI"`
	MongoDatabase string `env:"APP_DATABASE_MONGO_NAME"`
}

// ServiceContext adds this project's own Config() to the framework's base
// ServiceContext (which only guarantees a Logger()) — config shape is
// project-specific, so the framework has no business knowing it.
type ServiceContext interface {
	hexports.ServiceContext
	Config() Config
}
