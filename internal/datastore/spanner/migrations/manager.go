package migrations

import (
	"cloud.google.com/go/spanner"

	"spicedb/pkg/migrate"
)

// SpannerMigrations implements a migration manager for the Spanner datastore.
var SpannerMigrations = migrate.NewManager[*SpannerMigrationDriver, Wrapper, *spanner.ReadWriteTransaction]()
