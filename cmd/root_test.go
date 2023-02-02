package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/TravisS25/model-gen/app"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

func TestRootCmdPreRunValidation(t *testing.T) {
	var err error

	if err = rootCmdPreRunValidation("", "", ""); err == nil {
		t.Fatalf("should have error\n")
	}

	if !errors.Is(err, errRequiredRootFields) {
		t.Fatalf("error should contain '%v'; got '%v'\n", errRequiredRootFields, err)
	}

	if err = rootCmdPreRunValidation("invalid", "url", ""); err == nil {
		t.Fatalf("should have error\n")
	}

	if !errors.Is(err, errInvalidDriver) {
		t.Fatalf("error should contain '%v'; got '%v'\n", errInvalidDriver, err)
	}

	if err = rootCmdPreRunValidation(app.PostgresDriver, "url", ""); err == nil {
		t.Fatalf("should have error\n")
	}

	if !errors.Is(err, errMustSetSchema) {
		t.Fatalf("error should contain '%v;' got '%v'\n", errMustSetSchema, err)
	}

	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fileDir := home
	fileName := ".test_model_gen"
	filepath := filepath.Join(fileDir, fileName) + ".yaml"

	file, err := os.Create(filepath)

	if err != nil {
		t.Fatalf(err.Error())
	}

	defer os.Remove(filepath)

	// Search config in home directory with name ".modelgen" (without extension).
	viper.AddConfigPath(home)
	viper.SetConfigName(fileName)

	if err = rootCmdPreRunValidation("", "", ""); err == nil {
		t.Fatalf("should have error\n")
	}

	if !errors.Is(err, errRootKeyNotSet) {
		t.Fatalf("error should contain '%v'; got '%v'\n", errRootKeyNotSet, err)
	}

	var mapBytes []byte

	if mapBytes, err = yaml.Marshal(map[string]interface{}{"root_cmd": "string"}); err != nil {
		t.Fatalf(err.Error())
	}

	if _, err = file.Write(mapBytes); err != nil {
		t.Fatalf(err.Error())
	}

	if err = rootCmdPreRunValidation("", "", ""); err == nil {
		t.Fatalf("should have error\n")
	}

	if !errors.Is(err, errRootKeyDictionary) {
		t.Fatalf("error should contain '%v'; got '%v'\n", errRootKeyDictionary, err)
	}

	if mapBytes, err = yaml.Marshal(map[string]interface{}{
		"root_cmd": map[string]interface{}{
			"foo": "bar",
		},
	}); err != nil {
		t.Fatalf(err.Error())
	}

	if _, err = file.Seek(0, 0); err != nil {
		t.Fatalf(err.Error())
	}

	if _, err = file.Write(mapBytes); err != nil {
		t.Fatalf(err.Error())
	}

	if err = rootCmdPreRunValidation("", "", ""); err == nil {
		t.Fatalf("should have error\n")
	}

	if !errors.Is(err, errRequiredRootFieldsConfig) {
		t.Fatalf("error should contain '%v'; got '%v'\n", errRequiredRootFieldsConfig, err)
	}

	if mapBytes, err = yaml.Marshal(map[string]interface{}{
		"root_cmd": map[string]interface{}{
			"url":    "/url",
			"driver": "invalid",
		},
	}); err != nil {
		t.Fatalf(err.Error())
	}

	if _, err = file.Seek(0, 0); err != nil {
		t.Fatalf(err.Error())
	}

	if _, err = file.Write(mapBytes); err != nil {
		t.Fatalf(err.Error())
	}

	if err = rootCmdPreRunValidation("", "", ""); err == nil {
		t.Fatalf("should have error\n")
	}

	if !errors.Is(err, errInvalidDriverConfig) {
		t.Fatalf("error should contain '%v'; got '%v'\n", errInvalidDriverConfig, err)
	}

	if mapBytes, err = yaml.Marshal(map[string]interface{}{
		"root_cmd": map[string]interface{}{
			"url":    "/url",
			"driver": "postgres",
		},
	}); err != nil {
		t.Fatalf(err.Error())
	}

	if _, err = file.Seek(0, 0); err != nil {
		t.Fatalf(err.Error())
	}

	if _, err = file.Write(mapBytes); err != nil {
		t.Fatalf(err.Error())
	}

	if err = rootCmdPreRunValidation("", "", ""); err == nil {
		t.Fatalf("should have error\n")
	}

	if !errors.Is(err, errMustSetSchemaConfig) {
		t.Fatalf("error should contain '%v'; got '%v'\n", errMustSetSchemaConfig, err)
	}
}
