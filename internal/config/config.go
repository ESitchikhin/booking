package config

import (
	"github.com/spf13/viper"
	"log"
)

// Config структура для хранения конфигурации.
type Config struct {
	Server   ServerConfig
	Supabase SupabaseConfig
}

// ServerConfig для настроек сервера.
type ServerConfig struct {
	Port string `mapstructure:"port"`
}

// SupabaseConfig для настроек Supabase.
type SupabaseConfig struct {
	URL    string `mapstructure:"url"`
	APIKey string `mapstructure:"api_key"`
}

// NewConfig загружает конфигурацию из файла.
func NewConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Unable to read config file, %v", err)
		// Попытка загрузить из переменных окружения, если файл не найден
		viper.BindEnv("server.port", "SERVER_PORT")
		viper.BindEnv("supabase.url", "SUPABASE_URL")
		viper.BindEnv("supabase.api_key", "SUPABASE_API_KEY")
	}

	var cfg Config
	err := viper.Unmarshal(&cfg)
	if err != nil {
		return nil, err
	}

	// Установка значений по умолчанию, если они не заданы
	if cfg.Server.Port == "" {
		cfg.Server.Port = "8080"
	}

	return &cfg, nil
}
