package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	LLM LLMConfig `mapstructure:"llm"`
}

type LLMConfig struct {
	URL   string `mapstructure:"url"`
	Model string `mapstructure:"model"`
	Token string `mapstructure:"token"`
	Type  string `mapstructure:"type"`
}

func Load() (Config, error) {
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return cfg, err
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.LLM.Type == "" {
		return nil
	}
	switch c.LLM.Type {
	case "openai", "anthropics", "gemini":
		return nil
	default:
		return fmt.Errorf("invalid llm.type: %s", c.LLM.Type)
	}
}
