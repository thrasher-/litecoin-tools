package main

import (
	"encoding/json"
	"io/ioutil"
	"time"
)

const (
	ConfigFile = "config.json"
)

type Config struct {
	HTTPServer               string               `json:"http_server"`
	Slack                    ConfigSlack          `json:"slack"`
	DNSSeeders               []ConfigDNSSeeders   `json:"dns_seeders"`
	Websites                 []ConfigWebsites     `json:"websites"`
	LitecoinServer           ConfigLitecoinServer `json:"litecoin_server"`
	CheckDelay               time.Duration        `json:"check_delay"`
	ErrorTransitionThreshold int                  `json:"error_transition_threshold"`
	KnownErrorEndpoints      string               `json:"known_error_endpoints"`
	ReportBlocks             bool                 `json:"report_blocks"`
	APIUrl                   string               `json:"api_url"`
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

func SaveConfig(cfg Config) error {
	payloadJSON, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(ConfigFile, payloadJSON, 0644)
}
