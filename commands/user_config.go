package commands

import (
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

func getUserConfig(cmd *cobra.Command) util.UserConfig {
	configValues, err := cmd.Flags().GetStringToString("config")
	if err != nil {
		panic(err.Error())
	}
	fileConfig := util.LoadUserConfigFile()
	return util.NewUserConfig(fileConfig, configValues)
}
