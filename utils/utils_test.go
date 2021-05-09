package utils_test

import (
	"crypto/tls"
	"net/http"
	"os"
	"testing"

	"github.com/terraform-providers/terraform-provider-aviatrix/goaviatrix"
	"github.com/vfiftyfive/Go-stuff/aviatrix/goavxinit/utils"
)

func TestRegisterLicense(t *testing.T) {
	// Skip Certificate Check
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	controllerIP := os.Getenv("PUBLIC_IP")
	var controllerURL = "https://" + controllerIP + "/v1/api"
	license := os.Getenv("AVX_LICENSE")
	password := os.Getenv("AVX_PASSWORD")
	client, err := goaviatrix.NewClient("admin", password, controllerIP, &http.Client{Transport: tr})
	if err != nil {
		t.Errorf(" Expected no error, got error: %v", err)
	}
	if err = utils.RegisterLicense(client, license, controllerURL); err != nil {
		t.Errorf(" Expected no error, got %v", err)
	}
}
