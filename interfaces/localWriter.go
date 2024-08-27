package interfaces

import (
	creds "github.com/oappi/awsssoroleswitcher/credentialFileLogic"
	"github.com/oappi/awsssoroleswitcher/sharedStructs"
)

type LocalWriter interface {
	UpdateShortTermKeys(accesskey, secretAccessKey, token string) error
}

type IniLogic struct {
	AWSFolderLocation string
}

func (ini IniLogic) UpdateShortTermKeys(accesskey, secretAccessKey, token string) error {
	creds.UpdateShortTermAWSKeys(accesskey, secretAccessKey, token, ini.AWSFolderLocation)
	return nil
}

func (ini IniLogic) GetSSOSettings() (sharedStructs.SSOSessionSettings, error) {
	return creds.GetAWSSSOSettings(ini.AWSFolderLocation)
}

func (ini IniLogic) SetSSOSettings(SSOSessionSettings sharedStructs.SSOSessionSettings) error {
	return creds.SetAWSSSOSettings(ini.AWSFolderLocation, SSOSessionSettings)
}
func (ini IniLogic) EnrichAccountNameFromAccountOverrides(ssoSettings sharedStructs.SSOSettingsObject) (sharedStructs.SSOSettingsObject, error) {
	return creds.EnrichAccountNameFromAccountOverrides(ini.AWSFolderLocation, ssoSettings)
}

func (ini IniLogic) DumpKeysToCredentialFile(accountInfo []sharedStructs.AccountObject, region *string) error {
	return creds.DumpKeysToCredentialFile(ini.AWSFolderLocation, accountInfo, region)
}
