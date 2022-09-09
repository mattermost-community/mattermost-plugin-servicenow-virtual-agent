package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest/mock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			errMessage:  errors.New("mockErrMessage"),
			expectedErr: errors.New("failed to call virtual agent bot integration API: mockErrMessage"),
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
			attachment := &MessageAttachment{}

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
			errMessage:  errors.New("mockErrMessage"),
			expectedErr: errors.New("failed to start conversation with virtual agent bot: mockErrMessage"),
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
		body        *OutputLink
		response    *model.SlackAttachment
	}{
		{
			description: "CreateOutputLinkAttachment returns proper slack attachment",
			body: &OutputLink{
				Header: "mockHeader",
				Label:  "mockLabel",
				Value: OutputLinkValue{
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

func Test_CreateTopicPickerControlAttachment(t *testing.T) {
	p := Plugin{}

	for _, testCase := range []struct {
		description string
		body        *TopicPickerControl
		response    *model.SlackAttachment
	}{
		{
			description: "CreateTopicPickerControlAttachment returns proper slack attachment",
			body: &TopicPickerControl{
				PromptMessage: "mockPrompt",
				Options: []Option{{
					Label: "mockLabel",
				}},
			},
			response: &model.SlackAttachment{
				Text: "mockPrompt",
				Actions: []*model.PostAction{
					{
						Name: "Select an option...",
						Integration: &model.PostActionIntegration{
							URL: fmt.Sprintf("%s%s", p.GetPluginURLPath(), PathActionOptions),
						},
						Type: "select",
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
		body        *Picker
		response    *model.SlackAttachment
	}{
		{
			description: "CreatePickerAttachment returns proper slack attachment",
			body: &Picker{
				Label: "mockLabel",
				Options: []Option{{
					Label: "mockLabel",
				}},
			},
			response: &model.SlackAttachment{
				Actions: []*model.PostAction{
					{
						Name: "Select an option...",
						Integration: &model.PostActionIntegration{
							URL: fmt.Sprintf("%s%s", p.GetPluginURLPath(), PathActionOptions),
						},
						Type: "select",
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

func Test_CreateMessageAttachment(t *testing.T) {
	p := Plugin{}

	defer monkey.UnpatchAll()

	for _, testCase := range []struct {
		description      string
		fileID           string
		response         *MessageAttachment
		getFileInfoError *model.AppError
		marshalError     error
		encryptError     error
		expectedError    string
	}{
		{
			description: "CreateMessageAttachment returns proper attachment",
			fileID:      "mockFileID",
			response: &MessageAttachment{
				URL:         "mockSiteURL" + p.GetPluginURLPath() + "/file/" + encode([]byte{}),
				ContentType: "mockMimeType",
				FileName:    "mockName",
			},
		},
		{
			description: "CreateMessageAttachment returns error while getting file info",
			fileID:      "mockFileID",
			response:    nil,
			getFileInfoError: &model.AppError{
				Message: "mockError",
			},
			expectedError: "error getting file info. Error: : mockError, ",
		},
		{
			description:   "CreateMessageAttachment returns error while marshalling file",
			fileID:        "mockFileID",
			response:      nil,
			marshalError:  errors.New("mockError"),
			expectedError: "error occurred while marshaling the file. Error: mockError",
		},
		{
			description:   "CreateMessageAttachment returns error while encrypting file",
			fileID:        "mockFileID",
			response:      nil,
			encryptError:  errors.New("mockError"),
			expectedError: "error occurred while encrypting the file. Error: mockError",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {

			p.setConfiguration(
				&configuration{
					EncryptionSecret:  "mockEncryptionSecret",
					MattermostSiteURL: "mockSiteURL",
				})

			mockAPI := plugintest.API{}

			mockAPI.On("GetFileInfo", mock.AnythingOfType("string")).Return(&model.FileInfo{
				MimeType: "mockMimeType",
				Name:     "mockName",
			}, testCase.getFileInfoError)

			p.SetAPI(&mockAPI)

			monkey.Patch(json.Marshal, func(_ interface{}) ([]byte, error) {
				return []byte{}, testCase.marshalError
			})

			monkey.Patch(encrypt, func(_, _ []byte) ([]byte, error) {
				return []byte{}, testCase.encryptError
			})

			res, err := p.CreateMessageAttachment(testCase.fileID)

			assert.EqualValues(t, testCase.response, res)

			if testCase.expectedError != "" {
				assert.EqualError(t, err, testCase.expectedError)
			}
		})
	}
}
