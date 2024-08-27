// This file basically includes GUI and minimal calls to logic
package main

import (
	"runtime"
	"strings"
	"sync"

	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/cmd/fyne_settings/settings"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	idp "github.com/oappi/awsssoroleswitcher/awsLogic"
	creds "github.com/oappi/awsssoroleswitcher/credentialFileLogic"
	"github.com/oappi/awsssoroleswitcher/interfaces"
	"github.com/oappi/awsssoroleswitcher/sharedStructs"
)

var version = "self-compiled"
var awsSession sharedStructs.SessionInfo
var lock = &sync.Mutex{}

var localWriter = interfaces.IniLogic{AWSFolderLocation: creds.GetAWSFolderStripError()}
var gregion = ""
var accountsList = []string{"Connect to credential service first"} //format should be accountName|accountId|roleName
var awsAccountObjectsList []sharedStructs.AccountIdNameRole        //contains objects, not strings required by UI component
var gOptionSelection *widget.SelectEntry
var SettingsInterface interfaces.SettingsInterface
var SettingsObject sharedStructs.SSOSettingsObject //contains ssoclient and token to fetch credentials
var selectedSessionTime = "1 hour session"
var placeholderAccountName = "not set"

func main() {
	a := app.NewWithID("io.fyne.oappi.AWSRoleSwitcher")
	//a.SetIcon(theme.FyneLogo())
	w := a.NewWindow("AWS SSO Role Switcher")
	w.Resize(fyne.NewSize(550, 480))
	_, err := creds.GetAWSFolder(runtime.GOOS)
	if err != nil {
		errorPopUp(a, "OS not Supported")
	}
	SetSVGAsIcon(a, w)

	settingsItem := fyne.NewMenuItem("GUI Settings", func() {
		w := a.NewWindow("Fyne Settings")
		w.SetContent(settings.NewSettings().LoadAppearanceScreen(w))
		w.Resize(fyne.NewSize(480, 480))
		w.Show()
	})

	connectAWSSettings := fyne.NewMenuItem("Connect via AWS SSO", func() {
		go showAWSSSOSettings(a)
	})

	advancedMenu := fyne.NewMenu("Advanced",
		fyne.NewMenuItem("Dump  ALL keys to credential file", func() {
			go showdumpKeys(a)
		}))

	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("Info", func() {
			go showInfo(a)
		}))

	connectMenu := fyne.NewMenu("Connect", connectAWSSettings)
	file := fyne.NewMenu("File", settingsItem)
	//file.Items = append(file.Items, fyne.NewMenuItemSeparator(), guiSettingsItem)

	mainMenu := fyne.NewMainMenu(
		// a quit item will be appended to our first menu
		file,
		connectMenu,
		advancedMenu,
		helpMenu,
	)
	w.SetMainMenu(mainMenu)
	w.SetMaster()

	accountName := widget.NewLabel(placeholderAccountName)
	accountName.TextStyle.Bold = true
	accountName.TextStyle.Italic = true
	accountName.Alignment = fyne.TextAlignLeading

	//reconnectButton.Importance = 1
	intro := widget.NewLabel("An introduction would probably go\nhere, as well as a")
	intro.Wrapping = fyne.TextWrapWord
	allElements, _ := filteredListForSelect(nil)
	accountSelectEntry := widget.NewSelectEntry(allElements)
	//timerOptions := []string{"1 hour session", "2 hour session", "4 hour session", "8 hour session", "12 hour session"}
	//timerSelectEntry := widget.NewSelectEntry(timerOptions)
	//timerSelectEntry.SetPlaceHolder("1 hour session")
	gOptionSelection = accountSelectEntry
	accountSelectEntry.PlaceHolder = "Type or select an account"
	accountSelectEntry.OnChanged = func(input string) {

		filteredList, match := filteredListForSelect(&input)
		accountSelectEntry.SetOptions(filteredList)
		if match {
			connectError := ssoConnectAccount(SettingsObject.SsoClient, SettingsObject.SSOAccessToken, accountSelectEntry.Text, localWriter)
			if connectError != nil {
				popError(a, connectError)
				accountSelectEntry.SetText("")
			} else {
				accountName.SetText(accountSelectEntry.Text)
				accountName.Alignment = fyne.TextAlignLeading
			}
			accountName.Alignment = fyne.TextAlignCenter

		}
	}
	/*timerSelectEntry.OnChanged = func(input string) {

		_, match := filteredCustomListForSelect(&input, timerOptions)
		if match {
			selectedSessionTime = input
		}
	}*/
	reconnectButton := widget.NewButton("Reconnect", func() {
		connectError := ssoConnectAccount(SettingsObject.SsoClient, SettingsObject.SSOAccessToken, accountSelectEntry.Text, localWriter)
		if connectError != nil {
			popError(a, connectError)
		}

	})
	openBrowserButton := widget.NewButton("Open in Browser", func() {
		idp.LoginBrowser(accountName.Text, awsSession, SettingsInterface)
	})

	reconnectButton.Importance = 0
	//openBrowserButton
	acclabelOpenBrowser := container.NewVSplit(accountName, openBrowserButton)
	bottomComponents := container.NewVSplit(acclabelOpenBrowser, reconnectButton)
	searchselect := container.NewVSplit(accountSelectEntry, bottomComponents)
	/*timeselector := container.NewVSplit(timerSelectEntry, searchselect)
	timeselector.Offset = 0.1
	*/
	searchselect.Offset = 0.1

	//searchselect := container.NewAdaptiveGrid(1, optionSelectEntry, accountName)
	w.SetContent(searchselect)
	w.Resize(fyne.NewSize(240, 260))

	w.ShowAndRun()
}

