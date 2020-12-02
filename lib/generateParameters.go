package lib

import (
	"github.com/spf13/viper"
	"io/ioutil"
	"strings"
)

func GenerateParameters() (string, error) {
	fileName := ""
	tempFile, err := ioutil.TempFile(viper.GetString("PARAMETERS_PATH"), "parameters_*.json")
	if err != nil {
		return fileName, err
	} else {
		fileName = tempFile.Name()
	}
	p := Parameters{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentParameters.json#",
		ContentVersion: "1.0.0.0",
		Parameters:     make(map[string]map[string]string),
	}
	p.AddParameter("domainName", strings.TrimSpace(viper.GetString("DOMAIN_NAME")))
	p.AddParameter("location", strings.TrimSpace(viper.GetString("LOCATION")))
	p.AddParameter("domainServicesVnetName", strings.TrimSpace(viper.GetString("DOMAIN_SERVICES_VNET_NAME")))
	p.AddParameter("domainServicesVnetAddressPrefix", strings.TrimSpace(viper.GetString("DOMAIN_SERVICES_VNET_ADDRESS_PREFIX")))
	p.AddParameter("domainServicesSubnetName", strings.TrimSpace(viper.GetString("DOMAIN_SERVICES_SUBNET_NAME")))
	p.AddParameter("domainServicesSubnetAddressPrefix", strings.TrimSpace(viper.GetString("DOMAIN_SERVICES_SUBNET_ADDRESS_PREFIX")))
	p.AddParameter("smcIpAddress", strings.TrimSpace(viper.GetString("NGINX_PUBLIC_IP_ADDRESS")))
	p.AddParameter("pfxBase64", strings.TrimSpace(viper.GetString("PFX_CERTIFICATE_BASE64")))
	p.AddParameter("pfxPassword", strings.TrimSpace(viper.GetString("PFX_CERTIFICATE_PASSWORD")))
	if err := p.ToJson(fileName); err != nil {
		return fileName, err
	}
	return fileName, nil
}
