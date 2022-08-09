package plugin

import (
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_LogAndSendErrorToUser(t *testing.T) {
	t.Run("Error is successfully sent to the user", func(t *testing.T) {
		p := Plugin{}
		mockAPI := &plugintest.API{}
		mockAPI.On("LogError", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("LogError error")

		mockAPI.On("SendEphemeralPost", mock.Anything, mock.Anything).Return(&model.Post{})

		p.SetAPI(mockAPI)

		p.logAndSendErrorToUser("mock-userID", "mock-channelID", "mockErrMesssage")

		res := p.generateUUID()
		require.NotNil(t, res)
	})
}
