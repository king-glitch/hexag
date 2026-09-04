package ports

import (
	hexports "github.com/king-glitch/hexag/framework/ports"
	"github.com/pkg/errors"
)

var ErrExampleNotFound = errors.New("example not found")

// init registers this project's domain sentinels with the framework's
// sentinel-to-ServiceErrorCode table, so hexports.ServiceErrorCodeFor
// resolves them without the framework ever needing to know this project's
// error types.
func init() {
	hexports.RegisterSentinel(ErrExampleNotFound, hexports.ServiceErrorCodeNotFound)
}
