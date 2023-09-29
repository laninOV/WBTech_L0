package configuration

import (
	"os"
)

func ConfigSetup() {
	os.Setenv("user", "postgres")
	os.Setenv("password", "qwe")
	os.Setenv("dbname", "WBTechDatabase")
	os.Setenv("sslmode", "disable")
	os.Setenv("CACHE_SIZE", "10")
	os.Setenv("APP_KEY", "WB-1")
}
