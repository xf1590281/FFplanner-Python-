package management

import (
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"io/ioutil"

	"github.com/nofdev/fastforward/Godeps/_workspace/src/github.com/Azure/go-pkcs12"
)

// ClientFromPublishSettingsData unmarshalls the contents of a publish settings file
// from https://manage.windowsazure.com/publishsettings.
// If subscriptionID is left empty, the first subscription in the file is used.
func ClientFromPublishSettingsData(settingsData []byte, subscriptionID string) (client Client, err error) {
	return ClientFromPublishSettingsDataWithConfig(settingsData, subscriptionID, DefaultConfig())
}

// ClientFromPublishSettingsFile reads a publish settings file downloaded from https://manage.windowsazure.com/publishsettings.
// If subscriptionID is left empty, the first subscription in the file is used.
func ClientFromPublishSettingsFile(filePath, subscriptionID string) (client Client, err error) {
	return ClientFromPublishSettingsFileWithConfig(filePath, subscriptionID, DefaultConfig())
}

// ClientFromPublishSettingsFileWithConfig reads a publish settings file downloaded from https://manage.windowsazure.com/publishsettings.
// If subscriptionID is left empty, the first subscription in the file is used.
func ClientFromPublishSettingsFileWithConfig(filePath, subscriptionID string, config ClientConfig) (client Client, err error) {
	if filePath == "" {
		return client, fmt.Errorf(errParamNotSpecified, "filePath")
	}

	publishSettingsContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		return client, err
	}

	return ClientFromPublishSettingsDataWithConfig(publishSettingsContent, subscriptionID, config)
}

// ClientFromPublishSettingsDataWithConfig unmarshalls the contents of a publish settings file
// from https://manage.windowsazure.com/publishsettings.
// If subscriptionID is left empty, the first subscription in the string is used.
func ClientFromPublishSettingsDataWithConfig(data []byte, subscriptionID string, config ClientConfig) (client Client, err error) {
	publishData := publishData{}
	if err = xml.Unmarshal(data, &publishData); err != nil {
		return client, err
	}

	for _, profile := range publishData.PublishProfiles {
		for _, sub := range profile.Subscriptions {
			if sub.ID == subscriptionID || subscriptionID == "" {
				base64Cert := sub.ManagementCertificate
				if base64Cert == "" {
					base64Cert = profile.ManagementCertificate
				}

				pfxData, err := base64.StdEncoding.DecodeString(base64Cert)
				if err != nil {
					return client, err
				}

				pems, err := pkcs12.ConvertToPEM(pfxData, nil)

				cert := []byte{}
				for _, b := range pems {
					cert = append(cert, pem.EncodeToMemory(b)...)
				}

				config.ManagementURL = sub.ServiceManagementURL
				return makeClient(sub.ID, cert, config)
			}
		}
	}

	return client, fmt.Errorf("could not find subscription '%s' in settings provided", subscriptionID)
}

type publishSettings struct {
	SubscriptionID   string
	SubscriptionCert []byte
	SubscriptionKey  []byte
}

type publishData struct {
	XMLName         xml.Name         `xml:"PublishData"`
	PublishProfiles []publishProfile `xml:"PublishProfile"`
}

type publishProfile struct {
	XMLName               xml.Name       `xml:"PublishProfile"`
	SchemaVersion         string         `xml:",attr"`
	PublishMethod         string         `xml:",attr"`
	URL                   string         `xml:"Url,attr"`
	ManagementCertificate string         `xml:",attr"`
	Subscriptions         []subscription `xml:"Subscription"`
}

type subscription struct {
	XMLName               xml.Name `xml:"Subscription"`
	ServiceManagementURL  string   `xml:"ServiceManagementUrl,attr"`
	ID                    string   `xml:"Id,attr"`
	Name                  string   `xml:",attr"`
	ManagementCertificate string   `xml:",attr"`
}
