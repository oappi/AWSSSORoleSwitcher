// this file includes call logic. Basically supports main.go
// Purpose is to have GUI definitions in main.go file, and actual logic in this file.
// Basically anything that doesnt draw something to GUI should be here
package main

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/oappi/awsssoroleswitcher/interfaces"
	"github.com/oappi/awsssoroleswitcher/sharedStructs"
	"github.com/pkg/browser"
)

func UpdateList() {
	var empty = ""
	filteredList, _ := filteredListForSelect(&empty)
	gOptionSelection.SetOptions(filteredList)
}

func UpdateUISettings(settings sharedStructs.SSOSettingsObject) {
	accountsList = convertaccountObjectListToStringList(settings.Accounts)
	SettingsObject = settings
}

type FederationAccountSettingsObject struct {
	MFA             string
	Alias           string
	AccessKey       string
	SecretAccessKey string
	Region          string
	Accounts        []string
	MFADevice       string
}

func convertaccountObjectListToStringList(accountObjectListPointer *[]sharedStructs.AccountIdNameRole) []string {
	var accountObjectList = *accountObjectListPointer
	accountStringList := []string{}
	for _, account := range accountObjectList {
		accountStringList = append(accountStringList, *account.Name+"|"+*account.Id+"|"+*account.Role)
	}
	return accountStringList
}

func filteredListForSelect(filter *string) (resultList []string, match bool) {
	var exactMatch = false
	if filter == nil || *filter == "" {
		return accountsList, false
	} else {
		for _, option := range accountsList {
			var filterstring = *filter
			if strings.Contains(strings.ToLower(option), strings.ToLower(filterstring)) {
				resultList = append(resultList, option)
				if option == filterstring {
					exactMatch = true
				}
			}
		}
		return resultList, exactMatch
	}
}

func filteredCustomListForSelect(filter *string, list []string) (resultList []string, match bool) {
	var exactMatch = false
	for _, option := range list {
		var filterstring = *filter
		if strings.Contains(strings.ToLower(option), strings.ToLower(filterstring)) {
			resultList = append(resultList, option)
			if option == filterstring {
				exactMatch = true
			}
		}
	}
	return resultList, exactMatch
}

func OverRideSavedIfUserGivesInput(userInput string, savedInput string) string {
	if len(userInput) > 0 {
		return userInput
	} else {
		return savedInput
	}
}

func fetchAccountCredentials(SSOSettings sharedStructs.SSOSettingsObject, accountId, accountRole, accountName *string) (sharedStructs.AccountObject, error) {
	var accountObject sharedStructs.AccountObject //object that holds also ways to assumerole to other account
	credentials, errSSO := SSOSettings.SsoClient.GetRoleCredentials(context.TODO(), &sso.GetRoleCredentialsInput{
		AccessToken: SSOSettings.SSOAccessToken,
		AccountId:   aws.String(*accountId),
		RoleName:    aws.String(*accountRole),
	})
	if errSSO != nil {
		return accountObject, errSSO
	}
	accountObject.AccountName = accountName
	accountObject.AccountID = accountId
	accountObject.AccessKey = credentials.RoleCredentials.AccessKeyId
	accountObject.SecretAccessKey = credentials.RoleCredentials.SecretAccessKey
	accountObject.Token = credentials.RoleCredentials.SessionToken
	return accountObject, errSSO

}

func GetAWSConfig(region string) aws.Config {
	cfg, _ := config.LoadDefaultConfig(context.TODO(), config.WithDefaultRegion(region))
	return cfg
}

func getAccessToken(settings interfaces.SettingsInterface, cfg aws.Config) (*string, error) {
	var SSOSettings *string
	oidcClient := ssooidc.NewFromConfig(cfg)
	register, errR := oidcClient.RegisterClient(context.TODO(), &ssooidc.RegisterClientInput{
		ClientName: aws.String("AWSSSORoleSwitcher"),
		ClientType: aws.String("public"),
	})
	if errR != nil {
		return SSOSettings, errors.New("Issue registering connection")
	}

	deviceAuth, errDA := oidcClient.StartDeviceAuthorization(context.TODO(), &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     register.ClientId,
		ClientSecret: register.ClientSecret,
		StartUrl:     aws.String(settings.GetSSOURL()),
	})
	if errDA != nil {
		return SSOSettings, errors.New("Issue registering connection. Check SSO-URL")
	}
	url := aws.ToString(deviceAuth.VerificationUriComplete)
	errBrowser := browser.OpenURL(url)
	if errBrowser != nil {
		return SSOSettings, errors.New("Failed to open authentication in browser please use url: " + url)
	}

	var token *ssooidc.CreateTokenOutput
	approved := false
	for !approved {
		t, err := oidcClient.CreateToken(context.TODO(), &ssooidc.CreateTokenInput{
			ClientId:     register.ClientId,
			ClientSecret: register.ClientSecret,
			DeviceCode:   deviceAuth.DeviceCode,
			GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
		})
		if err != nil {
			isPending := strings.Contains(err.Error(), "AuthorizationPendingException:")
			if isPending {
				log.Println("Authorization pending...")
				time.Sleep(time.Duration(deviceAuth.Interval) * time.Second)
				continue
			}
		}
		approved = true
		token = t
		time.Sleep(1 * time.Second)
	}

	return token.AccessToken, nil
}

