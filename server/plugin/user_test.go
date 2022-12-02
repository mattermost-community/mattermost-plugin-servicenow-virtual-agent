package plugin

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/golang/mock/gomock"
	mock_plugin "github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/mocks"
	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/serializer"
	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/testutils"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func Test_GetUser(t *testing.T) {
	for _, testCase := range []struct {
		description string
		errMessage  error
		expectedErr string
		loadedUser  *serializer.User
	}{
		{
			description: "User is loaded successfully from KV store using mattermostID",
			errMessage:  nil,
			loadedUser:  &serializer.User{},
		},
		{
			description: "Error in loading user from KV store using mattermostID",
			errMessage:  errors.New("error in loading the user from KVstore"),
			loadedUser:  nil,
			expectedErr: "error in loading the user from KVstore",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}

			mockCtrl := gomock.NewController(t)
			mockedStore := mock_plugin.NewMockStore(mockCtrl)

			mockedStore.EXPECT().LoadUser("mock-userID").Return(testCase.loadedUser, testCase.errMessage)

			p.store = mockedStore

			user, err := p.GetUser("mock-userID")

			if testCase.expectedErr != "" {
				require.Nil(t, user)
				require.EqualError(t, err, testCase.expectedErr)
			} else {
				require.Nil(t, err)
				require.IsTypef(t, &serializer.User{}, user, "mockMsg")
			}
		})
	}
}

func Test_DisconnectUser(t *testing.T) {
	for _, testCase := range []struct {
		description string
		errMessage  error
		expectedErr string
	}{
		{
			description: "User is deleted successfully from KV store using mattermostID",
			errMessage:  nil,
		},
		{
			description: "Error in deleting user from KV store using mattermostID",
			errMessage:  errors.New("error in deleting the user from KVstore"),
			expectedErr: "error in deleting the user from KVstore",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}

			mockCtrl := gomock.NewController(t)
			mockedStore := mock_plugin.NewMockStore(mockCtrl)

			mockedStore.EXPECT().DeleteUser("mock-userID").Return(testCase.errMessage)

			p.store = mockedStore

			err := p.DisconnectUser("mock-userID")

			if testCase.expectedErr != "" {
				require.EqualError(t, err, testCase.expectedErr)
			} else {
				require.Nil(t, err)
			}
		})
	}
}

func Test_CreateDisconnectUserAttachment(t *testing.T) {
	t.Run("CreateDisconnectUserAttachment created successfully", func(t *testing.T) {
		p := Plugin{}

		disconnectUserPath := fmt.Sprintf("%s%s", p.GetPluginURLPath(), PathUserDisconnect)
		expectedResponse := &model.SlackAttachment{
			Title: DisconnectUserConfirmationMessge,
			Color: "#FF0000",
			Actions: []*model.PostAction{
				{
					Type: "button",
					Name: "Yes",
					Integration: &model.PostActionIntegration{
						URL: disconnectUserPath,
						Context: map[string]interface{}{
							DisconnectUserContextName: true,
						},
					},
				},
				{
					Type: "button",
					Name: "No",
					Integration: &model.PostActionIntegration{
						URL: disconnectUserPath,
						Context: map[string]interface{}{
							DisconnectUserContextName: false,
						},
					},
				},
			},
		}

		res := p.CreateDisconnectUserAttachment()
		require.EqualValues(t, res, expectedResponse)
	})
}

func Test_GetDisconnectUserPost(t *testing.T) {
	for _, testCase := range []struct {
		description                      string
		errMessage                       *model.AppError
		expectedErr                      string
		userID                           string
		getPostWithSlackAttachmentResult *model.Channel
	}{
		{
			description:                      "GetDisconnectUserPost returns proper response",
			errMessage:                       nil,
			userID:                           "mock-userID",
			getPostWithSlackAttachmentResult: &model.Channel{},
			expectedErr:                      "",
		},
		{
			description:                      "GetDisconnectUserPost return error because GetDirectChannel return error due to invalid userID",
			errMessage:                       &model.AppError{},
			expectedErr:                      "userID is invalid",
			userID:                           "invalid-UserID",
			getPostWithSlackAttachmentResult: nil,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}

			mockAPI := &plugintest.API{}

			mockAPI.On("LogError", testutils.GetMockArgumentsWithType("string", 5)...).Return()

			mockAPI.On("GetDirectChannel", testCase.userID, mock.AnythingOfType("string")).Return(testCase.getPostWithSlackAttachmentResult, testCase.errMessage)

			p.SetAPI(mockAPI)

			res, err := p.GetDisconnectUserPost(testCase.userID, "mockMessage")
			if testCase.expectedErr != "" {
				require.Nil(t, res)
				require.IsType(t, &model.AppError{}, err)
			} else {
				require.Nil(t, err)
				require.IsType(t, &model.Post{}, res)
			}
		})
	}
}

