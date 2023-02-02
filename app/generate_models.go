package app

import (
	"fmt"
	"log"

	"github.com/kenshaw/snaker"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gen"
	"gorm.io/gorm"
)

type foreignKey struct {
	ColumnName       string
	ForeignTableName string
}

func GenerateModels(cfg GenerateModelConfig) {
	var err error
	var tableNames []string
	var gormDB *gorm.DB

	g := gen.NewGenerator(gen.Config{
		FieldNullable:     cfg.FieldNullable,
		FieldCoverable:    cfg.FieldCoverable,
		FieldSignable:     cfg.FieldSignable,
		FieldWithIndexTag: cfg.FieldWithIndexTag,
		FieldWithTypeTag:  cfg.FieldWithTypeTag,
		OutFile:           cfg.OutFile,
		OutPath:           cfg.QueryOutPath,
		ModelPkgPath:      cfg.ModelOutPath,
	})

	switch cfg.Driver {
	case PostgresDriver:
		gormDB, err = gorm.Open(postgres.Open(cfg.URL))
	case MysqlDriver:
		gormDB, err = gorm.Open(mysql.Open(cfg.URL))
	case SqliteDriver:
		gormDB, err = gorm.Open(sqlite.Open(cfg.URL))
	}

	if err != nil {
		log.Fatalf("model-gen: init db err: %s\n", err.Error())
	}

	g.UseDB(gormDB)

	if err = gormDB.Raw(
		getTableNamesQuery(cfg.Driver, cfg.Schema),
	).Scan(&tableNames).Error; err != nil {
		log.Fatalf("model-gen: query table name err: %s\n", err.Error())
	}

	for _, tableName := range tableNames {
		var fks []foreignKey
		var opts []gen.ModelOpt
		var cols []string

		if err = gormDB.Raw(
			getColumnNameQuery(cfg.Driver, cfg.Schema, tableName),
		).Scan(&cols).Error; err != nil {
			log.Fatalf("column query error: %+v\n", err)
		}

		for _, col := range cols {
			opts = append(opts, gen.FieldNewTag(col, `db:"`+col+`"`))
		}

		if err = gormDB.Raw(
			getForeignKeyQuery(cfg.Driver, cfg.Schema, tableName),
		).Scan(&fks).Error; err != nil {
			log.Fatalf("foreign key error: %+v\n", err)
		}

		for _, fk := range fks {
			columnName := fk.ColumnName[:len(fk.ColumnName)-3]
			fieldName := snaker.SnakeToCamel(columnName)
			opts = append(opts, gen.FieldNew(fieldName, "*"+snaker.SnakeToCamel(fk.ForeignTableName), `db:"`+columnName+`"`))
		}

		g.ApplyBasic(g.GenerateModel(tableName, opts...))
	}

	g.Execute()
}

func getTableNamesQuery(driver DBDriver, schema string) string {
	switch driver {
	case PostgresDriver:
		return fmt.Sprintf(
			`
			select
				table_name
			from
				information_schema.tables
			where
				table_schema = '%s'
			`,
			schema,
		)
	case MysqlDriver:
		return `
		select
			table_name
		from
			information_schema.tables
		`
	default:
		return `
		select
    		name
		from
			sqlite_schema
		where
			type ='table'
		and
			name NOT LIKE 'sqlite_%';
		`
	}
}

func getForeignKeyQuery(driver DBDriver, schema, tableName string) string {
	switch driver {
	case PostgresDriver:
		return fmt.Sprintf(
			`
			select
				kcu.column_name,
				ccu.table_name AS "foreign_table_name"
			from
				information_schema.table_constraints AS tc
				JOIN information_schema.key_column_usage AS kcu
				ON tc.constraint_name = kcu.constraint_name
				and tc.table_schema = kcu.table_schema
				JOIN information_schema.constraint_column_usage AS ccu
				ON ccu.constraint_name = tc.constraint_name
				and ccu.table_schema = tc.table_schema
			where
				tc.table_schema = '%s'
			and
				tc.constraint_type = 'FOREIGN KEY'
			and
				tc.table_name='%s';
			`,
			schema,
			tableName,
		)
	case MysqlDriver:
		return fmt.Sprintf(
			`
			select
				column_name,
				referenced_table_name as "foreign_table_name"
			from
				information_schema.key_column_usgae
			where
				table_name = '%s'
			and
				referenced_table_name is not null;
			`,
			tableName,
		)
	default:
		return fmt.Sprintf(
			`
			select
				"from" as "column_name",
				"table" as "foreign_table_name"
			from
				pragma_foreign_key_list('%s');
			`,
			tableName,
		)
	}
}

func getColumnNameQuery(driver DBDriver, schema, tableName string) string {
	switch driver {
	case PostgresDriver:
		return fmt.Sprintf(
			`
			select
				column_name
			from
				information_schema.columns
			where
				table_schema = '%s'
			and
				table_name   = '%s';
			`,
			schema,
			tableName,
		)
	case MysqlDriver:
		return fmt.Sprintf(
			`
			select
				column_name
			from
				information_schema.columns
			where
				table_name = '%s';
			`,
			tableName,
		)
	default:
		return fmt.Sprintf(
			`
			select
				name
			from
				pragma_table_info('%s');
			`,
			tableName,
		)
	}
}
