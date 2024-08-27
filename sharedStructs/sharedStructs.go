package sharedStructs

import (
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
)

type STSConfig struct {
	STS stsiface.STSAPI
}

type SessionInfo struct {
	Accesskey       string
	SecretAccessKey string
	Token           string
}

type SSOSessionSettings struct {
	SSOAccountID   string
	SSOAccountRole string
	SsoURL         string
	AccountRegion  string
	SSORegion      string
	Alias          string
}

/*
	type MFAInfo struct {
		profilename   string
		opdomain      string
		opuuid        string
		password      string
		accountnumber string
		switchrole    string
		lock          *sync.Mutex
		awsSession    SessionInfo
		region        string
	}
*/
func CredentialFileSplitter(r rune) bool {
	return r == '[' || r == ']'
}

func ConfigsSplitter(r rune) bool {
	return r == '\n' || r == '\r'
}

/*
type FederationAccountSettingsObject struct {
	Alias           string
	AccessKey       string
	SecretAccessKey string
	token           string
	Region          string
	Accounts        []string
}
*/

type SSOSettingsObject struct {
	Alias          *string
	SSOAccessToken *string
	Region         *string
	SsoClient      *sso.Client
	Accounts       *[]AccountIdNameRole
}

type AccountIdNameRole struct {
	Id   *string
	Name *string
	Role *string
}

type AccountObject struct {
	AccountName     *string
	AccountID       *string
	AccessKey       *string
	SecretAccessKey *string
	Token           *string
}
