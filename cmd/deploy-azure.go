package cmd

import (
	"fmt"
	"github.cicd.cloud.fpdev.io/BD/bd-azure-smc-deployment/lib"
	"github.com/cheggaaa/pb/v3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"strings"
	"time"
)

const (
	RUNNING_DEPLOYMENT  = 10
	FINISHED_DEPLOYMENT = 10
)

var deployCmd = &cobra.Command{
	Use:   "deploy-azure",
	Short: "Deploy Azure Template",
	Long: `Deploy an azure template to allocate required resources for azure AD DS with LDAPS
and usage of using your command. For example:
`,
	Run: func(cmd *cobra.Command, args []string) {
		if !AzureCLIInstance.IsLogin {
			if err := AzureCLIInstance.Login(); err != nil {
				logrus.Fatal(err)
			}
		}
		// configure azure app provisioning
		if viper.GetBool("CREATE_GROUPS_SMC") {
			if err := lib.GenerateAppScimTemplate(viper.GetString("SCIM_TEMPLATE")); err != nil {
				logrus.Fatal(err)
			}
			logrus.Infof("Your app %s is been configured for provisioning with SCIM", viper.GetString("APP_NAME"))
			if err := lib.AddSpTag(viper.GetString("APP_NAME"), "WindowsAzureActiveDirectoryOnPremApp"); err != nil {
				logrus.Fatal("failed in adding a tag to sp")
			}
		}
		// read all exists groups
		c := "az group list --query [].name -o tsv"
		output, err := lib.ExecuteCmd(c)
		if err != nil {
			logrus.Fatal(errors.Wrap(err, "failed in reading all exists resource groups"))
		}
		groups := strings.Split(output, "\n")
		resourceExists := false
		for _, group := range groups {
			if strings.TrimSpace(group) == viper.GetString("RESOURCE_GROUP") {
				resourceExists = true
				break
			}
		}
		if viper.GetBool("CREATE_GROUPS_SMC") {
			smcGroups := []string{"Operator", "Editor", "Reports Manager", "Superuser", "Owner",
				"NSX Role", "Viewer", "Monitor", "Logs Viewer"}
			for _, name := range smcGroups {
				if err := lib.CreateGroup(name); err != nil {
					logrus.Error(errors.Wrap(err, "failed in creating group: "+name))
				} else {
					logrus.Infof("Created Azure Active Directory Group '%s' for SMC Roles", name)
				}
				time.Sleep(3 * time.Second)
			}
		}

		if !resourceExists {
			//create resource group
			c := fmt.Sprintf("az group create -l %s -n '%s'", viper.GetString("LOCATION"),
				viper.GetString("RESOURCE_GROUP"))
			_, err := lib.ExecuteCmd(c)
			if err != nil {
				logrus.Fatal(errors.Wrap(err, "failed in creating resource group"))
			}
		}
		parameters, err := lib.GenerateParameters()
		if err != nil {
			logrus.Fatal(err.Error())
		}
		c = fmt.Sprintf("az group deployment create --resource-group %s --template-file %s --parameters %s --no-wait",
			viper.GetString("RESOURCE_GROUP"),
			viper.GetString("DEPLOYMENT_TEMPLATE"),
			parameters)
		_, err = lib.ExecuteCmd(c)
		if err != nil {
			if !strings.Contains(err.Error(), "deprecated") {
				logrus.Fatal(err)
			}
		}
		err = os.Remove(parameters)
		logrus.Info("Preparing for deployment...")
		time.Sleep(30 * time.Second)
		logrus.Info("Starting Deployment...")
		time.Sleep(40 * time.Second)
		logrus.Info("Starting Deployment Monitoring...")
		count := (55 * 60) / RUNNING_DEPLOYMENT
		inerCount := 0
		//tmpl := `{{ red "Deploying:" }} {{ bar . "┣" "┃" (cycle . "↖" "↗" "↘" "↙" ) "." "┫"}} {{speed . | rndcolor }} {{percent .}}`
		tmpl := `{{ red "Deploying:" }} {{ bar . "┣" "┃" (cycle . "↖" "↗" "↘" "↙" ) "." "┫"}}  {{percent . | rndcolor }}`

		// start bar based on our template
		bar := pb.ProgressBarTemplate(tmpl).Start(count)
		// set values for string elements
		bar.Set("my_green_string", "green").
			Set("my_blue_string", "blue")
		time.Sleep(20 * time.Second)
		for i := range lib.MonitorDeployment() {
			if i == lib.CommandFailed {
				if err := AzureCLIInstance.Logout(); err != nil {
					logrus.Error(err)
				}
				logrus.Fatal("Failed In monitoring the deployment")
			}
			// deployment failed
			if i == lib.FailedDeployment {
				output, err := lib.ReadError()
				if err != nil {
					if err := AzureCLIInstance.Logout(); err != nil {
						logrus.Error(err)
					}
					logrus.Fatal(err.Error())
				}
				outputStr := strings.ReplaceAll(string(output), "\\r\\n", "")
				parts := strings.Split(outputStr, "\"details\\\":")
				bar.Finish()
				if err := AzureCLIInstance.Logout(); err != nil {
					logrus.Error(err)
				}
				logrus.Fatal(parts[1])
				break
			} else if i == lib.RunningDeployment {
				bar.Increment()
				inerCount++
				time.Sleep(RUNNING_DEPLOYMENT * time.Second)
				continue
			} else if i == lib.SucceededDeployment {
				for i := inerCount; i <= count; i++ {
					bar.Increment()
					inerCount++
					time.Sleep(FINISHED_DEPLOYMENT * time.Millisecond)
				}
				bar.Finish()
				logrus.Println("The Template Deployment process is finished.")
				logrus.Printf("The Deployment for azure AD DS(%s) is started this process can take up to 30 minutes.\n You can use azure portal to monitor this process",
					viper.GetString("DOMAIN_NAME"))
				break
			}
		}
		if err := lib.AddMemberToGroup("AAD DC Administrators",
			viper.GetString("AZURE_ADMIN_LOGIN_NAME")); err != nil {
			logrus.Error(err)
		}
		if err := AzureCLIInstance.Logout(); err != nil {
			logrus.Error(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.Flags().StringP("azure-admin-password", "u", "",
		"Azure admin login password")
	if err := viper.BindPFlag("AZURE_ADMIN_LOGIN_PASSWORD",
		deployCmd.Flags().Lookup("azure-admin-password")); err != nil {
		logrus.Fatal(err.Error())
	}
	deployCmd.Flags().BoolP("create-groups", "g", false, "Create groups for SMC roles")
	if err := viper.BindPFlag("CREATE_GROUPS_SMC", deployCmd.Flags().Lookup("create-groups")); err != nil {
		logrus.Fatal(err.Error())
	}
}
