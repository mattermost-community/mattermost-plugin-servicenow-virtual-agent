package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValid(t *testing.T) {
	for _, testCase := range []struct {
		description string
		config      *configuration
		errMsg      string
	}{
		{
			description: "valid configuration: pre-registered app",
			config: &configuration{
				ServiceNowURL:               "mockServiceNowURL",
				ServiceNowOAuthClientID:     "mockServiceNowOAuthClientID",
				ServiceNowOAuthClientSecret: "mockServiceNowOAuthClientSecret",
				EncryptionSecret:            "mockEncryptionSecret",
				WebhookSecret:               "mockWebhookSecret",
			},
		},
		{
			description: "invalid configuration: ServiceNow URL empty",
			config: &configuration{
				ServiceNowURL:               "",
				ServiceNowOAuthClientID:     "mockServiceNowOAuthClientID",
				ServiceNowOAuthClientSecret: "mockServiceNowOAuthClientSecret",
				EncryptionSecret:            "mockEncryptionSecret",
				WebhookSecret:               "mockWebhookSecret",
			},
			errMsg: "serviceNow URL should not be empty",
		},
		{
			description: "invalid configuration: ServiceNowOAuthClientID empty",
			config: &configuration{
				ServiceNowURL:               "mockServiceNowURL",
				ServiceNowOAuthClientID:     "",
				ServiceNowOAuthClientSecret: "mockServiceNowOAuthClientSecret",
				EncryptionSecret:            "mockEncryptionSecret",
				WebhookSecret:               "mockWebhookSecret",
			},
			errMsg: "serviceNow OAuth clientID should not be empty",
		},
		{
			description: "invalid configuration: ServiceNowOAuthClientSecret empty",
			config: &configuration{
				ServiceNowURL:               "mockServiceNowURL",
				ServiceNowOAuthClientID:     "mockServiceNowOAuthClientID",
				ServiceNowOAuthClientSecret: "",
				EncryptionSecret:            "mockEncryptionSecret",
				WebhookSecret:               "mockWebhookSecret",
			},
			errMsg: "serviceNow OAuth clientSecret should not be empty",
		},
		{
			description: "invalid configuration: EncryptionSecret empty",
			config: &configuration{
				ServiceNowURL:               "mockServiceNowURL",
				ServiceNowOAuthClientID:     "mockServiceNowOAuthClientID",
				ServiceNowOAuthClientSecret: "mockServiceNowOAuthClientSecret",
				EncryptionSecret:            "",
				WebhookSecret:               "mockWebhookSecret",
			},
			errMsg: "encryption secret should not be empty",
		},
		{
			description: "invalid configuration: WebhookSecret empty",
			config: &configuration{
				ServiceNowURL:               "mockServiceNowURL",
				ServiceNowOAuthClientID:     "mockServiceNowOAuthClientID",
				ServiceNowOAuthClientSecret: "mockServiceNowOAuthClientSecret",
				EncryptionSecret:            "mockEncryptionSecret",
				WebhookSecret:               "",
			},
			errMsg: "webhook secret should not be empty",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			err := testCase.config.IsValid()
			if testCase.errMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
