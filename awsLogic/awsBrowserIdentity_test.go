package awsLogic

import (
	"testing"
)

func TestSignInURL(t *testing.T) {
	var testcredentialfile = GetSignInURL("eu-west-1", "token")
	if testcredentialfile != "https://us-east-1.signin.aws.amazon.com/oauth?Action=logout&redirect_uri=https%3A%2F%2Feu-west-1.signin.aws.amazon.com%2Ffederation%3FAction%3Dlogin%26Destination%3Dhttps%253A%252F%252Feu-west-1.console.aws.amazon.com%26SigninToken%3Dtoken" {
		t.Fail()
	}
}
