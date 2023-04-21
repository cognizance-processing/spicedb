package usagemetrics

import (
	"context"

	"spicedb/internal/middleware/usagemetrics"
	dispatch "spicedb/pkg/proto/dispatch/v1"
)

// FromContext returns any metadata that was stored in the context.
func FromContext(ctx context.Context) *dispatch.ResponseMeta {
	return usagemetrics.FromContext(ctx)
}
