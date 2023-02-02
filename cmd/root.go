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
	"errors"
	"fmt"
	"os"

	"github.com/TravisS25/model-gen/app"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

type generateModelCmdConfig struct {
	Driver flagName
	URL    flagName

	FieldNullable     flagName
	FieldCoverable    flagName
	FieldSignable     flagName
	FieldWithIndexTag flagName
	FieldWithTypeTag  flagName
	OutFile           flagName
	QueryOutPath      flagName
	ModelOutPath      flagName
	Schema            flagName
}

var (
	errRequiredRootFields       = errors.New("model-gen: --driver and --url flags are required if config file is not used")
	errInvalidDriver            = errors.New("model-gen: must choose valid --driver.  Options are 'postgres', 'mysql', 'sqlite'")
	errMustSetSchema            = errors.New("model-gen: --schema flag must be set when --driver is set to 'postgres'")
	errRootKeyNotSet            = errors.New("model-gen: root_cmd key in config file must be set")
	errRootKeyDictionary        = errors.New("model-gen: root_cmd key must be dictionary type")
	errRequiredRootFieldsConfig = errors.New("model-gen: fields 'driver' and 'url' must be set under 'root_cmd' key")
	errInvalidDriverConfig      = errors.New("model-gen: must choose valid driver field under 'root_cmd' key.  Options are 'postgres', 'mysql', 'sqlite'")
	errMustSetSchemaConfig      = errors.New("model-gen: 'schema' field must be set when 'driver' field is set to 'postgres'")
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

		return rootCmdPreRunValidation(app.DBDriver(driver), url, schema)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("root_cmd config: %+v\n", viper.Get("root_cmd"))

		return nil
	},
}

func rootCmdPreRunValidation(driver app.DBDriver, url, schema string) error {
	var usingConfig bool
	var err error

	if err = viper.ReadInConfig(); err == nil {
		usingConfig = true
	}

	if !usingConfig {
		if driver == "" || url == "" {
			return errRequiredRootFields
		}

		if driver != app.PostgresDriver && driver != app.MysqlDriver &&
			driver != app.SqliteDriver {
			return errInvalidDriver
		}

		if driver == app.PostgresDriver && schema == "" {
			return errMustSetSchema
		}
	} else {
		rootCmdVal := viper.Get("root_cmd")

		if rootCmdVal == nil {
			return errRootKeyNotSet
		}

		rootCmdMap, ok := rootCmdVal.(map[string]interface{})

		if !ok {
			return errRootKeyDictionary
		}

		mapDriver, driverOK := rootCmdMap["driver"]
		mapURL, urlOK := rootCmdMap["url"]
		mapSchema, schemaOK := rootCmdMap["schema"]

		if (mapDriver == "" || !driverOK) || (mapURL == "" || !urlOK) {
			return errRequiredRootFieldsConfig
		}

		if mapDriver != string(app.PostgresDriver) && mapDriver != string(app.MysqlDriver) &&
			mapDriver != string(app.SqliteDriver) {
			return errInvalidDriverConfig
		}

		if mapDriver == string(app.PostgresDriver) && (mapSchema == "" || !schemaOK) {
			return errMustSetSchemaConfig
		}
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

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.modelgen.yaml)")
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
		"Query code file name",
	)
	rootCmd.PersistentFlags().String(
		generateModelCmdCfg.QueryOutPath.LongHand,
		"",
		"Path the query code will be generated to.  Can be relative path of where program is executed",
	)
	rootCmd.PersistentFlags().String(
		generateModelCmdCfg.ModelOutPath.LongHand,
		"",
		"Path the model code will be generated to.  Can be relative path of where program is executed",
	)
	rootCmd.PersistentFlags().String(
		generateModelCmdCfg.Schema.LongHand,
		"",
		"Schema to base model generation off.  Required if driver is 'postgres'",
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
