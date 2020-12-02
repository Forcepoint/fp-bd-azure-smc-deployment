package lib

import (
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
)

// Download a file and save it
func DownloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if resp == nil {
		return errors.New("got a nil response from downloading " + url)
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}