func Test_InitOAuth2(t *testing.T) {
	for _, testCase := range []struct {
		description   string
		errMessage    error
		expectedErr   string
		loadedUser    *serializer.User
		loadUserError error
	}{
		{
			description: "User is already connected to ServiceNow",
			expectedErr: "user is already connected to ServiceNow",
			loadedUser:  &serializer.User{},
		},
		{
			description:   "OAuth2 is initialized successfully",
			expectedErr:   "",
			loadUserError: errors.New("user is not present in KVstore"),
		},
		{
			description:   "Error occurred while storing oauth2 state",
			errMessage:    errors.New("error storing OAuth2 state"),
			expectedErr:   "error storing OAuth2 state",
			loadUserError: errors.New("user is not present in KVstore"),
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}

			mockCtrl := gomock.NewController(t)
			mockedStore := mock_plugin.NewMockStore(mockCtrl)

			mockedStore.EXPECT().LoadUser("mock-userID").Return(testCase.loadedUser, testCase.loadUserError)

			if testCase.loadUserError != nil {
				mockedStore.EXPECT().StoreOAuth2State(gomock.Any()).Return(testCase.errMessage)
			}

			p.store = mockedStore

			res, err := p.InitOAuth2("mock-userID")
			if testCase.expectedErr != "" {
				require.Equal(t, "", res)
				require.NotNil(t, err)
			} else {
				require.NotEqual(t, "", res)
				require.Nil(t, err)
			}
		})
	}
}

