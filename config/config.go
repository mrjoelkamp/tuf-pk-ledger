package config

import (
	"log"
	"time"

	"github.com/spf13/viper"
)

// Provider defines a set of read-only methods for accessing the application
// configuration params as defined in one of the config files.
type Provider interface {
	ConfigFileUsed() string
	Get(key string) interface{}
	GetBool(key string) bool
	GetDuration(key string) time.Duration
	GetFloat64(key string) float64
	GetInt(key string) int
	GetInt64(key string) int64
	GetSizeInBytes(key string) uint
	GetString(key string) string
	GetStringMap(key string) map[string]interface{}
	GetStringMapString(key string) map[string]string
	GetStringMapStringSlice(key string) map[string][]string
	GetStringSlice(key string) []string
	GetTime(key string) time.Time
	InConfig(key string) bool
	IsSet(key string) bool
}

var defaultConfig *viper.Viper

// Config returns a default config provider
func Config() Provider {
	return defaultConfig
}

// LoadConfigProvider returns a configured viper instance
func LoadConfigProvider(appName string, configFilePath string) Provider {
	return readViperConfig(appName, configFilePath)
}

func init() {
	defaultConfig = readViperConfig("OPKL-UPDATER", "./config.opkl-updater.yaml")
}

func readViperConfig(appName string, configFilePath string) *viper.Viper {
	v := viper.New()
	v.SetEnvPrefix(appName)
	v.AutomaticEnv()

	// global defaults
	v.SetDefault("json_logs", false)
	v.SetDefault("loglevel", "debug")

	// Read configuration from file
	if configFilePath != "" {
		v.SetConfigFile(configFilePath)
		if err := v.ReadInConfig(); err != nil {
			log.Fatalf(err.Error())
		}
	}

	// Merge with environment variables
	v.MergeInConfig()

	return v
}
