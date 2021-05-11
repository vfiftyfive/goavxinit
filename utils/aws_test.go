package utils

import (
	"crypto/tls"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

func TestAWSCFStack(t *testing.T) {
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
		OnFailure:    aws.String("DELETE"),
	}
	//Deploy Controller with Cloudformation
	stackOutput, err := DeployCFT(cftStackInput)
	if err != nil {
		t.Errorf("Expected no error, but got error %v", err)
	} else {
		t.Logf("Stack Output is %v", stackOutput)
	}
}

func TestAWSCFOutput(t *testing.T) {
	sess, _ := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	// Create Cloudformation service client
	svc := cloudformation.New(sess)
	output, err := svc.DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String("aviatrix-controller")})
	if err != nil {
		t.Errorf("Expected no error, but got error: %v", err)
	}
	type avxOutput struct {
		ControllerEIP       string
		AccountID           string
		ControllerPrivateIP string
		RoleAppARN          string
		RoleEC2ARN          string
	}
	out := avxOutput{}
	for _, element := range output.Stacks[0].Outputs {
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
	t.Logf("Struct info: %#v", out)
}

func TestControllerReady(t *testing.T) {
	//Wait for Controller to be ready
	// Skip Certificate Check

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}
	controllerIP := os.Getenv("PUBLIC_IP")
	t.Logf("Connecting to %v", controllerIP)
	count := 0
	for {
		resp, err := client.Get("https://" + controllerIP)
		if err != nil {
			t.Errorf("Expected no error, but got error: %v", err)
		}
		if resp != nil {
			break
		}
		if count == 4 {
			t.Errorf("Maximum retries reached")
			break
		}
		t.Logf("count: %v", count)
		count += 1
		time.Sleep(30 * time.Second)
	}
}
