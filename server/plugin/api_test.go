package plugin

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"bou.ke/monkey"
	mock_plugin "github.com/Brightscout/mattermost-plugin-servicenow-virtual-agent/server/mocks"
	"github.com/Brightscout/mattermost-plugin-servicenow-virtual-agent/server/serializer"
	"github.com/golang/mock/gomock"
	"golang.org/x/oauth2"

	"github.com/Brightscout/mattermost-plugin-servicenow-virtual-agent/server/testutils"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type panicHandler struct {
}

func (ph panicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	panic("bad handler")
}

func TestWithRecovery(t *testing.T) {
	defer func() {
		if x := recover(); x != nil {
			require.Fail(t, "got panic")
		}
	}()

	p := Plugin{}
	api := &plugintest.API{}
	api.On("LogError", "Recovered from a panic", "URL", "http://random", "Error", "bad handler", "Stack", mock.Anything)
	p.SetAPI(api)

	ph := panicHandler{}
	handler := p.withRecovery(ph)

	req := httptest.NewRequest(http.MethodGet, "http://random", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.Body != nil {
		defer resp.Body.Close()
		_, err := io.Copy(ioutil.Discard, resp.Body)
		require.NoError(t, err)
	}
}

func TestPlugin_handleUserDisconnect(t *testing.T) {
	defer monkey.UnpatchAll()

	httpTestJSON := testutils.HTTPTest{
		T:       t,
		Encoder: testutils.EncodeJSON,
	}

	for name, test := range map[string]struct {
		httpTest                 testutils.HTTPTest
		request                  testutils.Request
		expectedResponse         testutils.ExpectedResponse
		userID                   string
		GetUserErr               error
		GetDisconnectUserPostErr error
		DisconnectUserErr        error
	}{
		"UserID id present in headers does not match and 'CheckAuth' fails": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathUserDisconnect),
				Body:   model.PostActionIntegrationRequest{},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusUnauthorized,
			},
			userID: "",
		},
		"Error while decoding request body": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathUserDisconnect),
				Body:   nil,
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID: "mock-userID",
		},
		"User is disconnected successfully": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathUserDisconnect),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						DisconnectUserContextName: true,
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID:                   "mock-userID",
			GetUserErr:               nil,
			GetDisconnectUserPostErr: nil,
			DisconnectUserErr:        nil,
		},
		"User not found and failed to create disconnect post": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathUserDisconnect),
				Body:   model.PostActionIntegrationRequest{},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID:                   "mock-userID",
			GetUserErr:               ErrNotFound,
			GetDisconnectUserPostErr: errors.New("mockErr"),
			DisconnectUserErr:        nil,
		},
		"User is found but error occurred while reading user from KV store": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathUserDisconnect),
				Body:   model.PostActionIntegrationRequest{},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID:                   "mock-userID",
			GetUserErr:               errors.New("mockError"),
			GetDisconnectUserPostErr: errors.New("mockError"),
			DisconnectUserErr:        nil,
		},
		"User not found and disconnect user post is created successfully": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathUserDisconnect),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						DisconnectUserContextName: "mockContextName",
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID:                   "mock-userID",
			GetUserErr:               ErrNotFound,
			GetDisconnectUserPostErr: nil,
			DisconnectUserErr:        nil,
		},
		"DisconnectUserContextName is false": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathUserDisconnect),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						DisconnectUserContextName: false,
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID:                   "mock-userID",
			GetUserErr:               nil,
			GetDisconnectUserPostErr: nil,
			DisconnectUserErr:        nil,
		},
		"Error occur while disconnecting user": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathUserDisconnect),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						DisconnectUserContextName: true,
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID:                   "mock-userID",
			GetUserErr:               nil,
			GetDisconnectUserPostErr: nil,
			DisconnectUserErr:        errors.New("mockError"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := new(Plugin)
			p.setConfiguration(
				&configuration{
					ServiceNowURL:               "mockURL",
					ServiceNowOAuthClientID:     "mockCLientID",
					ServiceNowOAuthClientSecret: "mockClientSecret",
					EncryptionSecret:            "mockEncryptionSecret",
					WebhookSecret:               "mockWebhookSecret",
					MattermostSiteURL:           "mockSiteURL",
					PluginID:                    "mockPluginID",
					PluginURL:                   "mockPluginURL",
					PluginURLPath:               "mockPluginURLPath",
				})

			mockAPI := &plugintest.API{}

			mockAPI.On("GetBundlePath").Return("mockString", nil)

			mockAPI.On("LogDebug", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("Logdebug error")

			mockAPI.On("LogError", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("LogError error")

			p.SetAPI(mockAPI)

			p.initializeAPI()

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "GetUser", func(_ *Plugin, _ string) (*serializer.User, error) {
				return &serializer.User{}, test.GetUserErr
			})

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "GetDisconnectUserPost", func(_ *Plugin, _, _ string) (*model.Post, error) {
				return &model.Post{}, test.GetDisconnectUserPostErr
			})

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "DisconnectUser", func(_ *Plugin, _ string) error {
				return test.DisconnectUserErr
			})

			req := test.httpTest.CreateHTTPRequest(test.request)
			req.Header.Add(HeaderMattermostUserID, test.userID)
			rr := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, rr, req)
			test.httpTest.CompareHTTPResponse(rr, test.expectedResponse)

			if (test.GetUserErr != ErrNotFound && test.GetUserErr != nil) || test.GetDisconnectUserPostErr != nil || test.DisconnectUserErr != nil {
				mockAPI.AssertNumberOfCalls(t, "LogError", 1)
			}
		})
	}
}

