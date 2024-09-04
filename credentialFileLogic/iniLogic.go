package credentialFileLogic

import (
	"errors"
	"os"

	"github.com/oappi/awsssoroleswitcher/sharedStructs"

	"gopkg.in/ini.v1"
)

func GetAWSSSOSettings(AWSFolderlocation string) (sharedStructs.SSOSessionSettings, error) {
	var settingObject sharedStructs.SSOSessionSettings
	cfg, err := ini.Load(AWSFolderlocation + "awsssoroleswitcher")
	if err != nil {
		return settingObject, err
	}
	ssoURL := cfg.Section("localSettings").Key("AWSSSOURL").String()
	SSORegion := cfg.Section("localSettings").Key("SSORegion").String()
	AccountRegion := cfg.Section("localSettings").Key("AccountRegion").String()
	alias := cfg.Section("localSettings").Key("Alias").String()
	settingObject.SsoURL = ssoURL
	settingObject.SSORegion = SSORegion
	settingObject.AccountRegion = AccountRegion
	settingObject.Alias = alias

	return settingObject, nil
}

func DumpKeysToCredentialFile(AWSFolderlocation string, accountInfo []sharedStructs.AccountObject, region *string) error {
	cfg, err := ini.Load(AWSFolderlocation + "credentials")
	if err != nil {
		return err
	}
	for _, account := range accountInfo {
		cfg.DeleteSection(*account.AccountName)
		cfg.NewSection(*account.AccountName)
		cfg.Section(*account.AccountName).NewKey("aws_access_key_id", *account.AccessKey)
		cfg.Section(*account.AccountName).NewKey("aws_secret_access_key", *account.SecretAccessKey)
		cfg.Section(*account.AccountName).NewKey("aws_session_token", *account.Token)
		cfg.Section(*account.AccountName).NewKey("region", *region)

	}
	saveWithReducedPriviliges(AWSFolderlocation+"credentials", cfg)
	return nil
}

func EnrichAccountNameFromAccountOverrides(AWSFolderlocation string, ssoSettings sharedStructs.SSOSettingsObject) (sharedStructs.SSOSettingsObject, error) {

	if _, err := os.Stat(AWSFolderlocation + "awsssoroleswitcherAccountOverrides"); errors.Is(err, os.ErrNotExist) {
		//ignore if override file does not exist
		return ssoSettings, nil
	}
	accountOverrides, err := ini.Load(AWSFolderlocation + "awsssoroleswitcherAccountOverrides")
	if err != nil {
		return ssoSettings, err
	}

	updatedAccounts := []sharedStructs.AccountIdNameRole{}

	for _, account := range *ssoSettings.Accounts {
		accountSection, accountSectionReaderr := accountOverrides.GetSection(*account.Id)
		if accountSectionReaderr == nil {
			accountName, accountNameReadError := accountSection.GetKey("AccountName")
			accountRole, accountRoleRedError := accountSection.GetKey("AccountRole")

			if accountNameReadError == nil && accountRoleRedError == nil {
				var accountnameString = accountName.String()
				var accountRoleString = accountRole.String()
				if *account.Role == accountRoleString {
					//case where we were able to read both values and override definition has same role for given account
					var overriden = true
					updatedAccount := sharedStructs.AccountIdNameRole{Id: account.Id, Role: account.Role, Name: &accountnameString, Overriden: &overriden}
					updatedAccounts = append(updatedAccounts, updatedAccount)
				} else {
					// case with multiple roles on same account
					updatedAccounts = append(updatedAccounts, account)
				}
			} else {
				//case where one of the values failed to be read so we wont override value
				updatedAccounts = append(updatedAccounts, account)
			}
		} else {
			updatedAccounts = append(updatedAccounts, account)
		}
	}
	*ssoSettings.Accounts = updatedAccounts

	return ssoSettings, nil
}

func saveWithReducedPriviliges(fullFilePath string, cfg *ini.File) error {
	err := cfg.SaveTo(fullFilePath)
	if err != nil {
		return err
	}
	cerr := os.Chmod(fullFilePath, 0600)
	if cerr != nil {
		return cerr
	}
	return nil
}

func SetAWSSSOSettings(AWSFolderlocation string, SSOSettings sharedStructs.SSOSessionSettings) error {
	cfg, err := ini.Load(AWSFolderlocation + "awsssoroleswitcher")
	if err != nil {
		cfg = ini.Empty()
	}
	cfg.Section("localSettings").Key("AWSSSOURL").SetValue(SSOSettings.SsoURL)
	cfg.Section("localSettings").Key("SSORegion").SetValue(SSOSettings.SSORegion)
	cfg.Section("localSettings").Key("AccountRegion").SetValue(SSOSettings.AccountRegion)
	cfg.Section("localSettings").Key("Alias").SetValue(SSOSettings.Alias)
	return saveWithReducedPriviliges(AWSFolderlocation+"awsssoroleswitcher", cfg)
}

func UpdateShortTermAWSKeys(accesskey, secretaccesskey, token, AWSFolderlocation string) error {
	cfg, err := ini.Load(AWSFolderlocation + "credentials")
	if err != nil {
		return err
	}
	cfg.Section("default").Key("aws_access_key_id").SetValue(accesskey)
	cfg.Section("default").Key("aws_secret_access_key").SetValue(secretaccesskey)
	cfg.Section("default").Key("aws_session_token").SetValue(token)
	return saveWithReducedPriviliges(AWSFolderlocation+"credentials", cfg)
}

func GetAccountList(AWSFolderlocation string) ([]string, error) {
	accountList := []string{}
	accountsList, err := ini.Load(AWSFolderlocation) //error on not able to read string
	if err != nil {
		return accountList, err
	}
	for _, s := range accountsList.Sections() {
		var credentialElementName = s.Name()
		accountId := s.Key("aws_account_id").String()
		roleName := s.Key("role_name").String()
		if accountId != "" && roleName != "" {
			accountList = append(accountList, credentialElementName+"|"+accountId+"|"+roleName)
		}
	}
	return accountList, nil
}
