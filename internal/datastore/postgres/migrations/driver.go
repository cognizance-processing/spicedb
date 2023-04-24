package migrations

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"spicedb/pkg/migrate"
)

const postgresMissingTableErrorCode = "42P01"

// AlembicPostgresDriver implements a schema migration facility for use in
// SpiceDB's Postgres datastore.
//
// It is compatible with the popular Python library, Alembic
type AlembicPostgresDriver struct {
	db *pgx.Conn
}
type Config struct {
	ServerPort string
	FontEndUrl string

	// main db
	DatabaseName           string
	DatabaseUser           string
	DatabasePassword       string
	InstanceConnectionName string
	SpiceDBSharedKey       string
}

func GetConfig(configFileName *string) (*Config, error) {
	// set places to look for config file
	// cloud run
	viper.AddConfigPath("../../config")
	viper.AddConfigPath("../config")
	viper.AddConfigPath("./config")

	// set the name of the config file
	viper.SetConfigName(*configFileName)
	if err := viper.ReadInConfig(); err != nil {
		log.Error().Err(err).Msgf("could not parse config file")
		return nil, err
	}

	// parse the config file
	cfg := new(Config)
	if err := viper.Unmarshal(cfg); err != nil {
		log.Error().Err(err).Msg("unmarshalling config file")
		return nil, err
	}

	return cfg, nil
}

// NewAlembicPostgresDriver creates a new driver with active connections to the database specified.
func NewAlembicPostgresDriver(url string) (*AlembicPostgresDriver, error) {
	var configFileName = "config.toml"
	config2, err := GetConfig(&configFileName)
	if err != nil {
		log.Fatal().Err(err).Msg("getting config from file")
	}
	var (
		dbUser                 = config2.DatabaseUser           // e.g. 'my-db-user'
		dbPwd                  = config2.DatabasePassword       // e.g. 'my-db-password'
		dbName                 = config2.DatabaseName           // e.g. 'my-database'
		instanceConnectionName = config2.InstanceConnectionName // e.g. 'project:region:instance'
		//usePrivate             = os.Getenv("PRIVATE_IP")
	)

	dsn := fmt.Sprintf("user=%s password=%s database=%s host=%s", dbUser, dbPwd, dbName, instanceConnectionName)

	db, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		return nil, err
	}

	return &AlembicPostgresDriver{db}, nil
}

// Conn returns the underlying pgx.Conn instance for this driver
func (apd *AlembicPostgresDriver) Conn() *pgx.Conn {
	return apd.db
}

func (apd *AlembicPostgresDriver) RunTx(ctx context.Context, f migrate.TxMigrationFunc[pgx.Tx]) error {
	return pgx.BeginFunc(ctx, apd.db, func(tx pgx.Tx) error {
		return f(ctx, tx)
	})
}

// Version returns the version of the schema to which the connected database
// has been migrated.
func (apd *AlembicPostgresDriver) Version(ctx context.Context) (string, error) {
	var loaded string

	if err := apd.db.QueryRow(ctx, "SELECT version_num from alembic_version").Scan(&loaded); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == postgresMissingTableErrorCode {
			return "", nil
		}
		return "", fmt.Errorf("unable to load alembic revision: %w", err)
	}

	return loaded, nil
}

// Close disposes the driver.
func (apd *AlembicPostgresDriver) Close(ctx context.Context) error {
	return apd.db.Close(ctx)
}

func (apd *AlembicPostgresDriver) WriteVersion(ctx context.Context, tx pgx.Tx, version, replaced string) error {
	result, err := tx.Exec(
		ctx,
		"UPDATE alembic_version SET version_num=$1 WHERE version_num=$2",
		version,
		replaced,
	)
	if err != nil {
		return fmt.Errorf("unable to update version row: %w", err)
	}

	updatedCount := result.RowsAffected()
	if updatedCount != 1 {
		return fmt.Errorf("writing version update affected %d rows, should be 1", updatedCount)
	}

	return nil
}

var _ migrate.Driver[*pgx.Conn, pgx.Tx] = &AlembicPostgresDriver{}
