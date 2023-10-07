package plugin

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/bluele/gcache"
	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"golang.org/x/oauth2"

	mock_plugin "github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/mocks"
	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/serializer"
	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/testutils"
)

func Test_MessageHasBeenPosted(t *testing.T) {
	defer monkey.UnpatchAll()

	for _, testCase := range []struct {
		description                       string
		Message                           string
		cacheGetError                     error
		cacheSetError                     error
		getChannelError                   *model.AppError
		getUserError                      error
		parseAuthTokenError               error
		sendMessageToVirtualAgentAPIError error
		createMessageAttachmentError      error
	}{
		{
			description: "Message is successfully sent to Virtual Agent when the channel is found in cache",
			Message:     "mockMessage",
		},
		{
			description:   "Message is successfully sent to Virtual Agent when the channel is not found in cache and error occurred while adding to cache",
			Message:       "mockMessage",
			cacheGetError: errors.New("key not found in cache"),
			cacheSetError: errors.New("error in setting value in cache"),
		},
		{
			description:     "Message is posted but failed to get current channel",
			Message:         "mockMessage",
			cacheGetError:   errors.New("key not found in cache"),
			getChannelError: &model.AppError{},
		},
		{
			description:  "Message is posted but failed to get user from KV store",
			getUserError: errors.New("error getting the user from KVstore"),
			Message:      "mockMessage",
		},
		{
			description:  "Message is posted but user is not connected to ServiceNow",
			getUserError: ErrNotFound,
			Message:      "mockMessage",
		},
		{
			description: "Message is posted but user is not connected to ServiceNow",
			Message:     "disconnect",
		},
		{
			description:         "Message is posted but failed to parse auth token",
			parseAuthTokenError: errors.New("error in parsing the auth token"),
			Message:             "mockMessage",
		},
		{
			description:                       "Message is posted but failed to send message to virtual agent API",
			sendMessageToVirtualAgentAPIError: errors.New("error in parsing the auth token"),
			Message:                           "mockMessage",
		},
		{
			description:                  "Message is posted but failed to create message attachment",
			createMessageAttachmentError: errors.New("error in creating message attachment"),
			Message:                      "mockMessage",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			mockedClient := mock_plugin.NewMockClient(mockCtrl)
			p, mockAPI := setupTestPlugin(&plugintest.API{}, nil)
			defer mockAPI.AssertExpectations(t)

			p.channelCache = &gcache.SimpleCache{}
			p.botUserID = "mock-botID"
			mockedClient.EXPECT().SendMessageToVirtualAgentAPI("", testCase.Message, true, &serializer.MessageAttachment{}).MinTimes(0).MaxTimes(1).Return(testCase.sendMessageToVirtualAgentAPIError)

			monkey.PatchInstanceMethod(reflect.TypeOf(p.channelCache), "Get", func(_ *gcache.SimpleCache, _ interface{}) (interface{}, error) {
				return true, testCase.cacheGetError
			})

			monkey.PatchInstanceMethod(reflect.TypeOf(p.channelCache), "SetWithExpire", func(_ *gcache.SimpleCache, _ interface{}, _ interface{}, _ time.Duration) error {
				return testCase.cacheSetError
			})

			if testCase.getChannelError != nil || testCase.parseAuthTokenError != nil || testCase.sendMessageToVirtualAgentAPIError != nil || (testCase.getUserError != nil && testCase.getUserError != ErrNotFound || testCase.createMessageAttachmentError != nil) {
				mockAPI.On("LogError", testutils.GetMockArgumentsWithType("string", 6)...).Return()
			}

			if testCase.cacheGetError != nil {
				mockAPI.On("GetChannel", "mockChannelID").Return(&model.Channel{
					Type: "D",
					Name: "mock-botID__mock",
				}, testCase.getChannelError)
			}

			if testCase.cacheSetError != nil {
				mockAPI.On("LogDebug", testutils.GetMockArgumentsWithType("string", 3)...).Return()
			}

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "ScheduleJob", func(_ *Plugin, _ string) error {
				return nil
			})

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "Ephemeral", func(_ *Plugin, _, _, _ string, _ ...interface{}) {})

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "GetUser", func(_ *Plugin, _ string) (*serializer.User, error) {
				return &serializer.User{}, testCase.getUserError
			})

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "DM", func(_ *Plugin, _, _ string, _ ...interface{}) (string, error) {
				return "mockPostID", nil
			})

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "DMWithAttachments", func(_ *Plugin, _ string, _ ...*model.SlackAttachment) (string, error) {
				return "mockPostID", nil
			})

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "ParseAuthToken", func(_ *Plugin, _ string) (*oauth2.Token, error) {
				return &oauth2.Token{}, testCase.parseAuthTokenError
			})

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "MakeClient", func(_ *Plugin, _ context.Context, _ *oauth2.Token) Client {
				return mockedClient
			})

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "CreateMessageAttachment", func(_ *Plugin, _, _ string) (*serializer.MessageAttachment, error) {
				return &serializer.MessageAttachment{}, testCase.createMessageAttachmentError
			})

			post := &model.Post{
				ChannelId: "mockChannelID",
				UserId:    "mock-userID",
				Message:   testCase.Message,
				FileIds:   []string{"mockFileID"},
			}

			p.MessageHasBeenPosted(&plugin.Context{}, post)
		})
	}
}
