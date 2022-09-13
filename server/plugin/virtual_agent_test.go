package plugin

import (
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

			err := c.SendMessageToVirtualAgentAPI("mock-userID", "mockMessage", true)

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
	for _, testCase := range []struct {
		description string
		userID      string
		errMessage  error
		expectedErr error
	}{
		{
			description: "Conversation is successfully started with VIrtual Agent",
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
			description: "CreatePickerAttachment returns proper stack attachment",
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

func Test_CreateOutputImagePost(t *testing.T) {
	defer monkey.UnpatchAll()

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
			body: &OutputImage{
				Value:   "https://test/test.jpg",
				AltText: "mockAltText",
			},
			contentType: "image/jpg",
		},
		{
			description: "No image post is created due to invalid image URL",
			body: &OutputImage{
				Value:   "htps://test/test.jpg",
				AltText: "mockAltText",
			},
			isErrorExpected: true,
			httpGetError:    errors.New("unsupported protocol scheme"),
			expectedError:   "unsupported protocol scheme",
		},
		{
			description: "Not able to get direct channel",
			body: &OutputImage{
				Value:   "https://test/test.jpg",
				AltText: "mockAltText",
			},
			getDirectChannelError: &model.AppError{
				Message: "mockErrorMessage",
			},
			isErrorExpected: true,
			expectedError:   "mockErrorMessage",
		},
		{
			description: "Not able to upload file on Mattermost",
			body: &OutputImage{
				Value:   "https://test/test.jpg",
				AltText: "mockAltText",
			},
			uploadFileError: &model.AppError{},
			contentType:     "image/jpg",
		},
		{
			description: "Error reading file data",
			body: &OutputImage{
				Value:   "https://test/test.jpg",
				AltText: "mockAltText",
			},
			readAllError:    errors.New("mockError"),
			isErrorExpected: true,
			expectedError:   "mockError",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}
			mockAPI := &plugintest.API{}

			mockAPI.On("LogInfo", testutils.GetMockArgumentsWithType("string", 5)...).Return("LogInfo error")

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
			}
		})
	}
}
