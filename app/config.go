package app

var (
	DBDrivers = []string{"mysql", "sqlite", "postgres"}
)

type GenerateModelConfig struct {
	Driver       string
	URL          string
	QueryOutPath string
	ModelOutPath string
}
