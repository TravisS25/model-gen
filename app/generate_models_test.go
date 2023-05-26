package app

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gen"
	"gorm.io/gorm"
)

type mockGenerator struct{}

func (m *mockGenerator) Execute()                  {}
func (m *mockGenerator) ApplyBasic(...interface{}) {}
func (m *mockGenerator) GenerateModel(model string, opts ...gen.ModelOpt) interface{} {
	return nil
}

func TestGenerateModels(t *testing.T) {
	var err error
	var gormDB *gorm.DB
	var db *sql.DB
	var mockDB sqlmock.Sqlmock
	var tableRows, userColumnRows, phoneForeignKeyRows, phoneColumnRows, userForeignKeyRows *sqlmock.Rows

	initNewMockDB := func() {
		var innerErr error

		if db != nil {
			db.Close()
		}

		db, mockDB, innerErr = sqlmock.New(sqlmock.QueryMatcherOption(sqlAnyMatcher))

		if innerErr != nil {
			t.Fatalf(innerErr.Error())
		}

		if gormDB, innerErr = gorm.Open(postgres.New(postgres.Config{Conn: db})); innerErr != nil {
			t.Fatalf(innerErr.Error())
		}
	}

	mockGen := &mockGenerator{}

	initNewMockDB()

	if err = GenerateModels(mockGen, gormDB, PostgresDriver, ""); err == nil {
		t.Fatalf("should have error\n")
	}

	if !errors.Is(err, ErrMustSetSchema) {
		t.Fatalf("should have error %v; got %v\n", ErrMustSetSchema, err)
	}

	sqlErr := errors.New("error")

	mockDB.ExpectQuery("").WillReturnError(sqlErr)

	if err = GenerateModels(mockGen, gormDB, PostgresDriver, "public"); err == nil {
		t.Fatalf("should have error\n")
	}

	if !errors.Is(err, ErrQueryTableNames) {
		t.Fatalf("should have error %v; got %v\n", ErrQueryTableNames, err)
	}

	initNewMockDB()
	tableRows = mockDB.NewRows([]string{"name"}).AddRow("user_profile").AddRow("phone")

	mockDB.ExpectQuery("").WillReturnRows(tableRows)
	mockDB.ExpectQuery("").WillReturnError(sqlErr)

	if err = GenerateModels(mockGen, gormDB, PostgresDriver, "public"); err == nil {
		t.Fatalf("should not have error\n")
	}

	if !errors.Is(err, ErrQueryColumnNames) {
		t.Fatalf("should have error %v; got %v\n", ErrQueryColumnNames, err)
	}

	initNewMockDB()
	tableRows = mockDB.NewRows([]string{"name"}).AddRow("phone").AddRow("user_profile")
	phoneColumnRows = mockDB.NewRows([]string{"cols"}).AddRow("id").AddRow("number").AddRow("user_profile_id")

	mockDB.ExpectQuery("select name from phone table").WillReturnRows(tableRows)
	mockDB.ExpectQuery("select columns from phone table").WillReturnRows(phoneColumnRows)
	mockDB.ExpectQuery("select foreignkey from phone table").WillReturnError(sqlErr)

	if err = GenerateModels(mockGen, gormDB, PostgresDriver, "public"); err == nil {
		t.Fatalf("should have error\n")
	}

	if !errors.Is(err, ErrQueryForeignKeys) {
		t.Fatalf("should have error %v; got %v\n", ErrQueryForeignKeys, err)
	}

	initNewMockDB()
	tableRows = mockDB.NewRows([]string{"name"}).AddRow("user_profile").AddRow("phone")
	phoneColumnRows = mockDB.NewRows([]string{"cols"}).AddRow("id").AddRow("number").AddRow("user_profile_id")
	phoneForeignKeyRows = mockDB.NewRows([]string{"column_name", "foreign_table_name"}).AddRow("user_profile_id", "user_profile")
	userColumnRows = mockDB.NewRows([]string{"col"}).AddRow("id").AddRow("name")
	userForeignKeyRows = mockDB.NewRows([]string{"column_name", "foreign_table_name"})

	mockDB.ExpectQuery("select name from tables").WillReturnRows(tableRows)
	mockDB.ExpectQuery("select columns from phone table").WillReturnRows(phoneColumnRows)
	mockDB.ExpectQuery("select foreignkey from phone table").WillReturnRows(phoneForeignKeyRows)
	mockDB.ExpectQuery("select columns from user table").WillReturnRows(userColumnRows)
	mockDB.ExpectQuery("select columns from user foreign keys table").WillReturnRows(userForeignKeyRows)

	if err = GenerateModels(mockGen, gormDB, PostgresDriver, "public"); err != nil {
		t.Fatalf("should not have error; %s\n", err.Error())
	}

}

func TestGenerateTsModels(t *testing.T) {
	var err error

	if err = GenerateTsModels(
		"/home/travis/programming/projects/pac-env/app/server/model/",
		"gen.go",
		"/home/travis/programming/projects/pac-env/app/web/src/",
		"model",
		"gen.ts",
		GenerateConfig{},
	); err != nil {
		t.Fatalf(err.Error())
	}

	t.Fatalf("boom")
}
