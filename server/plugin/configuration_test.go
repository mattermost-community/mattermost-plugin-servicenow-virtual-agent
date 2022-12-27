package plugin

import (
	"testing"

	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/constants"
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
				ChannelCacheSize:            10000,
			},
		},
		{
			description: "invalid configuration: ServiceNow URL empty",
			config: &configuration{
				ServiceNowURL: "",
			},
			errMsg: constants.EmptyServiceNowURLErrorMessage,
		},
		{
			description: "invalid configuration: ServiceNowOAuthClientID empty",
			config: &configuration{
				ServiceNowURL:           "mockServiceNowURL",
				ServiceNowOAuthClientID: "",
			},
			errMsg: constants.EmptyServiceNowOAuthClientIDErrorMessage,
		},
		{
			description: "invalid configuration: ServiceNowOAuthClientSecret empty",
			config: &configuration{
				ServiceNowURL:               "mockServiceNowURL",
				ServiceNowOAuthClientID:     "mockServiceNowOAuthClientID",
				ServiceNowOAuthClientSecret: "",
			},
			errMsg: constants.EmptyServiceNowOAuthClientSecretErrorMessage,
		},
		{
			description: "invalid configuration: EncryptionSecret empty",
			config: &configuration{
				ServiceNowURL:               "mockServiceNowURL",
				ServiceNowOAuthClientID:     "mockServiceNowOAuthClientID",
				ServiceNowOAuthClientSecret: "mockServiceNowOAuthClientSecret",
				EncryptionSecret:            "",
			},
			errMsg: constants.EmptyEncryptionSecretErrorMessage,
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
			errMsg: constants.EmptyWebhookSecretErrorMessage,
		},
		{
			description: "invalid configuration: ChannelCacheSize invalid",
			config: &configuration{
				ServiceNowURL:               "mockServiceNowURL",
				ServiceNowOAuthClientID:     "mockServiceNowOAuthClientID",
				ServiceNowOAuthClientSecret: "mockServiceNowOAuthClientSecret",
				EncryptionSecret:            "mockEncryptionSecret",
				WebhookSecret:               "mockWebhookSecret",
				ChannelCacheSize:            -1,
			},
			errMsg: constants.InvalidChannelCacheSizeErrorMessage,
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
