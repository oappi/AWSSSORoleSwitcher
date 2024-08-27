package credentialFileLogic

import (
	"errors"
	"os"
	"runtime"
)

/*
getLocation returns aws credential location for multiple OS systems based on logic given by AWS
*/
func GetAWSFolder(userOS string) (string, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch userOS {
	case "linux":
		return homedir + "/.aws/", nil
	case "windows":
		return homedir + "\\.aws\\", nil
	case "darwin":
		return homedir + "/.aws/", nil
	default:
		return "", errors.New("OS not supported")
	}
}

func GetAWSFolderStripError() string {
	location, _ := GetAWSFolder(runtime.GOOS)
	return location
}
