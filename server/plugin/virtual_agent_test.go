package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest/mock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/constants"
	mock_plugin "github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/mocks"
	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/serializer"
	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/testutils"
)

func Test_SendMessageToVirtualAgentAPI(t *testing.T) {
	defer monkey.UnpatchAll()

	for _, testCase := range []struct {
		description string
		errMessage  error
		expectedErr error
	}{
		{
			description: "Message is successfully sent to Virtual Agent API",
			errMessage:  nil,
		},
		{
			description: "Error while sending message to Virtual Agent API",
			errMessage:  errors.New("error in calling the Virtual Agent API"),
			expectedErr: errors.New("failed to call virtual agent bot integration API: error in calling the Virtual Agent API"),
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			c := new(client)

			monkey.PatchInstanceMethod(reflect.TypeOf(c), "Call", func(_ *client, _, _, _ string, _ io.Reader, _ interface{}, _ url.Values) (responseData []byte, err error) {
				if testCase.errMessage != nil {
					return nil, testCase.errMessage
				}
				return nil, nil
			})
			attachment := &serializer.MessageAttachment{}

			err := c.SendMessageToVirtualAgentAPI("mock-userID", "mockMessage", true, attachment)
			if testCase.errMessage != nil {
				require.Error(t, err)
				require.EqualError(t, testCase.expectedErr, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_StartConverstaionWithVirtualAgent(t *testing.T) {
	defer monkey.UnpatchAll()

	for _, testCase := range []struct {
		description string
		userID      string
		errMessage  error
		expectedErr error
	}{
		{
			description: "Conversation is successfully started with Virtual Agent",
			errMessage:  nil,
		},
		{
			description: "Error in starting conversation with Virtual Agent",
			errMessage:  errors.New("error in calling the Virtual Agent API"),
			expectedErr: errors.New("failed to start conversation with virtual agent bot: error in calling the Virtual Agent API"),
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			c := new(client)

			monkey.PatchInstanceMethod(reflect.TypeOf(c), "Call", func(_ *client, _, _, _ string, _ io.Reader, _ interface{}, _ url.Values) (responseData []byte, err error) {
				if testCase.errMessage != nil {
					return nil, testCase.errMessage
				}
				return nil, nil
			})

			err := c.StartConverstaionWithVirtualAgent("mock-userID")
			if testCase.errMessage != nil {
				require.Error(t, err)
				require.EqualError(t, testCase.expectedErr, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_CreateOutputLinkAttachment(t *testing.T) {
	for _, testCase := range []struct {
		description string
		body        *serializer.OutputLink
		response    *model.SlackAttachment
	}{
		{
			description: "CreateOutputLinkAttachment returns proper slack attachment",
			body: &serializer.OutputLink{
				Header: "mockHeader",
				Label:  "mockLabel",
				Value: serializer.OutputLinkValue{
					Action: "mockAction",
				},
			},
			response: &model.SlackAttachment{
				Pretext: "mockHeader",
				Text:    fmt.Sprintf("[%s](%s)", "mockLabel", "mockAction"),
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}

			res := p.CreateOutputLinkAttachment(testCase.body)
			require.EqualValues(t, testCase.response, res)
		})
	}
}

func Test_CreateOutputCardImageAttachment(t *testing.T) {
	for _, testCase := range []struct {
		description string
		body        *serializer.OutputCardImageData
		response    *model.SlackAttachment
	}{
		{
			description: "CreateOutputCardImageAttachment returns proper slack attachment",
			body: &serializer.OutputCardImageData{
				Image:       "mockImage",
				Title:       "mockTitle",
				Description: "mockDescription",
			},
			response: &model.SlackAttachment{
				Text:     "**mockTitle**\nmockDescription",
				ImageURL: "mockImage",
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}

			res := p.CreateOutputCardImageAttachment(testCase.body)

			require.EqualValues(t, testCase.response, res)
		})
	}
}

func Test_CreateOutputCardVideoAttachment(t *testing.T) {
	for _, testCase := range []struct {
		description string
		body        *serializer.OutputCardVideoData
		response    *model.SlackAttachment
	}{
		{
			description: "CreateOutputCardVideoAttachment returns proper slack attachment",
			body: &serializer.OutputCardVideoData{
				Title:       "mockTitle",
				Link:        "mockLink",
				URL:         "mockURL",
				Description: "mockDescription",
			},
			response: &model.SlackAttachment{
				Text: fmt.Sprintf("**[%s](%s)**\n%s", "mockTitle", "mockLink", "mockDescription"),
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}

			res := p.CreateOutputCardVideoAttachment(testCase.body)

			require.EqualValues(t, testCase.response, res)
		})
	}
}

func Test_CreateOutputCardRecordAttachment(t *testing.T) {
	for _, testCase := range []struct {
		description string
		body        *serializer.OutputCardRecordData
		response    *model.SlackAttachment
	}{
		{
			description: "CreateOutputCardRecordAttachment returns proper slack attachment",
			body: &serializer.OutputCardRecordData{
				Title:    "mockTitle",
				Subtitle: "mockSubtitle",
				URL:      "mockURL",
				Fields: []*serializer.RecordFields{
					{
						FieldLabel: "mockLabel",
						FieldValue: "mockValue",
					},
				},
			},
			response: &model.SlackAttachment{
				Fields: []*model.SlackAttachmentField{
					{
						Title: "mockTitle",
						Value: fmt.Sprintf("[%s](%s)", "mockSubtitle", "mockURL"),
					},
					{
						Title: "mockLabel",
						Value: "mockValue",
					},
				},
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}

			res := p.CreateOutputCardRecordAttachment(testCase.body)

			require.EqualValues(t, testCase.response, res)
		})
	}
}

func Test_CreateTopicPickerControlAttachment(t *testing.T) {
	p := Plugin{}

	for _, testCase := range []struct {
		description string
		body        *serializer.TopicPickerControl
		response    *model.SlackAttachment
	}{
		{
			description: "CreateTopicPickerControlAttachment returns proper slack attachment",
			body: &serializer.TopicPickerControl{
				PromptMessage: "mockPrompt",
				Options: []serializer.Option{{
					Label: "mockLabel",
				}},
			},
			response: &model.SlackAttachment{
				Text: "mockPrompt",
				Actions: []*model.PostAction{
					{
						Name: "Select an option...",
						Integration: &model.PostActionIntegration{
							URL: fmt.Sprintf("%s%s", p.GetPluginURLPath(), constants.PathActionOptions),
						},
						Type: model.POST_ACTION_TYPE_SELECT,
						Options: []*model.PostActionOptions{
							{
								Text:  "mockLabel",
								Value: "mockLabel",
							},
						},
					},
				},
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			res := p.CreateTopicPickerControlAttachment(testCase.body)
			require.EqualValues(t, testCase.response, res)
		})
	}
}

func Test_CreatePickerAttachment(t *testing.T) {
	p := Plugin{}

	for _, testCase := range []struct {
		description string
		body        *serializer.Picker
		response    *model.SlackAttachment
	}{
		{
			description: "CreatePickerAttachment returns proper slack attachment",
			body: &serializer.Picker{
				Label: "mockLabel",
				Options: []serializer.Option{{
					Label: "mockLabel",
				}},
			},
			response: &model.SlackAttachment{
				Actions: []*model.PostAction{
					{
						Name: "Select an option...",
						Integration: &model.PostActionIntegration{
							URL: fmt.Sprintf("%s%s", p.GetPluginURLPath(), constants.PathActionOptions),
						},
						Type: model.POST_ACTION_TYPE_SELECT,
						Options: []*model.PostActionOptions{
							{
								Text:  "mockLabel",
								Value: "mockLabel",
							},
						},
					},
				},
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			res := p.CreatePickerAttachment(testCase.body)
			require.EqualValues(t, testCase.response, res)
		})
	}
}

func Test_CreateDefaultDateAttachment(t *testing.T) {
	p := Plugin{}

	for _, testCase := range []struct {
		description string
		body        *serializer.DefaultDate
		response    *model.SlackAttachment
	}{
		{
			description: "CreateDefaultDateAttachment returns proper slack attachment",
			body: &serializer.DefaultDate{
				UIType: "mockUIType",
				Label:  "mockLabel",
			},
			response: &model.SlackAttachment{
				Text: "mockLabel",
				Actions: []*model.PostAction{
					{
						Name: "Set mockUIType",
						Integration: &model.PostActionIntegration{
							URL: fmt.Sprintf("%s%s", p.GetPluginURLPath(), constants.PathSetDateTimeDialog),
							Context: map[string]interface{}{
								"type": "mockUIType",
							},
						},
						Type: model.POST_ACTION_TYPE_BUTTON,
					},
				},
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			res := p.CreateDefaultDateAttachment(testCase.body)
			require.EqualValues(t, testCase.response, res)
		})
	}
}

func Test_CreateMessageAttachment(t *testing.T) {
	p := Plugin{}

	defer monkey.UnpatchAll()

	for _, testCase := range []struct {
		description   string
		userID        string
		response      *serializer.MessageAttachment
		setupAPI      func(api *plugintest.API)
		marshalError  error
		encryptError  error
		expectedError string
	}{
		{
			description: "CreateMessageAttachment returns a valid attachment",
			userID:      testutils.GetID(),
			response: &serializer.MessageAttachment{
				URL:         "mockSiteURL" + p.GetPluginURLPath() + "/file/" + encode([]byte{}),
				ContentType: "mockMimeType",
				FileName:    "mockName",
			},
			setupAPI: func(api *plugintest.API) {
				api.On("GetFileInfo", mock.AnythingOfType("string")).Return(testutils.GetFile(false), nil)
			},
		},
		{
			description: "CreateMessageAttachment returns an error while getting file info",
			userID:      testutils.GetID(),
			setupAPI: func(api *plugintest.API) {
				api.On("GetFileInfo", mock.AnythingOfType("string")).Return(nil, testutils.GetAppError("error in getting the file info"))
			},
			expectedError: "error getting the file info. Error: error in getting the file info",
		},
		{
			description: "CreateMessageAttachment returns an error because the file is already deleted",
			userID:      testutils.GetID(),
			setupAPI: func(api *plugintest.API) {
				api.On("GetFileInfo", mock.AnythingOfType("string")).Return(testutils.GetFile(true), nil)
			},
			expectedError: "file is deleted from the server",
		},
		{
			description: "CreateMessageAttachment returns an error because the file does not belong to the user",
			userID:      "mock-userID",
			setupAPI: func(api *plugintest.API) {
				api.On("GetFileInfo", mock.AnythingOfType("string")).Return(testutils.GetFile(false), nil)
			},
			expectedError: "file does not belong to the Mattermost user: mock-userID",
		},
		{
			description: "CreateMessageAttachment returns an error while marshaling file",
			userID:      testutils.GetID(),
			setupAPI: func(api *plugintest.API) {
				api.On("GetFileInfo", mock.AnythingOfType("string")).Return(testutils.GetFile(false), nil)
			},
			marshalError:  errors.New("error in marshaling the file"),
			expectedError: "error occurred while marshaling the file. Error: error in marshaling the file",
		},
		{
			description: "CreateMessageAttachment returns an error while encrypting file",
			userID:      testutils.GetID(),
			setupAPI: func(api *plugintest.API) {
				api.On("GetFileInfo", mock.AnythingOfType("string")).Return(testutils.GetFile(false), nil)
			},
			encryptError:  errors.New("error in encrypting the file"),
			expectedError: "error occurred while encrypting the file. Error: error in encrypting the file",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p.setConfiguration(
				&configuration{
					EncryptionSecret:  "mockEncryptionSecret",
					MattermostSiteURL: "mockSiteURL",
				})

			mockAPI := &plugintest.API{}
			testCase.setupAPI(mockAPI)
			p.SetAPI(mockAPI)

			monkey.Patch(json.Marshal, func(_ interface{}) ([]byte, error) {
				return []byte{}, testCase.marshalError
			})

			monkey.Patch(encrypt, func(_, _ []byte) ([]byte, error) {
				return []byte{}, testCase.encryptError
			})

			res, err := p.CreateMessageAttachment(testutils.GetID(), testCase.userID)

			assert.EqualValues(t, testCase.response, res)

			if testCase.expectedError != "" {
				assert.EqualError(t, err, testCase.expectedError)
			}
		})
	}
}

func Test_HandleCarouselInput(t *testing.T) {
	p := Plugin{}
	for _, test := range []struct {
		description   string
		expectedError error
		setupPlugin   func(p *Plugin, api *plugintest.API)
		setupStore    func(s *mock_plugin.MockStore)
	}{
		{
			description: "carousel is handled successfully with no error",
			setupPlugin: func(p *Plugin, api *plugintest.API) {
				monkey.PatchInstanceMethod(reflect.TypeOf(p), "DMWithAttachments", func(_ *Plugin, _ string, _ ...*model.SlackAttachment) (string, error) {
					return testutils.GetID(), nil
				})

				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 5)...).Return()
			},
			setupStore: func(s *mock_plugin.MockStore) {
				s.EXPECT().StorePostIDs(testutils.GetID(), []string{testutils.GetID()}).Return(errors.New("unable to store post IDs"))
			},
		},
		{
			description: "error while sending attachments to the user and IsCharCountSafe returns false",
			setupPlugin: func(p *Plugin, api *plugintest.API) {
				monkey.PatchInstanceMethod(reflect.TypeOf(p), "DMWithAttachments", func(_ *Plugin, _ string, _ ...*model.SlackAttachment) (string, error) {
					return testutils.GetID(), errors.New("error in sending attachments to the user")
				})
				monkey.PatchInstanceMethod(reflect.TypeOf(p), "IsCharCountSafe", func(_ *Plugin, _ []*model.SlackAttachment) bool {
					return false
				})
			},
			setupStore:    func(s *mock_plugin.MockStore) {},
			expectedError: errors.New("error in sending attachments to the user"),
		},
	} {
		t.Run(test.description, func(t *testing.T) {
			defer monkey.UnpatchAll()
			mockAPI := &plugintest.API{}
			test.setupPlugin(&p, mockAPI)
			p.SetAPI(mockAPI)
			defer mockAPI.AssertExpectations(t)
			mockCtrl := gomock.NewController(t)
			mockedStore := mock_plugin.NewMockStore(mockCtrl)
			test.setupStore(mockedStore)
			p.store = mockedStore

			err := p.HandleCarouselInput(testutils.GetID(), testutils.GetPickerBodyWithCarouselOptions(3))
			if test.expectedError == nil {
				require.Nil(t, err)
			} else {
				assert.EqualError(t, err, test.expectedError.Error())
			}
		})
	}
}

func Test_HandlePreviousCarouselPosts(t *testing.T) {
	p := Plugin{}
	for _, test := range []struct {
		description string
		setupAPI    func(api *plugintest.API)
		setupStore  func(s *mock_plugin.MockStore)
	}{
		{
			description: "error in loading postIDs",
			setupAPI: func(api *plugintest.API) {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 5)...).Return()
			},
			setupStore: func(s *mock_plugin.MockStore) {
				s.EXPECT().LoadPostIDs(testutils.GetID()).Return(nil, errors.New("error in loading post IDs"))
			},
		},
		{
			description: "no postIDs present in KV store",
			setupAPI:    func(api *plugintest.API) {},
			setupStore: func(s *mock_plugin.MockStore) {
				s.EXPECT().LoadPostIDs(testutils.GetID()).Return([]string{}, nil)
			},
		},
		{
			description: "error in storing postIDs and getting post from API",
			setupAPI: func(api *plugintest.API) {
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 5)...).Return()
				api.On("GetPost", testutils.GetID()).Return(nil, testutils.GetAppError("error in getting post"))
			},
			setupStore: func(s *mock_plugin.MockStore) {
				s.EXPECT().LoadPostIDs(testutils.GetID()).Return([]string{testutils.GetID()}, nil)
				s.EXPECT().StorePostIDs(testutils.GetID(), []string{}).Return(errors.New("error in storing post IDs"))
			},
		},
		{
			description: "error in updating post",
			setupAPI: func(api *plugintest.API) {
				api.On("GetPost", testutils.GetID()).Return(testutils.GetPostWithAttachments(2), nil)
				api.On("UpdatePost", mock.AnythingOfType("*model.Post")).Return(nil, testutils.GetAppError("error in updating post"))
				api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 5)...).Return()
			},
			setupStore: func(s *mock_plugin.MockStore) {
				s.EXPECT().LoadPostIDs(testutils.GetID()).Return([]string{testutils.GetID()}, nil)
				s.EXPECT().StorePostIDs(testutils.GetID(), []string{}).Return(nil)
			},
		},
	} {
		t.Run(test.description, func(t *testing.T) {
			mockAPI := &plugintest.API{}
			test.setupAPI(mockAPI)
			p.SetAPI(mockAPI)
			mockCtrl := gomock.NewController(t)
			mockedStore := mock_plugin.NewMockStore(mockCtrl)
			test.setupStore(mockedStore)
			p.store = mockedStore

			p.handlePreviousCarouselPosts(testutils.GetID())
		})
	}
}
