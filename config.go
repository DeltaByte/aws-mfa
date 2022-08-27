package main

import (
	"encoding/json"
	"os"

	homedir "github.com/mitchellh/go-homedir"
)

type Config struct {
	PreviousSourceProfile string
	PreviousMFADevice     string
	PreviousRegion        string
	SourceProfiles        map[string]SourceProfile
}

type SourceProfile struct {
	Name               string
	MfaDevice          string
	DestinationProfile string
	TTL                int
}

func (cfg *Config) Save() error {
	configPath, _ := homedir.Expand("~/aws-mfa-util.json")

	// encode JSON
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	// write to file
	return os.WriteFile(configPath, raw, 0666)
}

func LoadConfig() *Config {
	configPath, _ := homedir.Expand("~/aws-mfa-util.json")
	config := &Config{SourceProfiles: map[string]SourceProfile{}}

	// try to read config file
	content, err := os.ReadFile(configPath)
	if err != nil {
		return config
	}

	// parse config file
	_ = json.Unmarshal(content, config)
	return config
}
