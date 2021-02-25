package userdir

import (
	_ "embed"
	"gopkg.in/yaml.v2"
	"log"
	"os"
)

//go:embed libman_default_config.yaml
var DefaultConfig []byte

type Config struct {
	SpotifyID     string `yaml:"spotify_id"`
	SpotifySecret string `yaml:"spotify_secret"`
	Prompt        string `yaml:"prompt"`
	DBPath        string `yaml:"db_dir_path"`
}

func LibmanConfig() *Config {
	configPath := confFile()
	f, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}
	lConf := &Config{}
	err = yaml.Unmarshal(f, &lConf)
	if err != nil {
		log.Fatal(err)
	}
	if lConf == nil {
		lConf = &Config{}
	}
	if lConf.Prompt == "" {
		lConf.Prompt = "libman> "
	}
	if lConf.DBPath == "" {
		lConf.DBPath = LibmanDBDir() + "/libman.db"
	}
	if lConf.SpotifyID == "" {
		lConf.SpotifyID = os.Getenv("SPOTIFY_ID")
	}
	if lConf.SpotifySecret == "" {
		lConf.SpotifySecret = os.Getenv("SPOTIFY_SECRET")
	}
	return lConf
}

func confFile() string {
	configPath := LibmanConfigDir()
	err := os.MkdirAll(configPath, 0600)
	if err != nil {
		log.Fatal(err)
	}
	if _, e := os.Stat(configPath + "/libman.yaml"); os.IsNotExist(e) {
		err := os.WriteFile(configPath+"/libman.yaml", DefaultConfig, 0600)
		if err != nil {
			log.Fatal(err)
		}
	}
	return configPath + "/libman.yaml"
}
