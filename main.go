package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-exec/tfinstall"
	"github.com/terraform-providers/terraform-provider-aviatrix/goaviatrix"
)

var boolPtr = flag.Bool("sample", false, "Display sample configuration")

//by default set firstBoot to true
var firstBoot = true

func addEmail(client *goaviatrix.Client, adminEmail string, controllerURL string) error {
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

func changePassword(client *goaviatrix.Client, currentPassword string, newPassword string, controllerURL string) error {
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
	_, err := client.Post(controllerURL, data)
	if err != nil {
		return err
	}
	return nil
}

func initialSetup(client *goaviatrix.Client, controllerURL string) error {
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

func main() {
	var password string
	flag.Parse()
	if *boolPtr {
		fmt.Println(
			`#Run the following commands and replace with your own values
	#export PUBLIC_IP=<controller_public_ip>
	#export PRIVATE_IP=<controller_private_ip>
	#export NEW_PASSWORD=<new_controller_password>
	#export ADMIN_EMAIL=<admin_email_address>

	export PUBLIC_IP=1.2.3.4
	export PRIVATE_IP=192.168.0.10
	export NEW_PASSWORD="Aviatrix123"
	export ADMIN_EMAIL="jane@aviatrix.com"
	export GIT_URL="https://github.com/janedoe/awesomeprojet"
	`)
		os.Exit(0)
	}
	//set variables with env
	controllerIP := os.Getenv("PUBLIC_IP")
	controllerPrivateIP := os.Getenv("PRIVATE_IP")
	newPassword := os.Getenv("NEW_PASSWORD")
	adminEmail := os.Getenv("ADMIN_EMAIL")
	gitURL := os.Getenv("GIT_URL")
	if (controllerIP == "") || (controllerPrivateIP == "") || (newPassword == "") || (adminEmail == "") {
		log.Fatal("Ooops, All env variables must be defined!")
	}

	var controllerURL = "https://" + controllerIP + "/v1/api"

	// Skip Certificate Check
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	//When controller is booting for the first time, the default password
	//is the controller's private IP address
	if firstBoot {
		password = controllerPrivateIP
	}

	//Create new Client object and login to controller
	client, err := goaviatrix.NewClient("admin", password, controllerIP, &http.Client{Transport: tr})
	if err != nil {
		log.Fatal(err)
	}

	//add email
	if err = addEmail(client, adminEmail, controllerURL); err != nil {
		log.Fatal(err)
	}

	//Change account password
	if firstBoot {
		if err = changePassword(client, password, newPassword, controllerURL); err != nil {
			log.Fatal(err)
		}
		//Update to latest software
		data := map[string]string{
			"action":    "initial_setup",
			"CID":       client.CID,
			"subaction": "run",
		}
		_, err = client.Post(controllerURL, data)
		if err != nil {
			log.Fatal(err)
		}
	}

	//Install Terraform
	tmpDir, err := ioutil.TempDir("", "tfinstall")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)
	execPath, err := tfinstall.Find(context.Background(), tfinstall.LatestVersion(tmpDir, false))
	if err != nil {
		panic(err)
	}

	//Pull repo to be used as Terraform source
	_, err = git.PlainClone(tmpDir, false, &git.CloneOptions{
		URL: gitURL,
	})
	if err != nil {
		log.Fatal(err)
	}
	workingDir := tmpDir
	tf, err := tfexec.NewTerraform(workingDir, execPath)
	if err != nil {
		panic(err)
	}
	tf.SetStdout(os.Stdout)

	err = tf.Init(context.Background(), tfexec.Upgrade(true), tfexec.LockTimeout("60s"))
	if err != nil {
		panic(err)
	}

	err = tf.Apply(context.Background())
	if err != nil {
		panic(err)
	}

}
