package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"github.cicd.cloud.fpdev.io/BD/bd-azure-smc-deployment/lib"
	"github.cicd.cloud.fpdev.io/BD/fp-smc-golang/src/smc"
	errorWraper "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

var SmcInstance smc.Smc

var deploySmcCmd = &cobra.Command{
	Use:   "deploy-smc",
	Short: "Create external LDAP user in Forcepoint SMC",
	Long:  `allow all required configurations to a Forcepoint SMC instance in order to create an external active directory and external authentication server`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := AzureCLIInstance.Login(); err != nil {
			logrus.Fatal(err)
		}

		SmcInstance = smc.Smc{
			APIVersion:  viper.GetString("SMC.API_VERSION"),
			Hostname:    viper.GetString("SMC.IP_ADDRESS"),
			Port:        viper.GetString("SMC.PORT"),
			AccessKey:   viper.GetString("SMC.KEY"),
			EntryPoints: nil,
			SetCookie:   false,
		}
		err := SmcInstance.Login()
		if err != nil {
			log.Fatal(err.Error())
		}
		err = createAD()
		if err != nil {
			logrus.Fatal(err)
		}

		logrus.Infof("An external active directory server with name '%s' is been created",
			viper.GetString("DOMAIN_NAME"))
		err = createExternalUser()
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.Infof("An external Users Authentication directory server with name '%s' is been created",
			viper.GetString("DOMAIN_NAME"))
		if err := AzureCLIInstance.Logout(); err != nil {
			logrus.Error(err)
		}
		if err := SmcInstance.Logout(); err != nil {
			log.Fatal(err.Error())
		}
	},
}

func init() {
	rootCmd.AddCommand(deploySmcCmd)
	deploySmcCmd.Flags().StringP("azure-admin-password", "u", "", "Azure admin login password")
	if err := viper.BindPFlag("AZURE_ADMIN_LOGIN_PASSWORD", deployCmd.Flags().Lookup("azure-admin-password")); err != nil {
		log.Fatal(err.Error())
	}
}

func createAD() error {
	if viper.GetString("DOMAIN_NAME") == "" {
		return errors.New("DOMAIN_NAME field is empty in the file. Please add your azure domain name to the config file")
	}
	baseOn := getBaseOn(viper.GetString("DOMAIN_NAME"))
	ldapIpAddress, err := GetLDAPExternalIpAddress()
	if err != nil {
		logrus.Fatal(err)
	}
	ldapIpAddress = strings.TrimSpace(ldapIpAddress)
	displayName, err := GetDisplayName(viper.GetString("AZURE_ADMIN_LOGIN_NAME"))
	if err != nil {
		logrus.Fatal(err)
	}
	displayName = strings.TrimSpace(displayName)
	bindUserId := fmt.Sprintf("CN=%s,OU=AADDC Users,%s", displayName, baseOn)
	ad := smc.ActiveDirectoryLDAPS{
		Address:                   ldapIpAddress,
		BaseDn:                    baseOn,
		BindPassword:              viper.GetString("AZURE_ADMIN_LOGIN_PASSWORD"),
		BindUserId:                bindUserId,
		Name:                      viper.GetString("DOMAIN_NAME"),
		Protocol:                  "ldaps",
		Port:                      636,
		Timeout:                   10,
		Retries:                   2,
		AuthPort:                  1812,
		ClientCertBasedUserSearch: "",
		GroupObjectClass: []string{"sggroup", "organizationalUnit", "organization", "groupOfNames",
			"group", "country"},
		PageSize:        1000,
		UserObjectClass: []string{"sguser", "person", "organizationalPerson", "inetOrgPerson"},
	}
	response, err := SmcInstance.CreateActiveDirectoryLdap(&ad)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusCreated {
		r, _ := ioutil.ReadAll(response.Body)
		return errors.New(string(r))
	}
	return nil
}

