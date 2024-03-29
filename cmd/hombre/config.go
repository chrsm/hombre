package main

import (
	"encoding/json"
	"os"
)

type Config struct {
	Token string `json:"token"`

	Lua struct {
		Path string `json:"path"` // code path

		// Services are long-running lua scripts and they receive a message channel,
		// but only receive messages that contain one of their Commands
		Services []LuaScript `json:"services"`

		// Scripts are one-off lua scripts and they receive a bare message,
		// but only receive a message if it is one of their Commands
		Scripts []LuaScript `json:"scripts"`
	} `json:"lua"`
}

type LuaScript struct {
	Name     string   `json:"name"`     // (name).lua
	Commands []string `json:"commands"` // eg spin, roulette..
}

func loadConfig(fileName string) (*Config, error) {
	f, err := os.Open(fileName)
	defer f.Close()
	if err != nil {
		return nil, err
	}

	c := &Config{}
	if err := json.NewDecoder(f).Decode(c); err != nil {
		return nil, err
	}

	return c, nil
}
