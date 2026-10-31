package catalog

import (
	"context"

	"github.com/yxxbc/spark-store-tui/internal/domain"
)

// Query expresses source-neutral browsing intent. Providers may reject fields
// their upstream API cannot support rather than silently changing behavior.
type Query struct {
	Architecture string
	Category     string
	Limit        int
}

// Provider normalizes an upstream catalog into the application model.
// Each implementation owns the source's API and JSON shape.
type Provider interface {
	Source() domain.CatalogSource
	ListApps(context.Context, Query) ([]domain.App, error)
}

// CategoryProvider is implemented by metadata catalogs that expose explicit
// category lists. Release-only providers generally do not implement it.
type CategoryProvider interface {
	ListCategories(context.Context) ([]domain.Category, error)
}