func Test_CompleteOAuth2(t *testing.T) {
	defer monkey.UnpatchAll()

	for _, testCase := range []struct {
		description                            string
		authedUserID                           string
		code                                   string
		state                                  string
		expectedErr                            string
		verifyOAuth2StateErr                   error
		exchangeError                          error
		getMeError                             error
		newEncodedAuthTokenError               error
		storeUserError                         error
		dMError                                error
		startConverstaionWithVirtualAgentError error
	}{
		{
			description:  "OAuth2 is completed successfully",
			authedUserID: "mock-authedUserID",
			code:         "mockCode",
			state:        "mockState_mock-authedUserID",
		},
		{
			description:  "OAuth2 fails because authedUserID is empty string",
			authedUserID: "",
			code:         "mockCode",
			state:        "mockState_mock-authedUserID",
			expectedErr:  "missing user, code or state",
		},
		{
			description:  "OAuth2 fails because code is empty string",
			authedUserID: "mock-authedUserID",
			code:         "",
			state:        "mockState_mock-authedUserID",
			expectedErr:  "missing user, code or state",
		},
		{
			description:  "OAuth2 fails because state is empty string",
			authedUserID: "mock-authedUserID",
			code:         "mockCode",
			state:        "",
			expectedErr:  "missing user, code or state",
		},
		{
			description:          "Error while veryfying oauth2 state",
			authedUserID:         "mock-authedUserID",
			code:                 "mockCode",
			state:                "mockState_mock-authedUserID",
			verifyOAuth2StateErr: errors.New("mockError"),
			expectedErr:          "missing stored state: mockError",
		},
		{
			description:  "OAuth2 fails because mattermostUserID does not match authedUserID",
			authedUserID: "mock-authedUserID",
			code:         "mockCode",
			state:        "mockState_mock-invalidAuthedUserID",
			expectedErr:  "not authorized, user ID mismatch",
		},
		{
			description:   "Error getting oauth2 token",
			authedUserID:  "mock-authedUserID",
			code:          "mockCode",
			state:         "mockState_mock-authedUserID",
			expectedErr:   "mockError",
			exchangeError: errors.New("mockError"),
		},
		{
			description:  "Error getting ServiceNow user",
			authedUserID: "mock-authedUserID",
			code:         "mockCode",
			state:        "mockState_mock-authedUserID",
			expectedErr:  "error getting the user details",
			getMeError:   errors.New("error getting the user details"),
		},
		{
			description:              "Error encoding the oauth2 token",
			authedUserID:             "mock-authedUserID",
			code:                     "mockCode",
			state:                    "mockState_mock-authedUserID",
			expectedErr:              "error in generating new OAuth token",
			newEncodedAuthTokenError: errors.New("error in generating new OAuth token"),
		},
		{
			description:    "Error storing user in KV store",
			authedUserID:   "mock-authedUserID",
			code:           "mockCode",
			state:          "mockState_mock-authedUserID",
			expectedErr:    "error in storing the user in KVstore",
			storeUserError: errors.New("error in storing the user in KVstore"),
		},
		{
			description:  "Error while posting ConnectSuccessMessage to user",
			authedUserID: "mock-authedUserID",
			code:         "mockCode",
			state:        "mockState_mock-authedUserID",
			expectedErr:  "error sending message to the user",
			dMError:      errors.New("error sending message to the user"),
		},
		{
			description:                            "Error while starting conversation with Virtual Agent",
			authedUserID:                           "mock-authedUserID",
			code:                                   "mockCode",
			state:                                  "mockState_mock-authedUserID",
			expectedErr:                            "error starting conversation with Virtual Agent",
			startConverstaionWithVirtualAgentError: errors.New("error starting conversation with Virtual Agent"),
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}

			mockCtrl := gomock.NewController(t)
			mockedStore := mock_plugin.NewMockStore(mockCtrl)

			if testCase.authedUserID != "" && testCase.code != "" && testCase.state != "" {
				mockedStore.EXPECT().VerifyOAuth2State(testCase.state).Return(testCase.verifyOAuth2StateErr)
			}

			monkey.PatchInstanceMethod(reflect.TypeOf(&oauth2.Config{}), "Exchange", func(_ *oauth2.Config, _ context.Context, _ string, _ ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
				return &oauth2.Token{}, testCase.exchangeError
			})

			monkey.PatchInstanceMethod(reflect.TypeOf(&p), "MakeClient", func(_ *Plugin, _ context.Context, _ *oauth2.Token) Client {
				return &client{}
			})

			monkey.PatchInstanceMethod(reflect.TypeOf(&client{}), "GetMe", func(_ *client, _ string) (*serializer.ServiceNowUser, error) {
				return &serializer.ServiceNowUser{}, testCase.getMeError
			})

			monkey.PatchInstanceMethod(reflect.TypeOf(&p), "NewEncodedAuthToken", func(_ *Plugin, _ *oauth2.Token) (string, error) {
				return "mockToken", testCase.newEncodedAuthTokenError
			})

			if testCase.state != "mockState_mock-invalidAuthedUserID" && testCase.authedUserID != "" && testCase.code != "" && testCase.state != "" && testCase.newEncodedAuthTokenError == nil && testCase.getMeError == nil && testCase.exchangeError == nil && testCase.verifyOAuth2StateErr == nil {
				mockedStore.EXPECT().StoreUser(gomock.Any()).Return(testCase.storeUserError)
			}

			monkey.PatchInstanceMethod(reflect.TypeOf(&p), "DM", func(_ *Plugin, _, _ string, _ ...interface{}) (string, error) {
				return "mockToken", testCase.dMError
			})

			monkey.PatchInstanceMethod(reflect.TypeOf(&client{}), "StartConverstaionWithVirtualAgent", func(_ *client, _ string) error {
				return testCase.startConverstaionWithVirtualAgentError
			})

			p.store = mockedStore

			err := p.CompleteOAuth2(testCase.authedUserID, testCase.code, testCase.state)
			if testCase.expectedErr != "" {
				require.EqualError(t, err, testCase.expectedErr)
			} else {
				require.Nil(t, err)
			}
		})
	}
}