func TestPlugin_handleVirtualAgentWebhook(t *testing.T) {
	httpTestJSON := testutils.HTTPTest{
		T:       t,
		Encoder: testutils.EncodeJSON,
	}

	httpTestString := testutils.HTTPTest{
		T:       t,
		Encoder: testutils.EncodeString,
	}

	for name, test := range map[string]struct {
		httpTest         testutils.HTTPTest
		request          testutils.Request
		expectedResponse testutils.ExpectedResponse
		isErrorExpected  bool
	}{
		"Webhook secret is present": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s?secret=mockWebhookSecret", PathVirtualAgentWebhook),
				Body:   VirtualAgentResponse{},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			isErrorExpected: false,
		},
		"Webhook secret is absent": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathVirtualAgentWebhook),
				Body:   VirtualAgentResponse{},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusForbidden,
			},
			isErrorExpected: true,
		},
		"handleVirtualAgentWebhook empty body": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s?secret=mockWebhookSecret", PathVirtualAgentWebhook),
				Body:   nil,
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusInternalServerError,
			},
			isErrorExpected: true,
		},
		"OutputLink response is received from Virtual Agent": {
			httpTest: httpTestString,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s?secret=mockWebhookSecret", PathVirtualAgentWebhook),
				Body: `{
					"requestId": "9ff925c1-893f-4a46-baa5-a64f4d8c89b1",
					"message": {
					  "text": "",
					  "typed": true
					},
					"userId": "6816f79cc0a8016401c5a33be04be441",
					"body": [
					  {
						"uiType": "OutputLink",
						"group": "DefaultText",
						"label": "Successful",
						"header": "header",
						"value": {
							"action": "action"
						}
					  }
					],
					"score": 1
				  }`,
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
		},
		"TopicPickerControl response is received from Virtual Agent": {
			httpTest: httpTestString,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s?secret=mockWebhookSecret", PathVirtualAgentWebhook),
				Body: `{
					"requestId": "9ff925c1-893f-4a46-baa5-a64f4d8c89b1",
					"message": {
					  "text": "",
					  "typed": true
					},
					"userId": "6816f79cc0a8016401c5a33be04be441",
					"body": [
					  { 
						"uiType":"TopicPickerControl", 
						"group":"DefaultPicker", 
						"nluTextEnabled":false, 
						"promptMsg":"Hi guest, please enter your request or make a selection of what I can help with. You can type help any time when you need help.", 
						"label":"Show me everything", 
						"options":[ 
						  { 
							"label":"b2b topic", 
							"value":"2bb7bd7670de6010f877c7f188266fc7", 
							"enabled":true 
						  }, 
						  { 
							 "label":"Live Agent Support.", 
							 "value":"ce2ee85053130010cf8cddeeff7b12bf", 
							 "enabled":true 
						  } 
						] 
					  }
					],
					"score": 1
				  }`,
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
		},
		"OutputText response is received from Virtual Agent": {
			httpTest: httpTestString,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s?secret=mockWebhookSecret", PathVirtualAgentWebhook),
				Body: `{
					"requestId": "9ff925c1-893f-4a46-baa5-a64f4d8c89b1",
					"message": {
					  "text": "",
					  "typed": true
					},
					"userId": "6816f79cc0a8016401c5a33be04be441",
					"body": [
					  {
						"uiType": "OutputText",
						"group": "DefaultText",
						"value": "Successful",
						"maskType": "NONE"
					  }
					],
					"score": 1
				  }`,
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
		},
		"Picker response is received from Virtual Agent": {
			httpTest: httpTestString,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s?secret=mockWebhookSecret", PathVirtualAgentWebhook),
				Body: `{
					"requestId": "9ff925c1-893f-4a46-baa5-a64f4d8c89b1",
					"message": {
					  "text": "",
					  "typed": true
					},
					"userId": "6816f79cc0a8016401c5a33be04be441",
					"body": [
						{
							"uiType":"Picker",
							"group":"DefaultPicker",
							"required":true,
							"nluTextEnabled":false,
							"label":"I want to be sure I got this right. What item best describes what you want to do?",
							"itemType":"List",
							"style":"list",
							"multiSelect":false,
							"options":[
							  {
								"label":"Live Agent Support.",
								"value":"Live Agent Support.",
								"renderStyle":"data",
								"enabled":false
							  },
							  {
								"label":"Virtual Agent Capabilities.",
								"value":"Virtual Agent Capabilities.",
								"renderStyle":"data",
								"enabled":false
							  },
							  {
								"label":"I want something else",
								"value":"-1",
								"renderStyle":"data",
								"enabled":false
							  }
							],
							"scriptedData":null
						  }
					],
					"score": 1
				  }`,
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
		},
		"GroupedPartsOutputControl response is received from Virtual Agent": {
			httpTest: httpTestString,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s?secret=mockWebhookSecret", PathVirtualAgentWebhook),
				Body: `{
					"requestId": "9ff925c1-893f-4a46-baa5-a64f4d8c89b1",
					"message": {
					  "text": "",
					  "typed": true
					},
					"userId": "6816f79cc0a8016401c5a33be04be441",
					"body": [
						{
							"uiType": "GroupedPartsOutputControl",
							"group": "DefaultGroupedPartsOutputControl",
							"groupPartType": "Link",
							"header": "header message",
							"values": [
							  {
								"action": "www.foo",
								"description": "description",
								"label": "link_1 label",
								"context": "ITSM"
							  }
							]
						  }
					],
					"score": 1
				  }`,
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := Plugin{}
			p.setConfiguration(
				&configuration{
					ServiceNowURL:               "mockURL",
					ServiceNowOAuthClientID:     "mockCLientID",
					ServiceNowOAuthClientSecret: "mockClientSecret",
					EncryptionSecret:            "mockEncryptionSecret",
					WebhookSecret:               "mockWebhookSecret",
					MattermostSiteURL:           "mockSiteURL",
					PluginID:                    "mockPluginID",
					PluginURL:                   "mockPluginURL",
					PluginURLPath:               "mockPluginURLPath",
				})

			mockAPI := &plugintest.API{}

			mockAPI.On("GetBundlePath").Return("mockString", nil)

			mockAPI.On("LogDebug", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("Logdebug error")

			mockAPI.On("LogError", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("LogError error")

			mockAPI.On("DM", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil, nil)

			mockAPI.On("DMWithAttachments", mock.AnythingOfType("string"), &model.SlackAttachment{}).Return(nil, nil)

			p.SetAPI(mockAPI)

			p.initializeAPI()

			mockCtrl := gomock.NewController(t)
			mockedStore := mock_plugin.NewMockStore(mockCtrl)

			if !test.isErrorExpected {
				mockedStore.EXPECT().LoadUserWithSysID(gomock.Any()).Return(&serializer.User{}, nil)
			}

			p.store = mockedStore

			req := test.httpTest.CreateHTTPRequest(test.request)
			rr := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, rr, req)
			test.httpTest.CompareHTTPResponse(rr, test.expectedResponse)
		})
	}
}

