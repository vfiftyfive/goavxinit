package utils

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

func DeployCFT(cftStackInput cloudformation.CreateStackInput) error {
	//Use shared configuration file (~/.aws/config)
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
	if _, err := svc.CreateStack(&cftStackInput); err != nil {
		return err
	}
	return nil
}
