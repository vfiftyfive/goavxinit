package utils

import (
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
