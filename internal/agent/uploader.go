package agent

import (
	"context"

	"github.com/theabys/enerflux/internal/measurements"
)

type Uploader interface {
	Upload(ctx context.Context, measurements []measurements.Measurement) error
}
