package utils

import (
	"crypto/tls"
	"errors"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/terraform-providers/terraform-provider-aviatrix/goaviatrix"
)

//Add email to Controller admin profile
func AddAdminEmail(client *goaviatrix.Client, adminEmail string, controllerURL string) error {
	data := map[string]string{
		"action":      "add_admin_email_addr",
		"CID":         client.CID,
		"admin_email": adminEmail,
	}
	_, err := client.Post(controllerURL, data)
	if err != nil {
		return err
	}
	return nil
}

//Change Admin password
func ChangeAdminPassword(client *goaviatrix.Client, currentPassword string, newPassword string, controllerURL string) error {
	data := map[string]string{
		"action":       "edit_account_user",
		"CID":          client.CID,
		"account_name": "admin",
		"username":     "admin",
		"password":     currentPassword,
		"what":         "password",
		"old_password": currentPassword,
		"new_password": newPassword,
	}
	if _, err := client.Post(controllerURL, data); err != nil {
		return err
	}
	return nil
}

//Initial Controller Setup with upgrade
func InitialSetup(client *goaviatrix.Client, controllerURL string) error {
	data := map[string]string{
		"action":    "initial_setup",
		"CID":       client.CID,
		"subaction": "run",
	}
	if _, err := client.Post(controllerURL, data); err != nil {
		return err
	}
	return nil
}

//Configure Aviatrix License
func RegisterLicense(client *goaviatrix.Client, license string, controllerURL string) error {
	data := map[string]string{
		"action":      "setup_customer_id",
		"CID":         client.CID,
		"customer_id": license,
	}
	if _, err := client.Post(controllerURL, data); err != nil {
		return err
	}
	return nil
}

//Wait for Controller to be ready
func WaitForController(client *http.Client, controllerIP string) error {
	//Force 10s time-out on http client
	client.Timeout = 10 * time.Second
	//Skip TLS verification
	client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	//Give-up after 3 tries to reach endpoint
	log.Info("Trying to contact Controller...Will retry if not successful!")
	time.Sleep(80 * time.Second)
	count := 0
	for {
		resp, err := client.Get("https://" + controllerIP)
		if resp != nil {
			break
		}
		if count == 3 {
			return err
		}
		time.Sleep(30 * time.Second)
		count += 1
	}
	//Give-up after 5 tries to receive HTTP 200
	for {
		resp, err := client.Get("https://" + controllerIP)
		if err != nil {
			log.Warnf("Endpoint not Ready, retrying...: %v", err)
		}
		if resp.StatusCode == http.StatusOK {
			log.Info("Received HTTP 200 OK!!!")
			break
		}
		if count == 5 {
			return (errors.New("Maximum retries reached :-("))
		}
		time.Sleep(30 * time.Second)
	}
	return nil
}
