package ports

import (
	serviceerrors "github.com/king-glitch/hexag/framework/api/service/errors"
	"github.com/pkg/errors"
)

var ErrExampleNotFound = errors.New("example not found")

// init registers this project's domain sentinels with the framework's
// sentinel-to-ServiceErrorCode table, so serviceerrors.ServiceErrorCodeFor
// resolves them without the framework ever needing to know this project's
// error types.
func init() {
	serviceerrors.RegisterSentinel(ErrExampleNotFound, serviceerrors.ServiceErrorCodeNotFound)
}
