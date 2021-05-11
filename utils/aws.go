package utils

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

//DeployCFT creates the Cloudformation Stack to deploy the controller.
//Returns the Cloudformation Output
func DeployCFT(cftStackInput cloudformation.CreateStackInput, awsRegion string) ([]*cloudformation.Output, error) {
	//Create API session in set AWS region
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
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
