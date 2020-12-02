package main

import (
	"fmt"
	"github.cicd.cloud.fpdev.io/BD/bd-azure-smc-deployment/lib"
)

func main() {
	generator := lib.SSLCertGenerator{
		Days:         365,
		Domain:       "corkbizdev.onmicrosoft.com",
		Password:     "Forcepoint1",
		TmpDirectory: "",
	}
	if err := generator.CreateTempFile(); err != nil {
		fmt.Println(err)
	}
	if err := generator.GeneratePrivateKey(); err != nil {
		fmt.Println(err)
	}
	if err := generator.GeneratePublicKey(); err != nil {
		fmt.Println(err)
	}
	if err := generator.GeneratePFX(); err != nil {
		fmt.Println(err)
	}
	if err := generator.ConvertToBase64(); err != nil {
		fmt.Println(err)
	}
	output, err := generator.OutputBase64()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(output))
	generator.CleanUp()

}
