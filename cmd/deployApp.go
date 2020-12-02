package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// deployAppCmd represents the deployApp command
var deployAppCmd = &cobra.Command{
	Use:   "deploy-app",
	Short: "Create azure App and configure it",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if !AzureCLIInstance.IsLogin {
			if err := AzureCLIInstance.Login(); err != nil {
				logrus.Fatal(err)
			}
		}
		if err := AzureCLIInstance.CreateApp(viper.GetString("APP_NAME")); err != nil {
			logrus.Fatal(err)
		}
		if err := AzureCLIInstance.CreateSP(viper.GetString("APP_NAME")); err != nil {
			logrus.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(deployAppCmd)

}
