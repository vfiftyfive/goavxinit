package utils

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	log "github.com/sirupsen/logrus"
)

//DeployCFT creates the Cloudformation Stack to deploy the controller.
//Returns the Cloudformation Output
func DeployCFT(cftStackInput cloudformation.CreateStackInput, awsRegion string, awsProfile string) ([]*cloudformation.Output, error) {
	//Set Credentials Profile
	os.Setenv("AWS_PROFILE", awsProfile)
	//Create API session and set AWS region
	sess, _ := session.NewSession()
	// Create Cloudformation service client
	svc := cloudformation.New(sess, aws.NewConfig().WithRegion(awsRegion))
	log.Infof("AWS Region is: %v", awsRegion)
	//Deploy Cloudformation Stack with expected parameters
	stackOutput, err := svc.CreateStack(&cftStackInput)
	if err != nil {
		return nil, err
	}

	//Wait for Cloudformation to complete
	avxStack := cloudformation.DescribeStacksInput{StackName: stackOutput.StackId}
	if err := svc.WaitUntilStackCreateComplete(&avxStack); err != nil {
		return nil, err
	}
	//Return Cloudformation outputs
	output, err := svc.DescribeStacks(&cloudformation.DescribeStacksInput{StackName: stackOutput.StackId})
	if err != nil {
		return nil, err
	}
	return output.Stacks[0].Outputs, nil
}
