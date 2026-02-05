package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dict-be/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Options struct {
	Config string
}

func NewRootCmd() *cobra.Command {
	opts := &Options{}
	defaultConfigPath := ""
	if homeDir, err := os.UserHomeDir(); err == nil {
		defaultConfigPath = filepath.Join(homeDir, ".dict-be.yml")
	}
	root := &cobra.Command{
		Use:   "dict-be",
		Short: "dict-be - CLI starter",
	}

	cobra.OnInitialize(func() {
		initConfig(opts.Config)
	})

	root.PersistentFlags().StringVar(
		&opts.Config,
		"config",
		defaultConfigPath,
		"config file (default: ~/.dict-be.yml)",
	)
	_ = viper.BindPFlag("config", root.PersistentFlags().Lookup("config"))

	root.AddCommand(newQueryCmd())
	root.AddCommand(newVersionCmd())
	return root
}

func initConfig(configFile string) {
	configFile = strings.TrimSpace(configFile)
	if configFile != "" {
		viper.SetConfigFile(expandHome(configFile))
	} else {
		if homeDir, err := os.UserHomeDir(); err == nil {
			viper.SetConfigFile(filepath.Join(homeDir, ".dict-be.yml"))
		}
	}

	viper.SetEnvPrefix("DICT_BE")
	viper.AutomaticEnv()
	viper.SetDefault("llm.url", "")
	viper.SetDefault("llm.model", "")
	viper.SetDefault("llm.token", "")
	viper.SetDefault("llm.type", "")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return
		}
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}

	if _, err := config.Load(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}

func expandHome(path string) string {
	if path == "~" {
		if homeDir, err := os.UserHomeDir(); err == nil {
			return homeDir
		}
		return path
	}
	if strings.HasPrefix(path, "~/") {
		if homeDir, err := os.UserHomeDir(); err == nil {
			return filepath.Join(homeDir, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}
