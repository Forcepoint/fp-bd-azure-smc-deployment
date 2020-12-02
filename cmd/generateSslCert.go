package cmd

import (
	"fmt"
	"github.cicd.cloud.fpdev.io/BD/bd-azure-smc-deployment/lib"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var generateSslCertCmd = &cobra.Command{
	Use:   "generate-ssl-cert",
	Short: "Generate PFX Base64 certificate",
	Long:  `This certificate will be used for Azure AD DS LDAP`,
	Run: func(cmd *cobra.Command, args []string) {
		generator := lib.SSLCertGenerator{
			Days:         viper.GetInt("PFX_CERTIFICATE_EXPIRY_DAYS"),
			Domain:       viper.GetString("DOMAIN_NAME"),
			Password:     viper.GetString("PFX_CERTIFICATE_PASSWORD"),
			TmpDirectory: "",
		}
		if err := generator.CreateTempFile(); err != nil {
			logrus.Error(errors.Wrap(err, "failed in creating temp file"))
		}
		if err := generator.GeneratePrivateKey(); err != nil {
			logrus.Error(errors.Wrap(err, "failed in creating private key"))
		}
		if err := generator.GeneratePublicKey(); err != nil {
			logrus.Error(errors.Wrap(err, "failed in creating public key"))
		}
		if err := generator.GeneratePFX(); err != nil {
			logrus.Error(errors.Wrap(err, "failed in generating PFX"))
		}
		if err := generator.ConvertToBase64(); err != nil {
			logrus.Error(errors.Wrap(err, "failed in converting to base64"))
		}
		output, err := generator.OutputBase64()
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(string(output))
		generator.CleanUp()
	},
}

func init() {
	rootCmd.AddCommand(generateSslCertCmd)
}
