/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/TravisS25/model-gen/app"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stretchr/objx"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gen"
	"gorm.io/gorm"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var (
	errRequiredRootFields    = errors.New("model-gen: --driver and --url flags are required if config file is not used")
	errInvalidDriver         = errors.New("model-gen: must choose valid --driver.  Options are 'postgres', 'mysql', 'sqlite'")
	errMustSetSchema         = errors.New("model-gen: --schema flag must be set when --driver is set to 'postgres'")
	errRootKeyNotSet         = errors.New("model-gen: root_cmd key in config file must be set")
	errRootKeyDictionary     = errors.New("model-gen: root_cmd key must be dictionary type")
	errInvalidTsFileSettings = errors.New("model-gen: --ts-dir and --ts-file must be set together")
)

var generateModelCmdCfg = generateModelCmdConfig{
	Driver: flagName{
		LongHand: "driver",
	},
	URL: flagName{
		LongHand: "url",
	},
	FieldNullable: flagName{
		LongHand: "field-nullable",
	},
	FieldCoverable: flagName{
		LongHand: "field-coverable",
	},
	FieldSignable: flagName{
		LongHand: "field-signable",
	},
	FieldWithIndexTag: flagName{
		LongHand: "field-with-index-tag",
	},
	FieldWithTypeTag: flagName{
		LongHand: "field-with-type-tag",
	},
	OutFile: flagName{
		LongHand: "out-file",
	},
	QueryOutPath: flagName{
		LongHand: "query-out-path",
	},
	ModelOutPath: flagName{
		LongHand: "model-out-path",
	},
	Schema: flagName{
		LongHand: "schema",
	},
	ConvertTimestamp: flagName{
		LongHand: "convert-timestamp",
	},
	ConvertDate: flagName{
		LongHand: "convert-date",
	},
	ConvertBigint: flagName{
		LongHand: "convert-bigint",
	},
	ConvertUUID: flagName{
		LongHand: "convert-uuid",
	},
	LanguageType: flagName{
		LongHand: "language-type",
	},
	RemoveGeneratedDirs: flagName{
		LongHand: "remove-generated-dirs",
	},
	TsDir: flagName{
		LongHand: "ts-dir",
	},
	TsFile: flagName{
		LongHand: "ts-file",
	},
	TsOutFile: flagName{
		LongHand: "ts-out-file",
	},
}

var languageTypeMap = map[app.LanguageType]bool{
	app.GoLanguageType: true,
	app.TsLanguageType: true,
}

var dbDriverMap = map[app.DBDriver]bool{
	app.PostgresDriver: true,
	app.MysqlDriver:    true,
	app.SqliteDriver:   true,
}

type rootViperConfig struct {
	dir  string
	file string
}

type rootCliConfig struct {
	driver app.DBDriver
	url    string
	schema string
	tsDir  string
	tsFile string
}

type rootValidationConfig struct {
	configFile string
	cli        rootCliConfig
}

type generateModelCmdConfig struct {
	Driver flagName
	URL    flagName

	FieldNullable       flagName
	FieldCoverable      flagName
	FieldSignable       flagName
	FieldWithIndexTag   flagName
	FieldWithTypeTag    flagName
	OutFile             flagName
	QueryOutPath        flagName
	ModelOutPath        flagName
	Schema              flagName
	ConvertTimestamp    flagName
	ConvertDate         flagName
	ConvertBigint       flagName
	ConvertUUID         flagName
	LanguageType        flagName
	RemoveGeneratedDirs flagName
	TsDir               flagName
	TsFile              flagName
	TsOutFile           flagName
}

// generator is a "wrapper" struct used to simply override the "GenerateModel" function
// from the gen.Generator struct
//
// Reason for this is that "GenerateModel" function returns type *generate.QueryStructMeta
// which lives within the "internal" folder of the library meaning users can't
// actually call it so it makes mocking for tests impossible
//
// So this struct wraps the *gen.Generator and overrides the "GenerateModel" function
// and returns interface{}
type generator struct {
	*gen.Generator
}

