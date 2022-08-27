package main

import (
	"log"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	ini "gopkg.in/ini.v1"
)

type Credentials struct {
	Names    []string
	Profiles map[string]Profile
}

type Profile struct {
	isNew           bool   `ini:"-"`
	isChanged       bool   `ini:"-"`
	AccessKeyId     string `ini:"aws_access_key_id"`
	SecretAccessKey string `ini:"aws_secret_access_key"`
	SessionToken    string `ini:"aws_session_token"`
}

func (creds *Credentials) New(name string) {
	if contains(creds.Names, name) {
		log.Fatalf("Profile already exists: %s\n", name)
	}
	creds.Names = append(creds.Names, name)
	creds.Profiles[name] = Profile{isNew: true}
}

func (creds *Credentials) Save() error {
	credsPath, _ := homedir.Expand("~/.aws/credentials")

	// try to read config file
	content, err := os.ReadFile(credsPath)
	if err != nil {
		log.Fatalln("Failed to open ~/.aws/credentials")
	}

	// parse creds file
	parsed, err := ini.Load(content)
	if err != nil {
		log.Fatalln("Failed to parse ~/.aws/credentials")
	}

	// overwrite changed sections
	for profileName, profile := range creds.Profiles {
		if !profile.isChanged {
			continue
		}

		// recreate section
		parsed.DeleteSection(profileName)
		section, err := parsed.NewSection(profileName)
		if err != nil {
			return err
		}

		// marshal profile to section
		if err := section.ReflectFrom(&profile); err != nil {
			return err
		}
	}

	return parsed.SaveTo(credsPath)
}

func LoadCredentials() *Credentials {
	credsPath, _ := homedir.Expand("~/.aws/credentials")
	creds := Credentials{Profiles: map[string]Profile{}}

	// try to read config file
	content, err := os.ReadFile(credsPath)
	if err != nil {
		log.Fatalln("Failed to open ~/.aws/credentials")
	}

	// parse creds file
	parsed, err := ini.Load(content)
	if err != nil {
		log.Fatalln("Failed to parse ~/.aws/credentials")
	}

	// map ini to credentials
	for _, section := range parsed.Sections() {
		if section.Name() == "DEFAULT" {
			continue
		}

		profile := &Profile{isNew: false, isChanged: false}
		if err := section.MapTo(profile); err != nil {
			panic(err)
		}

		creds.Names = append(creds.Names, section.Name())
		creds.Profiles[section.Name()] = *profile
	}

	return &creds
}
