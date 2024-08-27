package credentialFileLogic

import (
	"log"
	"os"
	"testing"
)

func createFolder(path string) {
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		log.Println(err)
	}
}

func createTestCredentialsFile(path string) {
	myfile, e := os.Create(path + "credentials")
	if e != nil {
		log.Fatal(e)
	}
	log.Println(myfile)
	myfile.Close()
}

func TestUpdateAWSKeys(t *testing.T) {
	var path = os.Getenv("HOME") + "/.aws/tests/"
	createFolder(path)
	createTestCredentialsFile(path)
	var testcredentialfile = UpdateShortTermAWSKeys("1234567", "098765", "5453653324", path)
	if testcredentialfile != nil {
		t.Fail()
	}
}
