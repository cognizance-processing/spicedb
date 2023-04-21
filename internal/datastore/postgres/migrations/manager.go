package migrations

import (
	"github.com/jackc/pgx/v5"

	"spicedb/pkg/migrate"
)

var (
	noNonatomicMigration migrate.MigrationFunc[*pgx.Conn]
	noTxMigration        migrate.TxMigrationFunc[pgx.Tx]
)

// DatabaseMigrations implements a migration manager for the Postgres Driver.
var DatabaseMigrations = migrate.NewManager[*AlembicPostgresDriver, *pgx.Conn, pgx.Tx]()
