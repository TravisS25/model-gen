package app

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/pkg/errors"

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
	ErrQueryTableNames  = errors.New("model-gen: query table name error")
	ErrQueryColumnNames = errors.New("model-gen: query column name error")
	ErrQueryForeignKeys = errors.New("model-gen: query foreign key error")
)

type GenExecutor interface {
	Execute()
	ApplyBasic(...interface{})
	GenerateModel(string, ...gen.ModelOpt) interface{}
}

type GenerateConfig struct {
	OutFile    string
	SingleFile string
}

type foreignKey struct {
	ColumnName       string
	ForeignTableName string
}

type tableColumn struct {
	ColumnName string
	DataType   string
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
		var cols []tableColumn

		if err = gormDB.Raw(
			getColumnNameQuery(driver, schema, tableName),
		).Scan(&cols).Error; err != nil {
			return fmt.Errorf(packageErr, ErrQueryColumnNames, err.Error())
		}

		for _, col := range cols {
			opts = append(
				opts,
				gen.FieldNewTag(col.ColumnName, `db:"`+col.ColumnName+`"`),
				gen.FieldJSONTag(col.ColumnName, snaker.ForceLowerCamelIdentifier(col.ColumnName)),
			)
		}

		if err = gormDB.Raw(
			getForeignKeyQuery(driver, schema, tableName),
		).Scan(&fks).Error; err != nil {
			return fmt.Errorf(packageErr, ErrQueryForeignKeys, err.Error())
		}

		for _, fk := range fks {
			columnName := fk.ColumnName[:len(fk.ColumnName)-3]
			fieldName := snaker.SnakeToCamel(columnName)
			opts = append(
				opts,
				gen.FieldNew(
					fieldName,
					"*"+snaker.SnakeToCamel(fk.ForeignTableName),
					`db:"`+columnName+`" json:"`+snaker.ForceLowerCamelIdentifier(columnName)+`"`,
				),
			)
		}

		g.ApplyBasic(g.GenerateModel(tableName, opts...))
	}

	g.Execute()
	return nil
}

func GenerateTsModels(goModelDir, goOutFile, tsDir, tsFile, tsOutFile string, cfg GenerateConfig) error {
	if tsDir == "" {
		return errors.WithStack(fmt.Errorf("model-gen: tsDir parameter can't be empty"))
	}
	if tsFile == "" {
		return errors.WithStack(fmt.Errorf("model-gen: tsFile parameter can't be empty"))
	}
	if tsOutFile == "" {
		return errors.WithStack(fmt.Errorf("model-gen: tsOutFile parameter can't be empty"))
	}

	strConv := []string{
		"Int64",
		"int64",
		"float64",
		"string",
	}

	numConv := []string{
		"int8",
		"int16",
		"int",
		"int32",
		"float32",
	}

	space := regexp.MustCompile(`\s+`)

	var err error

	if err = os.MkdirAll(tsDir, os.ModePerm); err != nil {
		return errors.WithStack(err)
	}

	newFile, err := os.Create(filepath.Join(tsDir, tsFile) + "." + tsOutFile)

	if err != nil {
		return errors.WithStack(err)
	}

	defer newFile.Close()

	return filepath.Walk(goModelDir, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			if !strings.HasSuffix(info.Name(), goOutFile) {
				return nil
			}

			openFile, err := os.Open(path)

			if err != nil {
				return errors.WithStack(err)
			}

			defer openFile.Close()

			newFileWriter := bufio.NewWriter(newFile)
			openFileReader := bufio.NewReader(openFile)

			withinStruct := false

			for {
				l, err := openFileReader.ReadString('\n')

				if err != nil {
					if err == io.EOF {
						break
					}

					return errors.WithStack(err)
				}

				ajustedLine := strings.TrimSpace(space.ReplaceAllString(l, " "))

				if strings.Contains(ajustedLine, " struct {") {
					structArr := strings.Split(ajustedLine, " ")
					newFileWriter.WriteString(fmt.Sprintf("export interface %s {\n", structArr[1]))
					withinStruct = true
					continue
				}

				if strings.TrimSpace(ajustedLine) == "}" && withinStruct {
					newFileWriter.WriteString("}\n\n")
					break
				}

				if withinStruct {
					lineArr := strings.Split(ajustedLine, " ")

					var fieldType, fieldName string

					if snaker.IsInitialism(lineArr[0]) {
						fieldName = strings.ToLower(lineArr[0])
					} else {
						fn := []rune(lineArr[0])
						fn[0] = unicode.ToLower(fn[0])
						fieldName = string(fn)
					}

					if lineArr[1][0] == '*' {
						fieldType = lineArr[1][1:len(lineArr[1])]
					}

					for _, v := range strConv {
						if strings.Contains(lineArr[1], v) {
							//fmt.Printf("string type: %s\n conv type: %s\n", lineArr[1], v)
							fieldType = "string"
						}
					}

					for _, v := range numConv {
						if strings.Contains(lineArr[1], v) {
							fieldType = "number"
						}
					}

					if lineArr[1] == "bool" || lineArr[1] == "*bool" {
						fieldType = "boolean"
					}

					//fmt.Printf("field type gettign hheeeeeer: %s\n", fieldType)

					if fieldType == "" {
						fieldType = lineArr[1]
					}

					newLine := fmt.Sprintf("\t%s?: %s\n", fieldName, fieldType)

					if _, err = newFileWriter.WriteString(newLine); err != nil {
						return errors.WithStack(err)
					}
				}
			}

			if err = newFileWriter.Flush(); err != nil {
				return errors.WithStack(err)
			}
		}

		return nil
	})
}

func RemoveGenDirs(queryOutPath, modelOutPath string) error {
	var err error

	if queryOutPath == "" {
		queryOutPath = "./query"
	}
	if modelOutPath == "" {
		modelOutPath = "./model"
	}

	if err = os.RemoveAll(queryOutPath); err != nil {
		return errors.WithStack(err)
	}
	if err = os.RemoveAll(modelOutPath); err != nil {
		return errors.WithStack(err)
	}

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
				column_name,
				data_type
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
				column_name,
				data_type
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
				name,
				data_type
			from
				pragma_table_info('%s');
			`,
			tableName,
		)
	}
}
