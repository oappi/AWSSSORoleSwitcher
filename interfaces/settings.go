package interfaces

import (
	"sync"
)

type SettingsInterface interface {
	GetAlias() string
	GetAccesskey() string
	GetSecretAccessKey() string
	GetSSORegion() (string, error)
	GetAccountRegion() (string, error)
	GetAccounts() ([]string, error)
	GetSSOURL() string
}

type AWSSSOSettings struct {
	Lock              *sync.Mutex
	SSOURL            string
	SSORegion         string
	Region            string
	UserAlias         string
	AWSFolderLocation string
	LocalWriter       LocalWriter
}

func (sso AWSSSOSettings) GetSSOURL() string {

	return sso.SSOURL
}

func (sso AWSSSOSettings) GetAlias() string {

	return sso.UserAlias
}

func (sso AWSSSOSettings) GetAccesskey() string {
	return "sso.credentialFetch"
}

func (sso AWSSSOSettings) GetSecretAccessKey() string {
	return "sso.credentialFetch"
}

func (sso AWSSSOSettings) GetToken() string {
	return "sso.credentialFetch"
}

func (sso AWSSSOSettings) GetSSORegion() (string, error) {

	return sso.Region, nil
}

func (sso AWSSSOSettings) GetAccountRegion() (string, error) {

	return sso.Region, nil
}

func (sso AWSSSOSettings) GetAccounts() ([]string, error) {
	return []string{"1"}, nil

}
