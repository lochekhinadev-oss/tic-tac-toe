package datasource

const (
	errDatabaseURLMustNotBeEmpty    = "database url must not be empty"
	errInvalidDatabaseURL           = "invalid database url"
	errOpenDatabase                 = "open database"
	errPingDatabase                 = "ping database"
	errRunMigrations                = "run migrations"
	errParseDatabaseURLForMigration = "parse database url for migrations"
	errPingDatabaseForMigration     = "ping database for migrations"
	errSetGooseDialect              = "set goose dialect"

	msgOpeningDatabaseConnection = "opening database connection"
	msgDatabaseReady             = "database ready"
)
