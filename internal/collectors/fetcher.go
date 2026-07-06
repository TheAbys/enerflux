package collectors

import "context"

type Fetcher interface {
	Fetch(ctx context.Context) ([]byte, error)
}