func getAccountInfo(SSOSettings sharedStructs.SSOSettingsObject, accountInfo sharedStructs.AccountIdNameRole) (sharedStructs.AccountObject, error) {
	var accountObject sharedStructs.AccountObject //object that holds also ways to assumerole to other account
	credentials, errSSO := SSOSettings.SsoClient.GetRoleCredentials(context.TODO(), &sso.GetRoleCredentialsInput{
		AccessToken: SSOSettings.SSOAccessToken,
		AccountId:   aws.String(*accountInfo.Id),
		RoleName:    aws.String(*accountInfo.Role),
	})
	if errSSO != nil {
		return accountObject, errSSO
	}

	accountObject.AccountID = accountInfo.Id
	accountObject.AccountName = accountInfo.Name
	accountObject.AccessKey = credentials.RoleCredentials.AccessKeyId
	accountObject.SecretAccessKey = credentials.RoleCredentials.SecretAccessKey
	accountObject.Token = credentials.RoleCredentials.SessionToken
	return accountObject, nil

}

func fetchAccountlist(ssoClient *sso.Client, token *string) ([]sharedStructs.AccountIdNameRole, error) {
	accountPaginator := sso.NewListAccountsPaginator(ssoClient, &sso.ListAccountsInput{
		AccessToken: token,
	})
	var accountList []sharedStructs.AccountIdNameRole

	for accountPaginator.HasMorePages() {
		sSOOutput, errPaginatorError := accountPaginator.NextPage(context.TODO())
		if errPaginatorError != nil {
			return accountList, errPaginatorError
		}

		for _, accountOutput := range sSOOutput.AccountList {
			rolePaginator := sso.NewListAccountRolesPaginator(ssoClient, &sso.ListAccountRolesInput{
				AccessToken: token,
				AccountId:   accountOutput.AccountId,
			})
			for rolePaginator.HasMorePages() {
				roleListOutput, roleListerr := rolePaginator.NextPage(context.TODO())
				if roleListerr != nil {
					return accountList, roleListerr
				}
				for _, roleOutput := range roleListOutput.RoleList {
					account := sharedStructs.AccountIdNameRole{Id: roleOutput.AccountId, Name: accountOutput.AccountName, Role: roleOutput.RoleName}
					accountList = append(accountList, account)
				}
			}
		}
	}
	return accountList, nil
}

func updateSettings(SettingsInterface interfaces.SettingsInterface) error {
	ssoRegion, _ := SettingsInterface.GetSSORegion()
	aWSConfig := GetAWSConfig(ssoRegion)
	ssoClient := sso.NewFromConfig(aWSConfig)
	token, tokenErr := getAccessToken(SettingsInterface, aWSConfig)
	if tokenErr != nil {
		return tokenErr
	}
	accounts, AccountFetchErrors := fetchAccountlist(ssoClient, token)
	if AccountFetchErrors != nil {
		return AccountFetchErrors
	}

	alias := SettingsInterface.GetAlias()

	ssoSettings := sharedStructs.SSOSettingsObject{Alias: &alias, SSOAccessToken: token, Region: &ssoRegion, SsoClient: ssoClient, Accounts: &accounts}
	ssoSettingsUpdated, localWriteError := localWriter.EnrichAccountNameFromAccountOverrides(ssoSettings)
	if localWriteError != nil {
		return localWriteError
	}
	UpdateUISettings(ssoSettingsUpdated)
	UpdateList()
	return nil
}

func SSOFetchAndSaveAccountCredentials(ssoClient *sso.Client, token *string, accountToConnect string, accountRole string) error {
	credentials, err := ssoClient.GetRoleCredentials(context.TODO(), &sso.GetRoleCredentialsInput{
		AccessToken: token,
		AccountId:   aws.String(accountToConnect),
		RoleName:    aws.String(accountRole),
	})
	if err != nil {
		return err
	}
	awsSession.Accesskey = *credentials.RoleCredentials.AccessKeyId
	awsSession.SecretAccessKey = *credentials.RoleCredentials.SecretAccessKey
	awsSession.Token = *credentials.RoleCredentials.SessionToken
	return nil
}

func ssoConnectAccount(ssoClient *sso.Client, token *string, selectedAccountInfo string, writer interfaces.LocalWriter) error {
	if selectedAccountInfo == "Connect to credential service first" {
		return errors.New("Your reading skill points have been reduced by one")
	}
	var splittedaccountinfo = strings.Split(selectedAccountInfo, "|")
	var accountToConnect = splittedaccountinfo[1]
	var accountRole = splittedaccountinfo[2]
	fetchError := SSOFetchAndSaveAccountCredentials(ssoClient, SettingsObject.SSOAccessToken, accountToConnect, accountRole)
	if fetchError != nil {
		return fetchError
	}
	writer.UpdateShortTermKeys(awsSession.Accesskey, awsSession.SecretAccessKey, awsSession.Token)
	return nil
}

/*
func ParseSessiontime(sessionTimeOption string) (int64, error) {
	sessionHoursString := strings.Split(sessionTimeOption, " ")[0]
	return strconv.ParseInt(sessionHoursString, 10, 64) //converts string to int, error if failed
}*/