func showInfo(a fyne.App) {
	win := a.NewWindow("Info")
	win.SetContent(widget.NewLabel("\n Author Ossi Ala-Peijari \n\n\n Lincese: GNU GENERAL PUBLIC LICENSE V3  \n\n\n" +
		"Version: " + version + "\n\n\n" + "Only official distribution site is github where full license can be read \n\n\n" +
		"https://github.com/oappi/AWSSSORoleSwitcher"))
	win.Resize(fyne.NewSize(350, 200))
	win.Show()
	win.Close()
}

func showAWSSSOSettings(a fyne.App) {
	win := a.NewWindow("Local Connect Settings")
	SSOURLLabel := widget.NewLabel("AWS-SSO URL")
	SSOURLText := widget.NewEntry()
	ssoRegionLabel := widget.NewLabel("SSO Region")
	ssoRegionText := widget.NewEntry()
	accountRegionLabel := widget.NewLabel("Account Region")
	accountRegionText := widget.NewEntry()
	aliasLabel := widget.NewLabel("Alias")
	aliasText := widget.NewEntry()

	ssoSettings, fetcherror := localWriter.GetSSOSettings()
	if fetcherror != nil {
		//customOpenError := errors.New("Could not read old settings. This is normal first time\n")
		//popError(a, customOpenError)
	} else {
		SSOURLText.SetPlaceHolder(ssoSettings.SsoURL)
		ssoRegionText.SetPlaceHolder(ssoSettings.SSORegion)
		accountRegionText.SetPlaceHolder(ssoSettings.AccountRegion)
		aliasText.SetPlaceHolder(ssoSettings.Alias)
	}

	labels := container.NewGridWithColumns(1, SSOURLLabel, ssoRegionLabel, accountRegionLabel, aliasLabel)
	textFields := container.NewGridWithColumns(1, SSOURLText, ssoRegionText, accountRegionText, aliasText)
	settingscontainer := container.NewGridWithColumns(2, labels, textFields)

	applySettingsButton := widget.NewButton("Connect", func() {
		SSOURLOption := OverRideSavedIfUserGivesInput(SSOURLText.Text, ssoSettings.SsoURL)
		SSoRegionOption := OverRideSavedIfUserGivesInput(ssoRegionText.Text, ssoSettings.SSORegion)
		AccountRegionOption := OverRideSavedIfUserGivesInput(accountRegionText.Text, ssoSettings.AccountRegion)
		UserAliasOption := OverRideSavedIfUserGivesInput(aliasText.Text, ssoSettings.Alias)
		var SSOSettings sharedStructs.SSOSessionSettings
		SSOSettings.SsoURL = SSOURLOption
		SSOSettings.SSORegion = SSoRegionOption
		SSOSettings.AccountRegion = AccountRegionOption
		SSOSettings.Alias = UserAliasOption

		SettingsInterface = interfaces.AWSSSOSettings{Lock: lock, SSOURL: SSOURLOption, Region: AccountRegionOption, SSORegion: SSoRegionOption, UserAlias: UserAliasOption, AWSFolderLocation: creds.GetAWSFolderStripError(), LocalWriter: localWriter}
		err := updateSettings(SettingsInterface)
		if err != nil {
			popError(a, err)

		} else {
			localWriter.SetSSOSettings(SSOSettings)
			win.Close()
		}
	})

	settingsplit := container.NewVSplit(settingscontainer, applySettingsButton)
	settingsplit.Offset = 0.9
	win.SetContent(settingsplit)
	win.Show()
	win.Close()
}