func (g *generator) GenerateModel(model string, opts ...gen.ModelOpt) interface{} {
	return g.Generator.GenerateModel(model, opts...)
}

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "model-gen",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },

	PreRunE: func(cmd *cobra.Command, args []string) error {
		driver, _ := cmd.Flags().GetString(generateModelCmdCfg.Driver.LongHand)
		url, _ := cmd.Flags().GetString(generateModelCmdCfg.URL.LongHand)
		schema, _ := cmd.Flags().GetString(generateModelCmdCfg.Schema.LongHand)
		tsDir, _ := cmd.Flags().GetString(generateModelCmdCfg.Schema.LongHand)
		tsFile, _ := cmd.Flags().GetString(generateModelCmdCfg.Schema.LongHand)

		return rootCmdPreRunValidation(rootValidationConfig{
			cli: rootCliConfig{
				driver: app.DBDriver(driver),
				url:    url,
				schema: schema,
				tsDir:  tsDir,
				tsFile: tsFile,
			},
		})
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var cfg gen.Config
		var gormDB *gorm.DB
		var err error
		var removeGenDirs, fieldNullable, fieldCoverable, fieldSignable, fieldWithIndexTag,
			fieldWithTypeTag bool
		var url, driver, schema, convertTimestamp, convertDate, convertBigint,
			convertUUID, outFile, queryOutPath string
		var modelOutPath, tsDir, tsFile, tsOutFile string

		if err = viper.ReadInConfig(); err == nil {
			rootCmd := objx.New(viper.Get("root_cmd").(map[string]interface{}))

			fieldNullable = rootCmd.Get("field_nullable").Bool()
			fieldCoverable = rootCmd.Get("field_coverable").Bool()
			fieldSignable = rootCmd.Get("field_signable").Bool()
			fieldWithIndexTag = rootCmd.Get("field_with_index_tag").Bool()
			fieldWithTypeTag = rootCmd.Get("field_with_type_tag").Bool()
			removeGenDirs = rootCmd.Get("remove_generated_dirs").Bool()

			driver = rootCmd.Get("driver").Str()
			url = rootCmd.Get("url").Str()
			schema = rootCmd.Get("schema").Str()
			outFile = rootCmd.Get("out_file").Str()
			queryOutPath = rootCmd.Get("query_out_path").Str()
			modelOutPath = rootCmd.Get("model_out_path").Str()
			convertTimestamp = rootCmd.Get("convert_timestamp").Str()
			convertDate = rootCmd.Get("convert_date").Str()
			convertBigint = rootCmd.Get("convert_bigint").Str()
			convertUUID = rootCmd.Get("convert_uuid").Str()
			tsDir = rootCmd.Get("ts_dir").Str()
			tsFile = rootCmd.Get("ts_file").Str()
			tsOutFile = rootCmd.Get("ts_out_file").Str()
		}

		fieldNullableTmp, _ := cmd.Flags().GetBool(generateModelCmdCfg.FieldNullable.LongHand)
		fieldCoverableTmp, _ := cmd.Flags().GetBool(generateModelCmdCfg.FieldCoverable.LongHand)
		fieldSignableTmp, _ := cmd.Flags().GetBool(generateModelCmdCfg.FieldSignable.LongHand)
		fieldWithIndexTagTmp, _ := cmd.Flags().GetBool(generateModelCmdCfg.FieldWithIndexTag.LongHand)
		fieldWithTypeTagTmp, _ := cmd.Flags().GetBool(generateModelCmdCfg.FieldWithTypeTag.LongHand)
		removeGenDirsTmp, _ := cmd.Flags().GetBool(generateModelCmdCfg.RemoveGeneratedDirs.LongHand)

		driverTmp, _ := cmd.Flags().GetString(generateModelCmdCfg.Driver.LongHand)
		urlTmp, _ := cmd.Flags().GetString(generateModelCmdCfg.URL.LongHand)
		schemaTmp, _ := cmd.Flags().GetString(generateModelCmdCfg.Schema.LongHand)
		outFileTmp, _ := cmd.Flags().GetString(generateModelCmdCfg.OutFile.LongHand)
		queryOutPathTmp, _ := cmd.Flags().GetString(generateModelCmdCfg.QueryOutPath.LongHand)
		modelOutPathTmp, _ := cmd.Flags().GetString(generateModelCmdCfg.ModelOutPath.LongHand)
		convertTimestampTmp, _ := cmd.Flags().GetString(generateModelCmdCfg.ConvertTimestamp.LongHand)
		convertDateTmp, _ := cmd.Flags().GetString(generateModelCmdCfg.ConvertDate.LongHand)
		convertBigintTmp, _ := cmd.Flags().GetString(generateModelCmdCfg.ConvertBigint.LongHand)
		convertUUIDTmp, _ := cmd.Flags().GetString(generateModelCmdCfg.ConvertUUID.LongHand)
		tsDirTmp, _ := cmd.Flags().GetString(generateModelCmdCfg.TsDir.LongHand)
		tsFileTmp, _ := cmd.Flags().GetString(generateModelCmdCfg.TsFile.LongHand)
		tsOutFileTmp, _ := cmd.Flags().GetString(generateModelCmdCfg.TsOutFile.LongHand)

		if fieldNullableTmp {
			fieldNullable = fieldNullableTmp
		}
		if fieldCoverableTmp {
			fieldCoverable = fieldCoverableTmp
		}
		if fieldSignableTmp {
			fieldSignable = fieldSignableTmp
		}
		if fieldWithIndexTagTmp {
			fieldWithIndexTag = fieldWithIndexTagTmp
		}
		if fieldWithTypeTagTmp {
			fieldWithTypeTag = fieldWithTypeTagTmp
		}
		if removeGenDirsTmp {
			removeGenDirs = removeGenDirsTmp
		}

		if driverTmp != "" {
			driver = driverTmp
		}
		if urlTmp != "" {
			url = urlTmp
		}
		if schemaTmp != "" {
			schema = schemaTmp
		}
		if outFileTmp != "" {
			outFile = outFileTmp
		}
		if queryOutPathTmp != "" {
			queryOutPath = queryOutPathTmp
		}
		if modelOutPathTmp != "" {
			modelOutPath = modelOutPathTmp
		}
		if convertTimestampTmp != "" {
			convertTimestamp = convertTimestampTmp
		}
		if convertDateTmp != "" {
			convertDate = convertDateTmp
		}
		if convertBigintTmp != "" {
			convertBigint = convertBigintTmp
		}
		if convertUUIDTmp != "" {
			convertUUID = convertUUIDTmp
		}
		if tsDirTmp != "" {
			tsDir = tsDirTmp
		}
		if tsFileTmp != "" {
			tsFile = tsFileTmp
		}
		if tsOutFileTmp != "" {
			tsOutFile = tsOutFileTmp
		}

		cfg = gen.Config{
			FieldNullable:     fieldNullable,
			FieldCoverable:    fieldCoverable,
			FieldSignable:     fieldSignable,
			FieldWithIndexTag: fieldWithIndexTag,
			FieldWithTypeTag:  fieldWithTypeTag,
			OutFile:           outFile,
			OutPath:           queryOutPath,
			ModelPkgPath:      modelOutPath,
		}

		switch app.DBDriver(driver) {
		case app.PostgresDriver:
			gormDB, err = gorm.Open(postgres.Open(url))
		case app.MysqlDriver:
			gormDB, err = gorm.Open(mysql.Open(url))
		default:
			gormDB, err = gorm.Open(sqlite.Open(url))
		}

		if err != nil {
			return fmt.Errorf("model-gen: init db err: %s\n", err.Error())
		}

		dataMap := map[string]func(detailType string) (dataType string){}

		// Convert any types given from cli or file to desired types
		if convertTimestamp != "" {
			dataMap["timestamptz"] = func(detailType string) (dataType string) {
				return convertTimestamp
			}
		}
		if convertDate != "" {
			dataMap["date"] = func(detailType string) (dataType string) {
				return convertDate
			}
		}
		if convertBigint != "" {
			dataMap["int8"] = func(detailType string) (dataType string) {
				return convertBigint
			}
		}
		if convertUUID != "" {
			dataMap["uuid"] = func(detailType string) (dataType string) {
				return convertUUID
			}
		}

		g := gen.NewGenerator(cfg)
		g.UseDB(gormDB)
		g.WithDataTypeMap(dataMap)

		if err = app.GenerateModels(
			&generator{Generator: g},
			gormDB,
			app.DBDriver(driver),
			schema,
		); err != nil {
			return errors.WithStack(err)
		}

		nonGoOutput := false

		if tsDir != "" && tsFile != "" {
			nonGoOutput = true

			fmt.Printf("Generating ts files....\n")
			fmt.Printf("%s", tsOutFile)

			if err = app.GenerateTsModels(
				modelOutPath,
				outFile,
				tsDir,
				tsFile,
				tsOutFile,
				app.GenerateConfig{},
			); err != nil {
				return errors.WithStack(err)
			}
		}

		if nonGoOutput && removeGenDirs {
			if err = app.RemoveGenDirs(queryOutPath, modelOutPath); err != nil {
				return err
			}
		}

		return nil
	},
}

