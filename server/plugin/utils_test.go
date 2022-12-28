package plugin

import (
	"encoding/json"
	"fmt"
	"testing"

	"bou.ke/monkey"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/testutils"
)

func Test_LogAndSendErrorToUser(t *testing.T) {
	t.Run("Error is successfully sent to the user", func(t *testing.T) {
		p := Plugin{}

		mockAPI := &plugintest.API{}
		mockAPI.On("LogError", testutils.GetMockArgumentsWithType("string", 3)...).Return()
		mockAPI.On("SendEphemeralPost", mock.Anything, mock.Anything).Return(&model.Post{})
		p.SetAPI(mockAPI)

		p.logAndSendErrorToUser("mock-userID", "mock-channelID", "mockErrMesssage")

		res := p.generateUUID()
		require.NotNil(t, res)
	})
}

func Test_validateDate(t *testing.T) {
	for _, testCase := range []struct {
		description string
		date        string
		expected    string
	}{
		{
			description: "Date is empty",
			date:        "",
			expected:    "Please enter a valid date",
		},
		{
			description: "Date is in incorrect format",
			date:        "1234:12:12",
			expected:    "Please enter a valid date",
		},
		{
			description: "Month is out of range",
			date:        "1234-14-12",
			expected:    "Please enter a valid date",
		},
		{
			description: "Year is out of range",
			date:        "1000-12-12",
			expected:    "Please enter year from",
		},
		{
			description: "Date is correct",
			date:        "2022-09-12",
			expected:    "",
		},
		{
			description: "Date is out of range",
			date:        "2022-09-31",
			expected:    "Please enter a valid date",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}
			res := p.validateDate(testCase.date)

			assert.Contains(t, res, testCase.expected)
		})
	}
}

func Test_validateTime(t *testing.T) {
	for _, testCase := range []struct {
		description string
		time        string
		expected    string
	}{
		{
			description: "Time is empty",
			time:        "",
			expected:    "Please enter a valid time",
		},
		{
			description: "Time is in incorrect format",
			time:        "12-12",
			expected:    "Please enter a valid time",
		},
		{
			description: "Hour is out of range",
			time:        "25:14",
			expected:    "Please enter a valid time",
		},
		{
			description: "Minute is out of range",
			time:        "10:65",
			expected:    "Please enter a valid time",
		},
		{
			description: "Time is correct",
			time:        "12:32",
			expected:    "",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}
			res := p.validateTime(testCase.time)

			assert.EqualValues(t, testCase.expected, res)
		})
	}
}

func Test_IsCharCountSafe(t *testing.T) {
	p := Plugin{}
	for _, test := range []struct {
		description  string
		marshalError error
	}{
		{
			description: "result is returned successfully with no error",
		},
		{
			description:  "result is returned successfully with an error",
			marshalError: fmt.Errorf("error in marshaling the attachments"),
		},
	} {
		t.Run(test.description, func(t *testing.T) {
			mockAPI := &plugintest.API{}
			if test.marshalError != nil {
				mockAPI.On("LogDebug", testutils.GetMockArgumentsWithType("string", 3)...).Return()
			}
			monkey.Patch(json.Marshal, func(_ interface{}) ([]byte, error) {
				return []byte{}, test.marshalError
			})

			p.SetAPI(mockAPI)
			_ = p.IsCharCountSafe([]*model.SlackAttachment{})
		})
	}
}
