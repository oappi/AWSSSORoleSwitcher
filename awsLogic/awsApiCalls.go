package awsLogic

import (

	/*
		"github.com/aws/aws-sdk-go/aws"
		"github.com/aws/aws-sdk-go/aws/credentials"
		"github.com/aws/aws-sdk-go/aws/session"*/

	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/oappi/awsssoroleswitcher/sharedStructs"
)

/*
func CreateSTSSession(settingsFile sharedStructs.SSOSettingsObject) (*sharedStructs.STSConfig, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(*settingsFile.Region),
		Credentials: credentials.NewStaticCredentials(settingsFile.AccessKey, settingsFile.SecretAccessKey, ""),
	})
	if err != nil {
		return &sharedStructs.STSConfig{}, fmt.Errorf("unable to create a session to aws with error: %v", err)
	}
	return &sharedStructs.STSConfig{
		STS: sts.New(sess),
	}, nil
}*/

func GetAsumeRoleCredentials(stsConfig *sharedStructs.STSConfig, settingsFile sharedStructs.SSOSettingsObject, accountnumber string, switchrole string, sessionTime int64) (string, string, string, error) {
	roleToAssumeArn := "arn:aws:iam::" + accountnumber + ":role/" + switchrole
	var duration int64 = 3600 * sessionTime
	result, err := stsConfig.STS.AssumeRole(&sts.AssumeRoleInput{
		RoleArn:         &roleToAssumeArn,
		RoleSessionName: settingsFile.Alias,
		DurationSeconds: &duration,
	})
	if err != nil {
		return "", "", "", err
	}
	return *result.Credentials.AccessKeyId, *result.Credentials.SecretAccessKey, *result.Credentials.SessionToken, nil
}