func hideSecret(password string) string {
	return strings.Repeat("*", utf8.RuneCountInString(password))
}

func replaceEmptyInputWithSavedValue(saved string, input string) string {
	outputString := ""
	if len(input) > 0 {
		outputString = input
	} else {
		outputString = saved
	}
	return outputString

}

// MARK:Dumpkeys
func showdumpKeys(a fyne.App) {
	win := a.NewWindow("Dumps all Account keys to credential file")
	infoLabel := widget.NewLabel("Dumps every AWS accounts keys to credential file with 1 hour timeout" +
		"\nNote that you should only use this when needed.")

	textField := container.NewGridWithColumns(1, infoLabel)
	infocontainer := container.NewGridWithColumns(1, textField)

	dumpCredentialsButton := widget.NewButton("Dump", func() {
		err := dumpKeys(SettingsInterface, SettingsObject, SettingsObject.SsoClient, localWriter)
		if err != nil {
			var errormessage = err.Error()
			go errorPopUp(a, errormessage)
		}
		win.Close()
	})

	settingsplit := container.NewVSplit(infocontainer, dumpCredentialsButton)

	settingsplit.Offset = 0.9
	win.SetContent(settingsplit)

	win.Show()
	win.Close()
}

func dumpKeys(interfaceSettings interfaces.SettingsInterface, SSoSettings sharedStructs.SSOSettingsObject, ssoclient *sso.Client, writer interfaces.IniLogic) error {
	accounts := []sharedStructs.AccountObject{}
	region, err := interfaceSettings.GetAccountRegion()
	if err != nil {
		return err
	}
	for _, account := range *SSoSettings.Accounts {
		accountWithCredentials, credentialFethError := fetchAccountCredentials(SSoSettings, account.Id, account.Role, account.Name)
		if credentialFethError != nil {
			return credentialFethError
		}
		accounts = append(accounts, accountWithCredentials)
	}

	writeErr := writer.DumpKeysToCredentialFile(accounts, &region)
	return writeErr
}

func popError(a fyne.App, err error) {
	var errormessage = err.Error()
	go errorPopUp(a, errormessage)
}

func errorPopUp(a fyne.App, message string) {

	win := a.NewWindow("Error")
	infoLabel := widget.NewLabel(message)

	textField := container.NewGridWithColumns(1, infoLabel)
	infocontainer := container.NewGridWithColumns(1, textField)

	applySettingsButton := widget.NewButton("Close", func() {
		win.Close()
	})

	settingsplit := container.NewVSplit(infocontainer, applySettingsButton)

	settingsplit.Offset = 0.9
	win.SetContent(settingsplit)
	win.Show()
	win.Close()
}
