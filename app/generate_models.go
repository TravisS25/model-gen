package app

import (
	"errors"
	"fmt"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kenshaw/snaker"
	"gorm.io/gen"
	"gorm.io/gorm"
)

const (
	packageErr = "model-gen: %w: %s"
)

var (
	sqlAnyMatcher = sqlmock.QueryMatcherFunc(func(expectedSQL, actualSQL string) error {
		return nil
	})

	ErrMustSetSchema    = errors.New("model-gen: 'schema' field must be set when 'driver' field is set to 'postgres'")
	ErrQueryTableNames  = errors.New("query table name error")
	ErrQueryColumnNames = errors.New("query column name error")
	ErrQueryForeignKeys = errors.New("query foreign key error")
)

type GenExecutor interface {
	Execute()
	ApplyBasic(...interface{})
	GenerateModel(string, ...gen.ModelOpt) interface{}
}

type foreignKey struct {
	ColumnName       string
	ForeignTableName string
}

func GenerateModels(g GenExecutor, gormDB *gorm.DB, driver DBDriver, schema string) error {
	var err error
	var tableNames []string

	if driver == PostgresDriver && schema == "" {
		return ErrMustSetSchema
	}

	if err = gormDB.Raw(
		getTableNamesQuery(driver, schema),
	).Scan(&tableNames).Error; err != nil {
		return fmt.Errorf(packageErr, ErrQueryTableNames, err.Error())
	}

	for _, tableName := range tableNames {
		var fks []foreignKey
		var opts []gen.ModelOpt
		var cols []string

		if err = gormDB.Raw(
			getColumnNameQuery(driver, schema, tableName),
		).Scan(&cols).Error; err != nil {
			return fmt.Errorf(packageErr, ErrQueryColumnNames, err.Error())
		}

		for _, col := range cols {
			opts = append(opts, gen.FieldNewTag(col, `db:"`+col+`"`))
		}

		if err = gormDB.Raw(
			getForeignKeyQuery(driver, schema, tableName),
		).Scan(&fks).Error; err != nil {
			return fmt.Errorf(packageErr, ErrQueryForeignKeys, err.Error())
		}

		for _, fk := range fks {
			columnName := fk.ColumnName[:len(fk.ColumnName)-3]
			fieldName := snaker.SnakeToCamel(columnName)
			opts = append(opts, gen.FieldNew(
				fieldName,
				"*"+snaker.SnakeToCamel(fk.ForeignTableName),
				`db:"`+columnName+`" json:"`+columnName+`"`),
			)
		}

		g.ApplyBasic(g.GenerateModel(tableName, opts...))
	}

	g.Execute()
	return nil
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
