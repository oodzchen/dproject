// Configs for the app runtime, env file path default to ./.env.local

package config

import (
	"fmt"

	"github.com/caarlos0/env/v9"
	"github.com/joho/godotenv"
)

type AppConfig struct {
	SessionSecret      string `env:"SESSION_SECRET"`
	CSRFSecret         string `env:"CSRF_SECRET"`
	SiteName           string `env:"SITE_NAME"`
	DomainName         string `env:"DOMAIN_NAME" envDefault:"localhost"`
	Port               int    `env:"PORT" envDefault:"3000"`
	ReplyDepthPageSize int    `env:"REPLY_DEPTH_PAGE_SIZE" envDefault:"10"`
	Debug              bool   `env:"DEBUG" envDefault:"false"`
	BrandName          string `env:"BRAND_NAME"`
	DB                 *DBConfig
}

func (ac *AppConfig) GetServerURL() string {
	return fmt.Sprintf("http://%s:%d", ac.DomainName, ac.Port)
}

// Get app host as host:port
func (ac *AppConfig) GetHost() string {
	return fmt.Sprintf("%s:%d", ac.DomainName, ac.Port)
}

type DBConfig struct {
	DBHost              string `env:"DB_HOST"`
	DBName              string `env:"DB_NAME"`
	DBPort              int    `env:"DB_PORT"`
	DBUser              string `env:"DB_USER"`
	AdminPassword       string `env:"ADMIN_PASSWORD"`
	UserDefaultPassword string `env:"USER_DEFAULT_PASSWORD"`
}

func (dbCfg *DBConfig) GetDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		dbCfg.DBUser,
		dbCfg.AdminPassword,
		dbCfg.DBHost,
		dbCfg.DBPort,
		dbCfg.DBName,
	)
}

var Config *AppConfig
var BrandName = "DizKaz"

func Init(envFile string) error {
	cfg, err := Parse(envFile)
	if err != nil {
		return err
	}
	Config = cfg
	return nil
}

// Parse env file and generate AppConfig struct
func Parse(envFile string) (*AppConfig, error) {
	if err := godotenv.Load(envFile); err != nil {
		return nil, err
	}

	dbCfg := &DBConfig{}
	if err := env.Parse(dbCfg); err != nil {
		return nil, err
	}

	cfg := &AppConfig{
		DB:        dbCfg,
		BrandName: BrandName,
	}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
