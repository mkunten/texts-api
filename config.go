package main

import (
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

// config
type config struct {
	Server struct {
		Port string
	}
	Mecab struct {
		Dicts []string
	}
}

func LoadConfig() (*config, error) {
	f, err := os.Open("config.yml")
	if err != nil {
		log.Fatal("loadConfig os.Open err:", err)
		return nil, err
	}
	defer f.Close()

	var cfg config
	err = yaml.NewDecoder(f).Decode(&cfg)
	return &cfg, err
}
