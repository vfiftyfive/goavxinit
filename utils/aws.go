package utils

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

//DeployCFT creates the Cloudformation Stack to deploy the controller.
//Returns the Cloudformation Output
func DeployCFT(cftStackInput cloudformation.CreateStackInput) ([]*cloudformation.Output, error) {
	//Use shared configuration file (~/.aws/config). You have to create this file.
	//Config file example:
	// [default]
	// region = eu-west-2
	// [profile iam.n.vermande]
	// region = eu-west-2
	// [profile nvermande-dev]
	// region = us-west-2
	sess, _ := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	// Create Cloudformation service client
	svc := cloudformation.New(sess)
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
