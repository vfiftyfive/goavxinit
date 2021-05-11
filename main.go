package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-exec/tfinstall"
	log "github.com/sirupsen/logrus"
	"github.com/terraform-providers/terraform-provider-aviatrix/goaviatrix"
	"github.com/vfiftyfive/Go-stuff/aviatrix/goavxinit/utils"
)

//Display sample environment configuration with -sample or --sample option
var boolPtr = flag.Bool("sample", false, "Display sample configuration.")

//By default set firstBoot to true
var firstBoot = true

func main() {
	flag.Parse()
	if *boolPtr {
		fmt.Println(
			`#Use this sample to create your own environment configuration.
#E.g.> ./goavxinit -sample > avxenv
#Then run
#> source ./avxenv

###############Variable Description##############################################
#export NEW_PASSWORD=<new_controller_password>
#export ADMIN_EMAIL=<admin_email_address>
#export AVX_LICENSE=<aviatrix customer ID>
#export AWS_REGION=<AWS Region for the Controller>
#export CFT_URL=<URL of the Cloudformation Template to use>
#export AWS_VPC_ID=<VPC where the controller will be deployed>
#export AWS_SUBNET=<Subnet where the controller will be deployed>
#export AWS_KEY_PAIR=<AWS key pair that will be used for the controller instance>
################################################################################

export NEW_PASSWORD="Av!@trix123"
export ADMIN_EMAIL="jane@aviatrix.com"
export GIT_URL="https://github.com/janedoe/awesomeprojet"
export BRANCH_NAME="master"
export AVX_LICENSE="123421234123412378"
export AWS_REGION="us-west-1"
export CFT_URL="http://nvermande.s3.amazonaws.com/Aviatrix/controller/AWS/aviatrix-controller-CFT.json"
export AWS_VPC_ID="vpc-0921eb763899faddc"
export AWS_SUBNET="subnet-0291c878d736c57fb"
export AWS_KEY_PAIR="avx-admin-london"`)
		os.Exit(0)
	}

	//set variables with env
	newPassword := os.Getenv("NEW_PASSWORD")
	adminEmail := os.Getenv("ADMIN_EMAIL")
	gitURL := os.Getenv("GIT_URL")
	strBranchName := os.Getenv("BRANCH_NAME")
	license := os.Getenv("AVX_LICENSE")
	password := os.Getenv("AVX_PASSWORD")
	varFilePath := os.Getenv("TF_VARFILE")
	awsRegion := os.Getenv("AWS_REGION")

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
	log.Info("Deploying Cloudformation template...")
	outputs, err := utils.DeployCFT(cftStackInput, awsRegion)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Done.")
	//Retrieve Controller Information
	type avxOutput struct {
		ControllerEIP       string
		AccountID           string
		ControllerPrivateIP string
		RoleAppARN          string
		RoleEC2ARN          string
	}
	out := avxOutput{}
	for _, element := range outputs {
		switch *element.OutputKey {
		case "AviatrixControllerEIP":
			out.ControllerEIP = *element.OutputValue
		case "AccountId":
			out.AccountID = *element.OutputValue
		case "AviatrixControllerPrivateIP":
			out.ControllerPrivateIP = *element.OutputValue
		case "AviatrixRoleAppARN":
			out.RoleAppARN = *element.OutputValue
		case "AviatrixRoleEC2ARN":
			out.RoleEC2ARN = *element.OutputValue
		}
	}
	log.Infof("Cloudformation Outputs: %+v", out)
	//Wait for Controller to be ready
	// Skip Certificate Check
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	//Force 10s time-out on http client
	httpClient := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}
	//Give-up after 3 tries to reach endpoing
	log.Info("Trying to contact Controller...Will retry if not successful!")
	time.Sleep(80 * time.Second)
	count := 0
	for {
		resp, err := httpClient.Get("https://" + out.ControllerEIP)
		if err != nil {
			log.Warn(err)
		}
		if resp != nil {
			break
		}
		if count == 3 {
			log.Fatal("Maximum retries reached")
			break
		}
		time.Sleep(30 * time.Second)
	}
	//Give-up after 3 tries to receive HTTP 200
	for {
		resp, err := httpClient.Get("https://" + out.ControllerEIP)
		if err != nil {
			log.Warn("Endpoint not Ready, retrying...: %v", err)
		}
		if resp.StatusCode == http.StatusOK {
			log.Info("Received HTTP 200 OK!!!")
			break
		}
		if count == 5 {
			log.Fatal("Maximum retries reached :-(")
			break
		}
		time.Sleep(30 * time.Second)
	}
	//When controller is booting for the first time, the default password
	//is the controller's private IP address
	var controllerURL = "https://" + out.ControllerEIP + "/v1/api"
	if firstBoot {
		password = out.ControllerPrivateIP
	}

	//Create new Client object and login to controller
	client, err := goaviatrix.NewClient("admin", password, out.ControllerEIP, httpClient)
	if err != nil {
		log.Fatal(err)
	}

	//add email
	log.Info("Setting up admin e-mail...")
	if err = utils.AddAdminEmail(client, adminEmail, controllerURL); err != nil {
		log.Fatal(err)
	}

	//Change account password
	if firstBoot {
		log.Info("Changing admin password...")
		if err = utils.ChangeAdminPassword(client, password, newPassword, controllerURL); err != nil {
			log.Fatal(err)
		}
		//Refresh client object with new password
		client, err = goaviatrix.NewClient("admin", newPassword, out.ControllerEIP, &http.Client{Transport: tr})
		if err != nil {
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

	//Configure License / Customer ID
	log.Info("Configuring license...")
	if err = utils.RegisterLicense(client, license, controllerURL); err != nil {
		log.Fatal(err)
	}

	//Install Terraform
	tmpDir, err := ioutil.TempDir("", "tfinstall")
	if err != nil {
		log.Fatal(err)
	}
	// defer os.RemoveAll(tmpDir)
	log.Info("Installing lastest Terraform version...")
	execPath, err := tfinstall.Find(context.Background(), tfinstall.LatestVersion(tmpDir, false))
	if err != nil {
		log.Fatal(err)
	}

	//Pull repo to be used as Terraform source
	log.Infof("Cloning Terraform Configuration from repository: %v", gitURL)
	gitDir := tmpDir + "/clone"
	_, err = git.PlainClone(gitDir, false, &git.CloneOptions{
		URL:           gitURL,
		ReferenceName: plumbing.NewBranchReferenceName(strBranchName),
		SingleBranch:  true,
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
	//and inject controller IP and AWS account id
	log.Info("Running Terraform apply...")
	err = tf.Apply(context.Background(), tfexec.VarFile(varFilePath), tfexec.Var("controller_ip="+out.ControllerEIP), tfexec.Var("aws_account_id="+out.AccountID))
	if err != nil {
		log.Fatal(err)
	}
}
