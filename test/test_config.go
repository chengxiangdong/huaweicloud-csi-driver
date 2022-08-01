package test

import (
	"fmt"
	"os"
	"testing"

	"github.com/huaweicloud/huaweicloud-csi-driver/pkg/config"
	"github.com/huaweicloud/huaweicloud-csi-driver/pkg/utils"
)


const configFile = "./cloud_config"

var (
	Region    = os.Getenv("HW_REGION_NAME")
	AccessKey = os.Getenv("HW_ACCESS_KEY")
	SecretKey = os.Getenv("HW_SECRET_KEY")
	ProjectID = os.Getenv("HW_PROJECT_ID")
)

func LoadConfig(t *testing.T) *config.CloudCredentials {
	err := initConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	defer utils.DeleteFile(configFile)

	cc, err := config.LoadConfig(configFile)
	if err != nil {
		t.Fatal(err)
	}
	return cc
}

func initConfigFile() error {
	content := fmt.Sprintf(`[Global]
region=%s
access-key=%s
secret-key=%s
project-id=%s
`, Region, AccessKey, SecretKey, ProjectID)

	err := utils.WriteToFile(configFile, content)
	if err != nil {
		return fmt.Errorf("Error creating cloud config file: %s", err)
	}
	return nil
}

