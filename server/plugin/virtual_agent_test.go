package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"bou.ke/monkey"
	"github.com/Brightscout/mattermost-plugin-servicenow-virtual-agent/server/testutils"
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

func Test_CreateOutputCardImageAttachment(t *testing.T) {
	for _, testCase := range []struct {
		description string
		body        *OutputCardImageData
		response    *model.SlackAttachment
	}{
		{
			description: "CreateOutputCardImageAttachment returns proper slack attachment",
			body: &OutputCardImageData{
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
		body        *OutputCardVideoData
		response    *model.SlackAttachment
	}{
		{
			description: "CreateOutputCardVideoAttachment returns proper slack attachment",
			body: &OutputCardVideoData{
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
		body        *OutputCardRecordData
		response    *model.SlackAttachment
	}{
		{
			description: "CreateOutputCardRecordAttachment returns proper slack attachment",
			body: &OutputCardRecordData{
				Title:    "mockTitle",
				Subtitle: "mockSubtitle",
				URL:      "mockURL",
				Fields: []*RecordFields{
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

func Test_CreateDefaultDateAttachment(t *testing.T) {
	p := Plugin{}

	for _, testCase := range []struct {
		description string
		body        *DefaultDate
		response    *model.SlackAttachment
	}{
		{
			description: "CreateDefaultDateAttachment returns proper slack attachment",
			body: &DefaultDate{
				UIType: "mockUIType",
				Label:  "mockLabel",
			},
			response: &model.SlackAttachment{
				Text: "mockLabel",
				Actions: []*model.PostAction{
					{
						Name: "Set mockUIType",
						Integration: &model.PostActionIntegration{
							URL: fmt.Sprintf("%s%s", p.GetPluginURLPath(), PathDateTimeSelectionDialog),
							Context: map[string]interface{}{
								"type": "mockUIType",
							},
						},
						Type: "button",
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

func Test_CreateOutputImagePost(t *testing.T) {
	defer monkey.UnpatchAll()

	mockBody := &OutputImage{
		Value:   "https://test/test.jpg",
		AltText: "mockAltText",
	}

	for _, testCase := range []struct {
		description           string
		body                  *OutputImage
		getDirectChannelError *model.AppError
		uploadFileError       *model.AppError
		isErrorExpected       bool
		expectedError         string
		readAllError          error
		httpGetError          error
		contentType           string
	}{
		{
			description: "Image post is created",
			body:        mockBody,
			contentType: "image/jpg",
		},
		{
			description:     "No image post is created due to invalid image URL",
			body:            mockBody,
			isErrorExpected: true,
			httpGetError:    errors.New("unsupported protocol scheme"),
			expectedError:   "unsupported protocol scheme",
		},
		{
			description: "Not able to get direct channel",
			body:        mockBody,
			getDirectChannelError: &model.AppError{
				Message: "error getting direct channel info",
			},
			isErrorExpected: true,
			expectedError:   "error getting direct channel info",
		},
		{
			description:     "Not able to upload file on Mattermost",
			body:            mockBody,
			uploadFileError: &model.AppError{},
			contentType:     "image/jpg",
		},
		{
			description:     "Error reading file data",
			body:            mockBody,
			readAllError:    errors.New("error reading file data"),
			isErrorExpected: true,
			expectedError:   "error reading file data",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}
			mockAPI := &plugintest.API{}

			mockAPI.On("LogError", testutils.GetMockArgumentsWithType("string", 5)...).Return()

			mockAPI.On("GetDirectChannel", testutils.GetMockArgumentsWithType("string", 2)...).Return(&model.Channel{}, testCase.getDirectChannelError)

			mockAPI.On("UploadFile", []byte{}, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&model.FileInfo{}, testCase.uploadFileError)

			p.SetAPI(mockAPI)

			monkey.Patch(http.Get, func(_ string) (*http.Response, error) {
				return &http.Response{
					Body: io.NopCloser(strings.NewReader("mockResponseBody")),
					Header: map[string][]string{
						"Content-Type": {testCase.contentType},
					},
				}, testCase.httpGetError
			})

			monkey.Patch(ioutil.ReadAll, func(_ io.Reader) ([]byte, error) {
				return []byte{}, testCase.readAllError
			})

			post, err := p.CreateOutputImagePost(testCase.body, "mockUserID")
			if testCase.isErrorExpected {
				assert.Contains(t, err.Error(), testCase.expectedError)
			} else {
				assert.NotNil(t, post)
				assert.Nil(t, err)
			}
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
			description: "CreateMessageAttachment returns a valid attachment",
			fileID:      "mockFileID",
			response: &MessageAttachment{
				URL:         "mockSiteURL" + p.GetPluginURLPath() + "/file/" + encode([]byte{}),
				ContentType: "mockMimeType",
				FileName:    "mockName",
			},
		},
		{
			description: "CreateMessageAttachment returns an error while getting file info",
			fileID:      "mockFileID",
			getFileInfoError: &model.AppError{
				Message: "error in getting the file info",
			},
			expectedError: "error getting the file info. Error: error in getting the file info",
		},
		{
			description:   "CreateMessageAttachment returns an error while marshaling file",
			fileID:        "mockFileID",
			marshalError:  errors.New("error in marshaling the file"),
			expectedError: "error occurred while marshaling the file. Error: error in marshaling the file",
		},
		{
			description:   "CreateMessageAttachment returns an error while encrypting file",
			fileID:        "mockFileID",
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
