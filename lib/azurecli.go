package lib

import (
	"bytes"
	"errors"
	"fmt"
	errorWrapper "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type AzureCLI struct {
	IsLogin bool
}

// login to azure
func (a AzureCLI) Login() error {
	if !a.IsLogin {
		var stdout, stderr bytes.Buffer
		if viper.GetString("AZURE_ADMIN_LOGIN_NAME") == "" {
			logrus.Fatal("the field AZURE_ADMIN_LOGIN_NAME in the config file is empty. Please add your azure administrator login name to the config file")
		}
		if viper.GetString("AZURE_ADMIN_LOGIN_PASSWORD") == "" {
			fmt.Printf("Enter the current password for '%s' and press Enter: ",
				viper.GetString("AZURE_ADMIN_LOGIN_NAME"))
			bytePassword, err := terminal.ReadPassword(syscall.Stdin)
			if err != nil {
				return err
			}
			password := string(bytePassword)
			fmt.Println() // do not remove it
			if len(password) == 0 {
				return errors.New("please enter a valid password")
			}
			viper.Set("AZURE_ADMIN_LOGIN_PASSWORD", strings.TrimSpace(password))
		}
		//login to azure
		c1 := fmt.Sprintf("az login -u %s -p '%s'",
			viper.GetString("AZURE_ADMIN_LOGIN_NAME"),
			viper.GetString("AZURE_ADMIN_LOGIN_PASSWORD"))
		exe := exec.Command("sh", "-c", c1)
		exe.Stderr = &stderr
		exe.Stdout = &stdout
		err := exe.Run()
		errorResult := string(stderr.Bytes())
		if len(errorResult) != 0 {
			if strings.Contains(errorResult, "Error validating credentials due to invalid username or password") {
				return errors.New("error in validating credentials due to invalid username or password")
			}
			return errors.New(errorResult)
		}
		if err != nil {
			return errors.New("failed in executing the azure login command")
		}
		a.IsLogin = true
	}
	return nil
}

// azure logout
func (a AzureCLI) Logout() error {
	exe := exec.Command("sh", "-c", "az logout")
	err := exe.Run()
	if err != nil {
		err = errorWrapper.Wrap(err, "Failed in executing the azure logout command")
		return err
	}
	a.IsLogin = false
	return nil
}

// create an app in azure
func (a *AzureCLI) CreateApp(appDisplayName string) error {
	c := fmt.Sprintf("az ad app create --display-name '%s' --available-to-other-tenants true --homepage %s --reply-urls %s",
		appDisplayName, viper.GetString("app.url"), viper.GetString("app.url"))
	_, err := ExecuteCmd(c)
	if err != nil {
		return err
	}
	return nil
}

// execute a bash command
func ExecuteCmd(cmd string) (string, error) {
	var stdout, stderr bytes.Buffer
	exe := exec.Command("sh", "-c", cmd)
	exe.Stderr = &stderr
	exe.Stdout = &stdout
	err := exe.Run()
	errorResult := string(stderr.Bytes())
	if len(errorResult) != 0 && !strings.Contains(errorResult, "deprecated") {
		return "", errors.New(errorResult)
	}
	if err != nil && !strings.Contains(errorResult, "deprecated") {
		return "", errors.New(fmt.Sprintf("failed in executing the azure command: %s", cmd))
	}
	output := string(stdout.Bytes())
	if len(output) != 0 {
		return output, nil
	}
	return "", nil
}

// create Sp for an azure app
func (a *AzureCLI) CreateSP(appDisplayName string) error {
	c := fmt.Sprintf("az ad app list  --display-name '%s' --query [].objectId -o tsv", appDisplayName)
	output, err := ExecuteCmd(c)
	if err != nil {
		return err
	}
	appId := strings.ReplaceAll(output, "\n", "")
	appId = strings.TrimSpace(appId)
	c = fmt.Sprintf("az ad sp create --id %s", appId)
	output, err = ExecuteCmd(c)
	if err != nil {
		return err
	}
	c = fmt.Sprintf("az ad sp list --display-name '%s' --query [].objectId -o tsv", appDisplayName)
	output, err = ExecuteCmd(c)
	if err != nil {
		return err
	}
	appSpId := strings.ReplaceAll(output, "\n", "")
	appSpId = strings.TrimSpace(appSpId)
	c = fmt.Sprintf("az ad sp update --id %s --add tags 'WindowsAzureActiveDirectoryIntegratedApp'", appSpId)
	output, err = ExecuteCmd(c)
	if err != nil {
		return err
	}
	return nil
}

