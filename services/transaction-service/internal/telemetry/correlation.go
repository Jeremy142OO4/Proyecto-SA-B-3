package telemetry

import "context"

// Correlation returns the operation identity to propagate in internal envelopes.
func Correlation(ctx context.Context) string {
	id, _ := ctx.Value(correlationKey{}).(string)
	return id
}
