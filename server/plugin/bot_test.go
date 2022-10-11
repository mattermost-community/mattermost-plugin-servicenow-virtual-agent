package plugin

import (
	"testing"

	"github.com/Brightscout/mattermost-plugin-servicenow-virtual-agent/server/testutils"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest/mock"
	"github.com/stretchr/testify/require"
)

func TestDM(t *testing.T) {
	for _, testCase := range []struct {
		description    string
		mockChannel    *model.Channel
		mockChannelErr *model.AppError
		mockPostErr    *model.AppError
		mockPost       *model.Post
	}{
		{
			description:    "Message is successfully posted",
			mockChannel:    &model.Channel{},
			mockChannelErr: nil,
			mockPostErr:    nil,
			mockPost:       &model.Post{},
		},
		{
			description:    "Channel is not found",
			mockChannel:    nil,
			mockChannelErr: &model.AppError{},
			mockPostErr:    nil,
			mockPost:       &model.Post{},
		},
		{
			description:    "Post is not created because of error in CreatePost method",
			mockChannel:    &model.Channel{},
			mockChannelErr: nil,
			mockPostErr:    &model.AppError{},
			mockPost:       nil,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}

			mockAPI := &plugintest.API{}
			mockAPI.On("LogInfo", testutils.GetMockArgumentsWithType("string", 5)...).Return()
			mockAPI.On("LogError", testutils.GetMockArgumentsWithType("string", 5)...).Return()
			mockAPI.On("GetDirectChannel", mock.Anything, mock.Anything).Return(testCase.mockChannel, testCase.mockChannelErr)
			mockAPI.On("CreatePost", mock.Anything).Return(testCase.mockPost, testCase.mockPostErr)

			p.SetAPI(mockAPI)

			_, err := p.DM("mock-userID", "mockFormat")

			if testCase.mockChannel == nil || testCase.mockPost == nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			_, err = p.DMWithAttachments("mock-userID", &model.SlackAttachment{})
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
	}{
		{
			description: "Ephemeral post is successfully created",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}

			mockAPI := &plugintest.API{}
			mockAPI.On("LogInfo", testutils.GetMockArgumentsWithType("string", 5)...).Return()
			mockAPI.On("LogError", testutils.GetMockArgumentsWithType("string", 3)...).Return()
			mockAPI.On("SendEphemeralPost", mock.Anything, mock.Anything).Return(&model.Post{})
			p.SetAPI(mockAPI)

			p.Ephemeral("mock-userID", "mockChannelID", "mockFormat")

			mockAPI.AssertNumberOfCalls(t, "SendEphemeralPost", 1)
		})
	}
}
