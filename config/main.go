// Configs for the app runtime, env file path default to ./.env.local

package config

import (
	"fmt"

	"github.com/caarlos0/env/v9"
	"github.com/joho/godotenv"
)

type AppConfig struct {
	SessionSecret string `env:"SESSION_SECRET"`
	CSRFSecret    string `env:"CSRF_SECRET"`
	// DomainName         string `env:"DOMAIN_NAME" envDefault:"localhost"`
	AppPort            int    `env:"APP_PORT" envDefault:"3000"`
	AppOuterPort       int    `env:"APP_OUTER_PORT" envDefault:"3000"`
	Debug              bool   `env:"DEBUG" envDefault:"false"`
	BrandName          string `env:"BRAND_NAME"`
	BrandDomainName    string `env:"BRAND_DOMAIN_NAME"`
	Slogan             string `env:"SLOGAN"`
	DB                 *DBConfig
	ReplyDepthPageSize int
	AdminEmail         string `env:"ADMIN_EMAIL"`
}

func (ac *AppConfig) GetServerURL() string {
	return fmt.Sprintf("http://%s:%d", "localhost", ac.AppPort)
}

// Get app host as host:port
// func (ac *AppConfig) GetHost() string {
// 	return fmt.Sprintf("%s:%d", ac.DomainName, ac.AppOuterPort)
// }

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

// var BrandName = "DizKaz"
var BrandName = "笛卡"
var BrandDomainName = "DizKaz.com"
var Slogan = "知无不言, 言无不尽"
var ReplyDepthPageSize = 10

func Init(envFile string) error {
	cfg, err := Parse(envFile)
	if err != nil {
		return err
	}
	Config = cfg
	return nil
}

func InitFromEnv() error {
	cfg, err := ParseFromEnv()
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

	return ParseFromEnv()
}

func ParseFromEnv() (*AppConfig, error) {
	dbCfg := &DBConfig{}
	if err := env.Parse(dbCfg); err != nil {
		return nil, err
	}

	cfg := &AppConfig{
		DB:                 dbCfg,
		BrandName:          BrandName,
		BrandDomainName:    BrandDomainName,
		Slogan:             Slogan,
		ReplyDepthPageSize: ReplyDepthPageSize,
	}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
