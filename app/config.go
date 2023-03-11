package app

type DBDriver string

var (
	PostgresDriver DBDriver = "postgres"
	SqliteDriver   DBDriver = "sqlite"
	MysqlDriver    DBDriver = "mysql"
)