func getDBFromDriver(driver app.DBDriver, url string) (*gorm.DB, error) {
	var gormDB *gorm.DB
	var err error

	switch app.DBDriver(driver) {
	case app.PostgresDriver:
		gormDB, err = gorm.Open(postgres.Open(url))
	case app.MysqlDriver:
		gormDB, err = gorm.Open(mysql.Open(url))
	default:
		gormDB, err = gorm.Open(sqlite.Open(url))
	}

	if err != nil {
		return nil, fmt.Errorf("model-gen: init db err: %s\n", err.Error())
	}

	return gormDB, err
}

func rootCmdPreRunValidation(cfg rootValidationConfig) error {
	var err error
	var ok bool
	var rootCmdMap map[string]interface{}

	if err = viper.ReadInConfig(); err == nil {
		rootCmdFromFile := viper.Get("root_cmd")

		if rootCmdFromFile == nil {
			return errors.WithStack(errRootKeyNotSet)
		}

		if rootCmdMap, ok = rootCmdFromFile.(map[string]interface{}); !ok {
			return errors.WithStack(errRootKeyDictionary)
		}
	} else {
		rootCmdMap = make(map[string]interface{})
	}

	rootCmdObjx := objx.New(rootCmdMap)

	driver := app.DBDriver(rootCmdObjx.Get("driver").Str())
	schema := rootCmdObjx.Get("schema").Str()
	url := rootCmdObjx.Get("url").Str()
	tsDir := rootCmdObjx.Get("ts_dir").Str()
	tsFile := rootCmdObjx.Get("ts_file").Str()

	if cfg.cli.driver != "" {
		driver = cfg.cli.driver
	}
	if cfg.cli.url != "" {
		url = cfg.cli.url
	}
	if cfg.cli.schema != "" {
		schema = cfg.cli.schema
	}
	if cfg.cli.tsDir != "" {
		tsDir = cfg.cli.tsDir
	}
	if cfg.cli.tsFile != "" {
		tsFile = cfg.cli.tsFile
	}

	if driver == "" || url == "" {
		return errors.WithStack(errRequiredRootFields)
	}

	if _, ok = dbDriverMap[driver]; !ok {
		return errors.WithStack(errInvalidDriver)
	}

	if driver == app.PostgresDriver && schema == "" {
		return errors.WithStack(errMustSetSchema)
	}

	if (tsDir != "" && tsFile == "") || (tsDir == "" && tsFile != "") {
		return errors.WithStack(errInvalidTsFileSettings)
	}

	return nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.model_gen.yaml)")
	rootCmd.PersistentFlags().String(
		generateModelCmdCfg.Driver.LongHand,
		"",
		"Database driver to connect to database.  Options are postgres, mysql, sqlite",
	)
	rootCmd.PersistentFlags().String(
		generateModelCmdCfg.URL.LongHand,
		"",
		"DSN of the database you want to connect to",
	)
	rootCmd.PersistentFlags().Bool(
		generateModelCmdCfg.FieldNullable.LongHand,
		false,
		"Generate pointer when field is nullable",
	)
	rootCmd.PersistentFlags().Bool(
		generateModelCmdCfg.FieldCoverable.LongHand,
		false,
		"Generate pointer when field has default value, to fix problem zero value cannot be assign",
	)
	rootCmd.PersistentFlags().Bool(
		generateModelCmdCfg.FieldSignable.LongHand,
		false,
		"Detect integer field's unsigned type, adjust generated data type",
	)
	rootCmd.PersistentFlags().Bool(
		generateModelCmdCfg.FieldWithIndexTag.LongHand,
		false,
		"Generate with gorm index tag",
	)
	rootCmd.PersistentFlags().Bool(
		generateModelCmdCfg.FieldWithTypeTag.LongHand,
		false,
		"Generate with gorm column type tag",
	)
	rootCmd.PersistentFlags().String(
		generateModelCmdCfg.OutFile.LongHand,
		"gen.go",
		"Query code file name for go",
	)
	rootCmd.PersistentFlags().String(
		generateModelCmdCfg.TsOutFile.LongHand,
		"gen.ts",
		"Query code file name for ts",
	)
	rootCmd.PersistentFlags().String(
		generateModelCmdCfg.QueryOutPath.LongHand,
		"",
		"Path the query code will be generated to.  Can be relative path of where model-gen is executed",
	)
	rootCmd.PersistentFlags().String(
		generateModelCmdCfg.ModelOutPath.LongHand,
		"",
		"Path the model code will be generated to.  Can be relative path of where model-gen is executed",
	)
	rootCmd.PersistentFlags().String(
		generateModelCmdCfg.Schema.LongHand,
		"",
		"Schema to base model generation off.  Required if driver is 'postgres'",
	)
	rootCmd.PersistentFlags().String(
		generateModelCmdCfg.ConvertTimestamp.LongHand,
		"",
		"Converts any db fields with timestamp data type to one entered",
	)
	rootCmd.PersistentFlags().Bool(
		generateModelCmdCfg.RemoveGeneratedDirs.LongHand,
		false,
		`This option will allow cleanup of the generated go files that are created when converting from go
		to whatever language selected`,
	)

	rootCmd.MarkFlagRequired(generateModelCmdCfg.Driver.LongHand)
	rootCmd.MarkFlagRequired(generateModelCmdCfg.URL.LongHand)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".modelgen" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".model_gen")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
