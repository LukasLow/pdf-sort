package services

import (
	"os"

	"gopkg.in/yaml.v3"

	"pdf-sort/src/config"
	"pdf-sort/src/models"
)

func LoadConfig() models.Config {
	var cfg models.Config

	data, err := os.ReadFile(config.YamlPath)
	if err != nil {
		return make(models.Config)
	}

	_ = yaml.Unmarshal(data, &cfg)

	if cfg == nil {
		return make(models.Config)
	}

	return cfg
}

func SaveConfig(cfg models.Config) {
	data, _ := yaml.Marshal(&cfg)
	_ = os.WriteFile(config.YamlPath, data, 0644)
}
