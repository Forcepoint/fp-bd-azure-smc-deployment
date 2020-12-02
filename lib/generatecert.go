package lib

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

type SSLCertGenerator struct {
	Days         int
	Domain       string
	Password     string
	TmpDirectory string
	PrivateKey   string
	PublicKey    string
	PFX          string
	Base64       string
}

// create a temporary file
func (s *SSLCertGenerator) CreateTempFile() error {
	name, err := ioutil.TempDir("/tmp", "azure_smc_")
	if err != nil {
		return err
	}
	s.TmpDirectory = name
	return nil
}

// remove all temporary files
func (s *SSLCertGenerator) CleanUp() error {
	if err := os.RemoveAll(s.TmpDirectory); err != nil {
		return err
	}
	return nil
}

// generate private key using openssl
func (s *SSLCertGenerator) GeneratePrivateKey() error {
	tempFile, err := ioutil.TempFile(s.TmpDirectory, "private_*.pem")
	if err != nil {
		return err
	}
	s.PrivateKey = tempFile.Name()
	cmd := exec.Command("sh", "-c", fmt.Sprintf("openssl genrsa 4096 > %s", s.PrivateKey))
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

//generate a public key using openssl
func (s *SSLCertGenerator) GeneratePublicKey() error {

	tempFile, err := ioutil.TempFile(s.TmpDirectory, "public_*.pem")
	if err != nil {
		return err
	}
	s.PublicKey = tempFile.Name()
	c := fmt.Sprintf("openssl req -x509 -days %d -new -key %s -out %s -addext extendedKeyUsage=serverAuth,clientAuth -subj \"/CN=*.%s\"",
		s.Days, s.PrivateKey, s.PublicKey, s.Domain)
	cmd := exec.Command("sh", "-c", c)
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil

}

// generate PFX certificate
func (s *SSLCertGenerator) GeneratePFX() error {
	tempFile, err := ioutil.TempFile(s.TmpDirectory, "cert_*.pfx")
	if err != nil {
		return err
	}
	s.PFX = tempFile.Name()
	c := fmt.Sprintf("openssl pkcs12 -export -in %s -inkey %s -out %s -password pass:%s",
		s.PublicKey, s.PrivateKey, s.PFX, s.Password)
	cmd := exec.Command("sh", "-c", c)
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

// convert PFX certificate to BASE64 sting
func (s *SSLCertGenerator) ConvertToBase64() error {
	tempFile, err := ioutil.TempFile(s.TmpDirectory, "base64_*.txt")
	if err != nil {
		return err
	}
	s.Base64 = tempFile.Name()
	c := fmt.Sprintf("openssl base64 -in %s -out %s", s.PFX, s.Base64)
	cmd := exec.Command("sh", "-c", c)
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func (s *SSLCertGenerator) OutputBase64() ([]byte, error) {
	c := fmt.Sprintf("cat %s | tr -d \"\n\"", s.Base64)
	cmd := exec.Command("sh", "-c", c)
	output, err := cmd.CombinedOutput()
	return output, err
}
