package cmd

import (
	"fmt"
	"github.cicd.cloud.fpdev.io/BD/bd-azure-smc-deployment/lib"
	"github.com/fsnotify/fsnotify"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var cfgFile string
var AzureCLIInstance lib.AzureCLI

var rootCmd = &cobra.Command{
	Use:   "bd-azure-smc-deployment",
	Short: "Deployment application",
	Long: `Deployment application is used to deploy Azure template for Azure AD DS with external LDAP 
enabled, Create required elements in Forcepoint SMC and generate BASE64 certificate for Azure AD DS LDAP `,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "the config file)")
	//if err := rootCmd.MarkPersistentFlagRequired("config"); err != nil {
	//	log.Fatal(err.Error())
	//}

}

func initConfig() {
	viper.SetDefault("AZURE_ADMIN_LOGIN_NAME", "")
	viper.SetDefault("APP_NAME", "")
	viper.SetDefault("AZURE_ADMIN_LOGIN_PASSWORD", "")
	viper.SetDefault("RESOURCE_GROUP", "forcepoint-smc-integration")
	viper.SetDefault("LOCATION", "westeurope")
	viper.SetDefault("DOMAIN_SERVICES_VNET_NAME", "domain-services-vnet")
	viper.SetDefault("DOMAIN_SERVICES_VNET_ADDRESS_PREFIX", "10.0.0.0/16")
	viper.SetDefault("DOMAIN_SERVICES_SUBNET_NAME", "domain-services-subnet")
	viper.SetDefault("DOMAIN_SERVICES_SUBNET_ADDRESS_PREFIX", "10.0.0.0/24")
	viper.SetDefault("LOGGER_JSON_FORMAT", false)
	viper.SetDefault("DEPLOYMENT_TEMPLATE", "/app/azure_smc_template.json")
	viper.SetDefault("SCIM_TEMPLATE", "/app/scim_template.json")
	viper.SetDefault("PARAMETERS_PATH", "/tmp")
	viper.SetDefault("CREATE_GROUPS_SMC", false)
	viper.SetDefault("SMC.PORT", "8082")
	viper.SetDefault("SMC.API_VERSION", "6.7")
	viper.SetDefault("app.url", "https://217.182.25.38")

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName("deployment")
	}
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		viper.WatchConfig()
		viper.OnConfigChange(func(e fsnotify.Event) {
			if viper.GetBool("LOGGER_JSON_FORMAT") {
				logrus.SetFormatter(&logrus.JSONFormatter{})
			} else {
				logrus.SetFormatter(&logrus.TextFormatter{})
			}
		})
	}
	if viper.GetBool("LOGGER_JSON_FORMAT") {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}
	logrus.SetOutput(os.Stdout)
	AzureCLIInstance = lib.AzureCLI{}
}
