package main

import (
	"fmt"
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

func BuildLitecoinServerURL(server ConfigLitecoinServer) string {
	return fmt.Sprintf("http://%s:%s@%s:%d", server.RPCUsername, server.RPCPassword, server.RPCServer, server.RPCPort)
}
