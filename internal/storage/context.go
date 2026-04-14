package storage

import "context"

// locKey is an unexported context key type so external packages cannot
// collide with or overwrite the stored Location.
type locKey struct{}

// WithLocation returns a new context carrying loc. The engine attaches this
// before invoking a media provider so the provider (which does not know
// about projects / pipelines / executions) can still persist its output in
// the correct folder.
func WithLocation(ctx context.Context, loc Location) context.Context {
	return context.WithValue(ctx, locKey{}, loc)
}

// LocationFromContext returns the Location carried by ctx and ok=true, or a
// zero Location and ok=false when no Location has been attached.
func LocationFromContext(ctx context.Context) (Location, bool) {
	if ctx == nil {
		return Location{}, false
	}
	loc, ok := ctx.Value(locKey{}).(Location)
	return loc, ok
}