func GenerateAppScimTemplate(template string) error {
	accessToken, err := GetGraphAccessToken()
	accessToken = "Bearer " + accessToken
	if err != nil {
		return errorWrapper.Wrap(err, "failed in getting an access token for Graph API")
	}
	appSpId, err := GetSpId(viper.GetString("APP_NAME"))
	if err != nil {
		return err
	}
	appSpScimId, err := GetSpScimId(viper.GetString("APP_NAME"))
	if err != nil {
		return err
	}
	b, err := ioutil.ReadFile(template)
	if err != nil {
		return err
	}
	text := string(b)
	text = strings.ReplaceAll(text, "APP_SP_ID", appSpId)
	text = strings.ReplaceAll(text, "APP_SP_SCIM_IP", appSpScimId)
	buff := []byte(text)
	if err := CreateProvisioningJob(accessToken, appSpId); err != nil {
		return err
	}
	if err := DeployAppSchema(buff, appSpId, appSpScimId, accessToken); err != nil {
		return err
	}
	nginxSmcUrl := fmt.Sprintf("https://%s/smc/", viper.GetString("NGINX_PUBLIC_IP_ADDRESS"))
	args := []string{"--homepage", nginxSmcUrl, "--reply-urls", nginxSmcUrl}
	if err := UpdateApp(viper.GetString("APP_NAME"), args); err != nil {
		return err
	}

	return nil
}

func AddSpTag(appName string, tag string) error {
	appId, err := GetSpId(appName)
	if err != nil {
		return err
	}
	c := fmt.Sprintf("az ad sp update --id %s --add tags '%s'", appId, tag)
	_, err = ExecuteCmd(c)
	if err != nil {
		return err
	}
	return nil
}

func GetGraphAccessToken() (string, error) {
	c := "az account get-access-token --resource https://graph.microsoft.com --query accessToken -o tsv"
	accessToken, err := ExecuteCmd(c)
	accessToken = strings.TrimSpace(accessToken)
	return accessToken, err
}

func GetSpId(appName string) (string, error) {
	c := fmt.Sprintf("az ad sp list --display-name '%s' --query [0].objectId -o tsv", appName)
	appId, err := ExecuteCmd(c)
	return strings.TrimSpace(appId), err
}

func GetSpScimId(appName string) (string, error) {
	c := fmt.Sprintf("az ad sp list --display-name '%s' --query \"[0].servicePrincipalNames[0]\" -o tsv", appName)
	appId, err := ExecuteCmd(c)
	return strings.TrimSpace(appId), err
}

func HttpRequest(method string, url string, body []byte, accessToken string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", accessToken)
	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func CreateProvisioningJob(accessToken string, appId string) error {
	url := fmt.Sprintf("https://graph.microsoft.com/beta/servicePrincipals/%s/synchronization/jobs", appId)
	body := `
{ 
    "templateId": "scim"
}`
	response, err := HttpRequest("POST", url, []byte(body), accessToken)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusCreated {
		return errors.New(fmt.Sprintf("got unexpected http status code: %s for creating a provision job",
			response.StatusCode))
	}
	return nil
}

func DeployAppSchema(body []byte, appId string, appScimId string, accessToken string) error {
	url := fmt.Sprintf("https://graph.microsoft.com/beta/servicePrincipals/%s/synchronization/jobs/scim.7868462e3eae47bb9d58896e03ce6c43.%s/schema", appId, appScimId)
	response, err := HttpRequest("PUT", url, body, accessToken)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusNoContent {
		return errors.New(fmt.Sprintf("got unexpected http status code: %d for deploying app SCIM schmea",
			response.StatusCode))
	}
	return nil
}

func UpdateApp(appName string, args []string) error {
	output, err := ExecuteCmd(fmt.Sprintf("az ad app list --display-name '%s' --query [].objectId --output tsv",
		appName))
	if err != nil {
		return err
	}
	argsString := strings.Join(args, " ")
	c := fmt.Sprintf("az ad app update --id %s %s", strings.TrimSpace(output), argsString)
	_, err = ExecuteCmd(c)
	if err != nil {
		return err
	}
	return nil
}

func CreateGroup(name string) error {
	mailNickname := strings.ToLower(name)
	mailNickname = strings.ReplaceAll(name, " ", ".")
	c := fmt.Sprintf("az ad group create --display-name '%s' --mail-nickname '%s'", name, mailNickname)
	_, err := ExecuteCmd(c)
	time.Sleep(3 * time.Second)
	if err != nil {
		return err
	}
	return nil
}

func AddMemberToGroup(groupName string, userEmail string) error {
	c := fmt.Sprintf("az ad user list --upn '%s' --query [].objectId -o tsv", userEmail)
	userIp, err := ExecuteCmd(c)
	if err != nil {
		return err
	}
	c = fmt.Sprintf("az ad group member add -g '%s' --member-id %s", groupName, userIp)
	_, err = ExecuteCmd(c)
	if err != nil {
		return err
	}
	return nil
}
