package lib

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const (
	CommandFailed       = -1
	FailedDeployment    = -2
	RunningDeployment   = 1
	SucceededDeployment = 0
)

type Parameters struct {
	Schema         string                       `json:"$schema"`
	ContentVersion string                       `json:"contentVersion"`
	Parameters     map[string]map[string]string `json:"parameters"`
}

func (p *Parameters) AddParameter(name string, value string) {
	parameter := make(map[string]string)
	parameter["value"] = value
	p.Parameters[name] = parameter
}

func (p *Parameters) ToJson(filePath string) error {
	file, err := json.MarshalIndent(p, "", " ")
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(filePath, file, 0644); err != nil {
		return err
	}
	return nil
}

func MonitorDeployment() <-chan int {
	time.Sleep(10 * time.Second)
	chnl := make(chan int)
	go func() {
		parts := strings.Split(viper.GetString("DEPLOYMENT_TEMPLATE"), "/")
		deplyName := parts[len(parts)-1]
		parts = strings.Split(deplyName, ".")
		viper.Set("templateName", parts[0])
		c := fmt.Sprintf("az group deployment operation list --resource-group %s --name %s 2> /dev/null | grep provisioningState | awk  '{print $NF}' | tr -d \"\n\"",
			viper.GetString("RESOURCE_GROUP"),
			viper.GetString("templateName"))
		for {

			cmd := exec.Command("sh", "-c", c)
			output, err := cmd.CombinedOutput()
			if err != nil {
				chnl <- CommandFailed
			}
			outputParts := strings.Split(string(output), ",")
			running := false
			failed := false
			for _, v := range outputParts {
				if v == "\"Running\"" {
					running = true
				}
				if v == "\"Failed\"" {
					failed = true
				}
			}
			if failed {
				chnl <- FailedDeployment
				break
			} else if running {
				chnl <- RunningDeployment
			} else {
				chnl <- SucceededDeployment
				break
			}
		}
		close(chnl)
	}()
	return chnl
}

func ReadError() ([]byte, error) {
	c := fmt.Sprintf("az group deployment show --resource-group %s --name %s 2> /dev/null | grep error | grep message",
		viper.GetString("RESOURCE_GROUP"),
		viper.GetString("templateName"))
	cmd := exec.Command("sh", "-c", c)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	return output, nil
}

func NewPassword() (string, error) {
	fmt.Printf("Enter a New Password for '%s': ",
		viper.GetString("AZURE_ADMIN_LOGIN_NAME"))
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		exe := exec.Command("sh", "-c", "az logout")
		_ = exe.Run()
		logrus.Fatal("Failed in reading the password")
	}
	password := string(bytePassword)
	fmt.Println() // do not remove it
	if len(password) == 0 {
		return "", errors.New("Please enter a valid password")
	}
	if len(password) < 8 {
		return "", errors.New("Your password must be longer than 7 characters.")
	}
	fmt.Printf("Enter a New Password for '%s': ",
		viper.GetString("AZURE_ADMIN_LOGIN_NAME"))
	bytePasswordConfirm, err := terminal.ReadPassword(int(syscall.Stdin))
	passwordConfirm := string(bytePasswordConfirm)
	fmt.Println() // do not remove it
	if len(passwordConfirm) == 0 {
		return "", errors.New("Please enter a valid password")
	}
	if password != passwordConfirm {
		return "", errors.New("The passwords you entered do not match")
	}
	return passwordConfirm, nil
}
