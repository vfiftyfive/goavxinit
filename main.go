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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/go-git/go-git/v5"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-exec/tfinstall"
	"github.com/terraform-providers/terraform-provider-aviatrix/goaviatrix"
	"github.com/vfiftyfive/Go-stuff/aviatrix/goavxinit/utils"
)

var boolPtr = flag.Bool("sample", false, "Display sample configuration")

//by default set firstBoot to true
var firstBoot = true

func main() {
	flag.Parse()
	if *boolPtr {
		fmt.Println(
			`#Run the following commands and replace with your own values
	#export PUBLIC_IP=<controller_public_ip>
	#export PRIVATE_IP=<controller_private_ip>
	#export NEW_PASSWORD=<new_controller_password>
	#export ADMIN_EMAIL=<admin_email_address>
	#export AVX_LICENSE=<aviatrix customer ID>

	export PUBLIC_IP=1.2.3.4
	export PRIVATE_IP=192.168.0.10
	export NEW_PASSWORD="Aviatrix123"
	export ADMIN_EMAIL="jane@aviatrix.com"
	export GIT_URL="https://github.com/janedoe/awesomeprojet"
	export AVX_LICENSE="123421234123412378"
	export AWS_REGION="us-west-1"
	export CFT_URL="http://nvermande.s3.amazonaws.com/Aviatrix/controller/AWS/aviatrix-controller-CFT.json"
	export AWS_VPC_ID="vpc-0921eb763899faddc"
	export AWS_SUBNET="subnet-0291c878d736c57fb"
	export AWS_KEY_PAIR="avx-admin-london"
	`)
		os.Exit(0)
	}

	//set variables with env
	controllerIP := os.Getenv("PUBLIC_IP")
	controllerPrivateIP := os.Getenv("PRIVATE_IP")
	newPassword := os.Getenv("NEW_PASSWORD")
	adminEmail := os.Getenv("ADMIN_EMAIL")
	gitURL := os.Getenv("GIT_URL")
	license := os.Getenv("AVX_LICENSE")
	password := os.Getenv("AVX_PASSWORD")
	varFilePath := os.Getenv("TF_VARFILE")

	//Create CFT stack input parameters
	cftStackInput := cloudformation.CreateStackInput{
		Parameters: []*cloudformation.Parameter{
			{
				ParameterKey:   aws.String("VPCParam"),
				ParameterValue: aws.String(os.Getenv("AWS_VPC_ID")),
			},
			{
				ParameterKey:   aws.String("SubnetParam"),
				ParameterValue: aws.String(os.Getenv("AWS_SUBNET")),
			},
			{
				ParameterKey:   aws.String("KeyNameParam"),
				ParameterValue: aws.String(os.Getenv("AWS_KEY_PAIR")),
			},
		},
		StackName:    aws.String("aviatrix-controller"),
		TemplateURL:  aws.String(os.Getenv("CFT_URL")),
		Capabilities: []*string{aws.String("CAPABILITY_NAMED_IAM")},
		// OnFailure:    aws.String("DELETE"),
	}

	//Deploy Controller with Cloudformation
	if err := utils.DeployCFT(cftStackInput); err != nil {
		log.Fatal(err)
	}

	//Retrieve Controller Information

	// Skip Certificate Check
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	//When controller is booting for the first time, the default password
	//is the controller's private IP address
	var controllerURL = "https://" + controllerIP + "/v1/api"
	if firstBoot {
		password = controllerPrivateIP
	}

	//Create new Client object and login to controller
	client, err := goaviatrix.NewClient("admin", password, controllerIP, &http.Client{Transport: tr})
	if err != nil {
		log.Fatal(err)
	}

	//add email
	if err = utils.AddAdminEmail(client, adminEmail, controllerURL); err != nil {
		log.Fatal(err)
	}

	//Change account password
	if firstBoot {
		if err = utils.ChangeAdminPassword(client, password, newPassword, controllerURL); err != nil {
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
		//Refresh client object with new password
		client, err = goaviatrix.NewClient("admin", newPassword, controllerIP, &http.Client{Transport: tr})
		if err != nil {
			log.Fatal(err)
		}
	}

	//Configure License / Customer ID
	if err = utils.RegisterLicense(client, license, controllerURL); err != nil {
		log.Fatal(err)
	}

	//Install Terraform
	tmpDir, err := ioutil.TempDir("", "tfinstall")
	if err != nil {
		log.Fatal(err)
	}
	// defer os.RemoveAll(tmpDir)
	execPath, err := tfinstall.Find(context.Background(), tfinstall.LatestVersion(tmpDir, false))
	if err != nil {
		log.Fatal(err)
	}

	//Pull repo to be used as Terraform source
	gitDir := tmpDir + "/clone"
	_, err = git.PlainClone(gitDir, false, &git.CloneOptions{
		URL: gitURL,
	})
	if err != nil {
		log.Fatal(err)
	}

	//Define new Terraform Structure
	workingDir := tmpDir + "/clone"
	tf, err := tfexec.NewTerraform(workingDir, execPath)
	if err != nil {
		log.Fatal(err)
	}
	tf.SetStdout(os.Stdout)

	//Run Terraform init
	err = tf.Init(context.Background(), tfexec.Upgrade(true))
	if err != nil {
		log.Fatal(err)
	}

	//Apply Terraform configuration
	err = tf.Apply(context.Background(), tfexec.VarFile(varFilePath))
	if err != nil {
		log.Fatal(err)
	}

}
