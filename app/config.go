package app

type DBDriver string

var (
	PostgresDriver DBDriver = "postgres"
	SqliteDriver   DBDriver = "sqlite"
	MysqlDriver    DBDriver = "mysql"
)

type GenerateModelConfig struct {
	Driver DBDriver
	URL    string

	FieldNullable     bool
	FieldCoverable    bool
	FieldSignable     bool
	FieldWithIndexTag bool
	FieldWithTypeTag  bool
	OutFile           string
	QueryOutPath      string
	ModelOutPath      string
	Schema            string
}
