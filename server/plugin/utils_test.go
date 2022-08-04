package plugin

import (
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLogAndSendErrorToUser(t *testing.T) {
	for _, testCase := range []struct {
		description string
		userID      string
		channelID   string
		errMessage  string
	}{
		{
			description: "Everything works fine",
			userID:      "mockID",
			channelID:   "mockChannelID",
			errMessage:  "mockErrMessage",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}
			mockAPI := &plugintest.API{}
			mockAPI.On("LogError", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("LogError error")

			mockAPI.On("SendEphemeralPost", mock.Anything, mock.Anything).Return(&model.Post{})

			p.SetAPI(mockAPI)

			p.logAndSendErrorToUser(testCase.userID, testCase.channelID, testCase.errMessage)

			res := p.generateUUID()
			require.NotNil(t, res)
		})
	}
}