func TestPlugin_handlePickerSelection(t *testing.T) {
	defer monkey.UnpatchAll()

	httpTestJSON := testutils.HTTPTest{
		T:       t,
		Encoder: testutils.EncodeJSON,
	}

	for name, test := range map[string]struct {
		httpTest              testutils.HTTPTest
		request               testutils.Request
		expectedResponse      testutils.ExpectedResponse
		ParseAuthTokenErr     error
		LoadUserErr           error
		getDirectChannelError *model.AppError
		callError             error
	}{
		"Selected option is successfully sent to virtual Agent": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathActionOptions),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						"selected_option": "mockOption",
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
		},
		"Error while decoding response body": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathActionOptions),
				Body:   nil,
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
		},
		"Failed to get direct channel": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathActionOptions),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						"selected_option": "mockOption",
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			getDirectChannelError: &model.AppError{},
		},
		"User is not present in store": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathActionOptions),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						"selected_option": "mockOption",
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			LoadUserErr: errors.New("mockErr"),
		},
		"Error occurs while parsing OAuth token": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathActionOptions),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						"selected_option": "mockOption",
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			ParseAuthTokenErr: errors.New("mockErr"),
		},
		"Error while sending selected option to Virtual Agent": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathActionOptions),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						"selected_option": "mockOption",
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			callError:         errors.New("mockError"),
			ParseAuthTokenErr: nil,
			LoadUserErr:       nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := new(Plugin)
			p.setConfiguration(
				&configuration{
					ServiceNowURL:               "mockURL",
					ServiceNowOAuthClientID:     "mockCLientID",
					ServiceNowOAuthClientSecret: "mockClientSecret",
					EncryptionSecret:            "mockEncryptionSecret",
					WebhookSecret:               "mockWebhookSecret",
					MattermostSiteURL:           "mockSiteURL",
					PluginID:                    "mockPluginID",
					PluginURL:                   "mockPluginURL",
					PluginURLPath:               "mockPluginURLPath",
				})

			mockAPI := &plugintest.API{}

			mockAPI.On("GetBundlePath").Return("mockString", nil)

			mockAPI.On("LogDebug", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("Logdebug error")

			mockAPI.On("LogError", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("LogError error")

			mockAPI.On("GetDirectChannel", mock.Anything, mock.Anything).Return(&model.Channel{
				Id: "mock-channelID",
			}, test.getDirectChannelError)

			p.SetAPI(mockAPI)

			p.initializeAPI()

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "ParseAuthToken", func(_ *Plugin, _ string) (*oauth2.Token, error) {
				return &oauth2.Token{}, test.ParseAuthTokenErr
			})

			var c client
			monkey.PatchInstanceMethod(reflect.TypeOf(&c), "Call", func(_ *client, _, _, _ string, _ io.Reader, _ interface{}, _ url.Values) (responseData []byte, err error) {
				return nil, test.callError
			})

			mockCtrl := gomock.NewController(t)
			mockedStore := mock_plugin.NewMockStore(mockCtrl)

			mockedStore.EXPECT().LoadUser("mock-userID").Return(&serializer.User{}, test.LoadUserErr)

			p.store = mockedStore

			req := test.httpTest.CreateHTTPRequest(test.request)
			req.Header.Add(HeaderMattermostUserID, "mock-userID")
			rr := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, rr, req)
			test.httpTest.CompareHTTPResponse(rr, test.expectedResponse)

			if test.ParseAuthTokenErr != nil || test.callError != nil || test.LoadUserErr != nil {
				mockAPI.AssertNumberOfCalls(t, "LogError", 1)
			}
		})
	}
}
