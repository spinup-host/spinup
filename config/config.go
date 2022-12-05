package config

import (
	"crypto/rsa"
)

const (
	DefaultNetworkName = "spinup_services"
)

type Configuration struct {
	Common struct {
		Architecture string `yaml:"architecture"`
		ProjectDir   string `yaml:"projectDir"`
		Ports        []int  `yaml:"ports"`
		ClientID     string `yaml:"client_id"`
		ClientSecret string `yaml:"client_secret"`
		ApiKey       string `yaml:"api_key"`
		LogDir       string `yaml:"log_dir"`
		LogFile      string `yaml:"log_file"`
		Monitoring   bool   `yaml:"monitoring"`
	} `yaml:"common"`
	VerifyKey  *rsa.PublicKey
	SignKey    *rsa.PrivateKey
	UserID     string
	PromConfig PrometheusConfig `yaml:"prom_config"`
}

type PrometheusConfig struct {
	Port int `yaml:"port"`
}
