package plugin

import (
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest/mock"
	"github.com/stretchr/testify/require"
)

func TestDM(t *testing.T) {
	for _, testCase := range []struct {
		description      string
		mattermostUserID string
		format           string
		mockChannel      *model.Channel
		mockChannelErr   *model.AppError
		mockPostErr      *model.AppError
		mockPost         *model.Post
	}{
		{
			description:      "Message is successfully posted",
			mattermostUserID: "mockID",
			format:           "mockFormat",
			mockChannel:      &model.Channel{},
			mockChannelErr:   nil,
			mockPostErr:      nil,
			mockPost:         &model.Post{},
		},
		{
			description:      "Channel is not found",
			mattermostUserID: "mockID",
			format:           "mockFormat",
			mockChannel:      nil,
			mockChannelErr:   &model.AppError{},
			mockPostErr:      nil,
			mockPost:         &model.Post{},
		},
		{
			description:      "Post is not created because of error in CreatePost method",
			mattermostUserID: "mockID",
			format:           "mockFormat",
			mockChannel:      &model.Channel{},
			mockChannelErr:   nil,
			mockPostErr:      &model.AppError{},
			mockPost:         nil,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}
			mockAPI := &plugintest.API{}
			mockAPI.On("LogInfo", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("Logdebug error")
			mockAPI.On("LogError", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("LogError error")

			mockAPI.On("GetDirectChannel", mock.Anything, mock.Anything).Return(testCase.mockChannel, testCase.mockChannelErr)

			mockAPI.On("CreatePost", mock.Anything).Return(testCase.mockPost, testCase.mockPostErr)

			p.SetAPI(mockAPI)

			_, err := p.DM(testCase.mattermostUserID, testCase.format)

			if testCase.mockChannel == nil || testCase.mockPost == nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			_, err = p.DMWithAttachments(testCase.mattermostUserID, &model.SlackAttachment{})
			if testCase.mockChannel == nil || testCase.mockPost == nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEphemeral(t *testing.T) {
	for _, testCase := range []struct {
		description string
		userID      string
		channelID   string
		format      string
	}{
		{
			description: "Ephemeral post is successfully created",
			userID:      "mock-userID",
			channelID:   "mockChannelID",
			format:      "mockFormat",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}
			mockAPI := &plugintest.API{}
			mockAPI.On("LogInfo", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("Logdebug error")
			mockAPI.On("LogError", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("LogError error")

			mockAPI.On("SendEphemeralPost", mock.Anything, mock.Anything).Return(&model.Post{})

			p.SetAPI(mockAPI)

			p.Ephemeral(testCase.userID, testCase.channelID, testCase.format)
		})
	}
}
