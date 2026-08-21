// this file includes call logic. Basically supports main.go
// Purpose is to have GUI definitions in main.go file, and actual logic in this file.
// Basically anything that doesnt draw something to GUI should be here
package main

import (
	"context"
	"errors"
	"log"
	"runtime"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	ssoTypes "github.com/aws/aws-sdk-go-v2/service/sso/types"
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

/*
fetchAccountCredentials is meant to return unique accountobject which means if name has not been overriden it will be in form of <AWS_provided_name>_<Role_of_the_credentials>
*/
func fetchUniqueAccountCredentials(SSOSettings sharedStructs.SSOSettingsObject, accountId, accountRole, accountName *string, accountNameOverriden *bool) (sharedStructs.AccountObject, error) {
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
	if *accountNameOverriden == false {
		var uniqueAccountName = *accountName + "_" + *accountRole
		accountObject.AccountName = &uniqueAccountName
	}
	accountObject.AccountID = accountId
	accountObject.AccessKey = credentials.RoleCredentials.AccessKeyId
	accountObject.SecretAccessKey = credentials.RoleCredentials.SecretAccessKey
	accountObject.Token = credentials.RoleCredentials.SessionToken
	return accountObject, errSSO

}

func GetAWSConfig(region string) aws.Config {
	cfg, _ := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	return cfg
}

func getAccessToken(settings interfaces.SettingsInterface, cfg aws.Config, UIproofcodeTextLabel *widget.Label) (*string, error) {
	var SSOSettings *string
	oidcClient := ssooidc.NewFromConfig(cfg)
	register, errR := oidcClient.RegisterClient(context.TODO(), &ssooidc.RegisterClientInput{
		ClientName: aws.String("AWSSSORoleSwitcher"),
		ClientType: aws.String("public"),
		Scopes:     []string{"sso:account:access"},
	})
	if errR != nil {
		log.Printf("RegisterClient error: %v", errR)
		return SSOSettings, errors.New("Issue registering connection: " + errR.Error())
	}

	ssourl := aws.String(settings.GetSSOURL())

	deviceAuth, errDA := oidcClient.StartDeviceAuthorization(context.TODO(), &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     register.ClientId,
		ClientSecret: register.ClientSecret,
		StartUrl:     ssourl,
	})

	if errDA != nil {
		log.Printf("StartDeviceAuthorization error: %v", errDA)
		return SSOSettings, errors.New("Issue registering connection. Check SSO-URL: " + errDA.Error())
	}

	log.Printf("Device auth started: UserCode=%s", *deviceAuth.UserCode)

	// Update UI from background thread
	fyne.Do(func() {
		UIproofcodeTextLabel.SetText(*deviceAuth.UserCode)
	})
	url := aws.ToString(deviceAuth.VerificationUriComplete)
	errBrowser := browser.OpenURL(url)
	if errBrowser != nil {
		return SSOSettings, errors.New("Failed to open authentication in browser please use url: " + url)
	}

	var token *ssooidc.CreateTokenOutput
	for {
		t, err := oidcClient.CreateToken(context.TODO(), &ssooidc.CreateTokenInput{
			ClientId:     register.ClientId,
			ClientSecret: register.ClientSecret,
			DeviceCode:   deviceAuth.DeviceCode,
			GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
		})
		if err != nil {
			errStr := err.Error()

			// we are waiting for user to approve in browser
			if strings.Contains(errStr, "AuthorizationPendingException") {
				log.Println("Authorization pending...")
				time.Sleep(time.Duration(deviceAuth.Interval) * time.Second)
				continue
			}
			if strings.Contains(errStr, "SlowDownException") {
				log.Println("Slow down...")
				time.Sleep(time.Duration(deviceAuth.Interval+2) * time.Second)
				continue
			}
			//some other error we should break loop
			return nil, err
		}

		if t == nil || t.AccessToken == nil || *t.AccessToken == "" {
			return nil, errors.New("received empty token from AWS")

		} else {
			token = t
		}

		return token.AccessToken, nil
	}
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

func fetchRolesForAccount(ssoClient *sso.Client, token *string, accountOutput ssoTypes.AccountInfo) ([]sharedStructs.AccountIdNameRole, error) {
	rolePaginator := sso.NewListAccountRolesPaginator(ssoClient, &sso.ListAccountRolesInput{
		AccessToken: token,
		AccountId:   accountOutput.AccountId,
	})

	var roles []sharedStructs.AccountIdNameRole
	for rolePaginator.HasMorePages() {
		roleListOutput, roleListerr := rolePaginator.NextPage(context.TODO())
		if roleListerr != nil {
			return nil, roleListerr
		}
		for _, roleOutput := range roleListOutput.RoleList {
			var notOverriden = false
			account := sharedStructs.AccountIdNameRole{
				Id:        roleOutput.AccountId,
				Name:      accountOutput.AccountName,
				Role:      roleOutput.RoleName,
				Overriden: &notOverriden,
			}
			roles = append(roles, account)
		}
	}
	return roles, nil
}

func fetchAccountlist(ssoClient *sso.Client, token *string) ([]sharedStructs.AccountIdNameRole, error) {
	accountPaginator := sso.NewListAccountsPaginator(ssoClient, &sso.ListAccountsInput{
		AccessToken: token,
	})

	var allAccounts []ssoTypes.AccountInfo
	for accountPaginator.HasMorePages() {
		sSOOutput, errPaginatorError := accountPaginator.NextPage(context.TODO())
		if errPaginatorError != nil {
			return nil, errPaginatorError
		}
		allAccounts = append(allAccounts, sSOOutput.AccountList...)
	}

	if len(allAccounts) == 0 {
		return nil, nil
	}

	maxWorkers := runtime.NumCPU()
	if maxWorkers < 2 {
		maxWorkers = 2
	}
	if maxWorkers > 8 {
		maxWorkers = 8
	}

	accountCh := make(chan ssoTypes.AccountInfo, len(allAccounts))
	for _, accountOutput := range allAccounts {
		accountCh <- accountOutput
	}
	close(accountCh)

	var (
		mu          sync.Mutex
		accountList []sharedStructs.AccountIdNameRole
		firstErr    error
	)

	var wg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for accountOutput := range accountCh {
				roles, err := fetchRolesForAccount(ssoClient, token, accountOutput)
				if err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
					return
				}

				mu.Lock()
				accountList = append(accountList, roles...)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	if firstErr != nil {
		return accountList, firstErr
	}
	return accountList, nil
}

func updateSettings(SettingsInterface interfaces.SettingsInterface, UIproofcodeTextLabel *widget.Label) error {
	ssoRegion, _ := SettingsInterface.GetSSORegion()
	log.Printf("Using SSO Region: %s", ssoRegion)
	log.Printf("Using SSO URL: %s", SettingsInterface.GetSSOURL())

	aWSConfig := GetAWSConfig(ssoRegion)
	ssoClient := sso.NewFromConfig(aWSConfig)
	token, tokenErr := getAccessToken(SettingsInterface, aWSConfig, UIproofcodeTextLabel)
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
