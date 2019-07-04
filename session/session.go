package session

import (
	"github.com/aws/aws-sdk-go/aws/session"
)

// Get an AWS session. This may turn out to be unncessary, but there are a lot of things which "might"
// need to happen for creating an aws session
// I could just pass in an aws session.Options....
func NewAwsSession(config map[string]interface{}) (*session.Session, error) {

	// use ~/.aws/config AND ~/.aws/credentials, I believe.
	// need to experiment with how this interracts with environment vars, passing in command line flags, etc.
	// read from config for more options
	options := session.Options{
		SharedConfigState: session.SharedConfigEnable,
		// Profile: "foo"
		// Config: aws.Config{Region: aws.String("us-east-1")},
	}
	return session.NewSessionWithOptions(options)

	// I think this just uses shared credentials but nothing else in ~/.aws?
	// sess, err := session.NewSession(&aws.Config{
	//   Region:      aws.String("us-east-1"),
	//   Credentials: credentials.NewSharedCredentials("", "test-account"),
	// })

	// sess, err := session.NewSession(&aws.Config{
	// 	Region: aws.String("us-east-1")},
	// )

}
