package utils

import (
	"os"
	"testing"

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
	}
	t.Logf("Stack Output is %v", stackOutput)
}

func TestAWSCFOutput(t *testing.T) {
	sess, _ := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	// Create Cloudformation service client
	svc := cloudformation.New(sess)
	output, err := svc.DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String("aviatrix-controller")})
	if err != nil {
		t.Errorf("Expected no error, but go error: %v", err)
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
