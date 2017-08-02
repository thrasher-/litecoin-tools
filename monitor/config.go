package main

import (
	"time"
)

const (
	ConfigFile = "config.json"
)

type Config struct {
	HTTPServer     string               `json:"http_server"`
	Slack          ConfigSlack          `json:"slack"`
	DNSSeeders     []ConfigDNSSeeders   `json:"dns_seeders"`
	Websites       []ConfigWebsites     `json:"websites"`
	LitecoinServer ConfigLitecoinServer `json:"litecoin_server"`
	CheckDelay     time.Duration        `json:"check_delay"`
	ReportBlocks   bool                 `json:"report_blocks"`
	APIUrl         string               `json:"api_url"`
}

func LoadConfig() (Config, error) {
	var cfg Config
	file, err := ReadFile(ConfigFile)
	if err != nil {
		return cfg, err
	}

	err = JSONDecode(file, &cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}
