package catalog

import "github.com/Xynrin/spark-store-tui/internal/domain"

// BuiltinSources exposes the single catalog used by the client. Download
// mirrors come from this catalog's Metalink documents, not extra app sources.
func BuiltinSources() []domain.CatalogSource {
	return []domain.CatalogSource{
		{
			ID:          "spark-store",
			Name:        "Spark Store",
			Kind:        domain.SourceKindSparkMetadata,
			Description: "Official Spark Store metadata",
			Mirrors: []domain.Mirror{
				{ID: "spark-global", Name: "Official", BaseURL: "https://d.spark-app.store", Region: "global", Priority: 100},
			},
		},
	}
}
