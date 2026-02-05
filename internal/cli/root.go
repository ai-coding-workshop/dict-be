package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Options struct {
	Config string
}

func NewRootCmd() *cobra.Command {
	opts := &Options{}
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
		"",
		"config file (default: ./dict-be.yaml)",
	)
	_ = viper.BindPFlag("config", root.PersistentFlags().Lookup("config"))

	root.AddCommand(newVersionCmd())
	return root
}

func initConfig(configFile string) {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("dict-be")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.config/dict-be")
	}

	viper.SetEnvPrefix("DICT_BE")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return
		}
		fmt.Fprintln(os.Stderr, err.Error())
	}
}