func GetLDAPExternalIpAddress() (string, error) {
	c := fmt.Sprintf("az resource list --resource-group %s --resource-type Microsoft.AAD/domainServices --name %s --query [].id --output tsv",
		viper.GetString("RESOURCE_GROUP"), viper.GetString("DOMAIN_NAME"))
	id, err := lib.ExecuteCmd(c)
	if err != nil {
		logrus.Fatal(errorWraper.Wrap(err, "Failed in getting the domainServices id"))
	}
	id = strings.TrimSpace(id)
	c = fmt.Sprintf("az resource show --ids '%s' --query properties.ldapsSettings.externalAccessIpAddress --output tsv", id)
	ipAddress, err := lib.ExecuteCmd(c)
	if err != nil {
		logrus.Fatal(errorWraper.Wrap(err, "Failed in getting the LDAP ip address"))
	}
	return ipAddress, nil
}

func getBaseOn(domain string) string {
	parts := strings.Split(domain, ".")
	for i, v := range parts {
		parts[i] = fmt.Sprintf("DC=%s", v)
	}
	return strings.Join(parts, ",")

}

func matchDisplayNameToPrincipalName() error {
	var stdout, stderr bytes.Buffer
	c2 := "az ad user list | grep userPrincipalName | awk -F':' '{print $2}'| tr -d \" \\n\""
	exe := exec.Command("sh", "-c", c2)
	exe.Stderr = &stderr
	exe.Stdout = &stdout
	err := exe.Run()
	if err != nil {
		return err
	}
	errorResult := string(stderr.Bytes())
	if len(errorResult) != 0 {
		return errors.New(errorResult)
	}
	output := string(stdout.Bytes())
	output = strings.ReplaceAll(output, "\"", "")
	parts := strings.Split(output, ",")
	if len(parts) == 0 {
		return errors.New("failed to extract userPrincipalName in order to make them to user DisplayName")
	}
	for _, v := range parts {
		if v != "" {
			stdout.Reset()
			stderr.Reset()
			c := fmt.Sprintf("az ad user update --id %s --display-name '%s'", v, v)
			exe := exec.Command("sh", "-c", c)
			exe.Stderr = &stderr
			exe.Stdout = &stdout
			err := exe.Run()
			if err != nil {
				return errorWraper.Wrapf(err, "failed in updating the display name for ", v)
			}
			errorResult := string(stderr.Bytes())
			if len(errorResult) != 0 {
				return errors.New(errorResult)
			}
		}
	}
	return nil
}

func GetDisplayName(useId string) (string, error) {
	c := fmt.Sprintf("az ad user show --id %s --query displayName --output tsv", useId)
	output, err := lib.ExecuteCmd(c)
	if err != nil {
		logrus.Fatal(err)
	}
	return strings.TrimSpace(output), nil
}

func createExternalUser() error {
	ldapAuthService, err := SmcInstance.FindExternalLdap()
	if err != nil {
		return err
	}

	activeDirectory, err := SmcInstance.FindExternalActiveDirectory(viper.GetString("DOMAIN_NAME"))
	if err != nil {
		return err
	}

	externalLdapUser := smc.ExternalLDAPUser{
		AuthMethod: ldapAuthService["href"],
		IsDefault:  true,
		LdapServer: []string{activeDirectory["href"]},
		Name:       viper.GetString("DOMAIN_NAME"),
		ReadOnly:   false,
		System:     false,
	}
	response, err := SmcInstance.CreateLdapExternalUser(&externalLdapUser)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusCreated {
		r, _ := ioutil.ReadAll(response.Body)
		return errors.New(string(r))
	}
	return nil
}

func changePassword(newPassword string) bool {
	var stdout, stderr bytes.Buffer
	c := fmt.Sprintf("az ad user update")
	fmt.Println(c)
	exe := exec.Command("sh", "-c", "az ad user update")
	exe.Stderr = &stderr
	exe.Stdout = &stdout
	err := exe.Run()
	if err != nil {
		logrus.Fatal("Failed in executing command dor changing password")
	}
	errorResult := string(stderr.Bytes())
	if len(errorResult) != 0 {
		if strings.Contains(errorResult, "specified password does not") {
			logrus.Error("Strong password required. Combine at least three of the following: uppercase letters, lowercase letters, numbers, and symbols.")
		} else {
			logrus.Error(errorResult)
		}
	}
	return false
}
