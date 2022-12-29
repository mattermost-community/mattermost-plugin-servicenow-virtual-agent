package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/constants"
	mock_plugin "github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/mocks"
	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/serializer"
	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/testutils"
)

type panicHandler struct {
}

func (ph panicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	panic("bad handler")
}

const pathPrefix = "/api/v1"

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
		_, err := io.Copy(io.Discard, resp.Body)
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
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathUserDisconnect),
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
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathUserDisconnect),
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
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathUserDisconnect),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						constants.DisconnectUserContextName: true,
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID: "mock-userID",
		},
		"User not found and failed to create disconnect post": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathUserDisconnect),
				Body:   model.PostActionIntegrationRequest{},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID:                   "mock-userID",
			GetUserErr:               ErrNotFound,
			GetDisconnectUserPostErr: errors.New("failed to create disconnect post"),
		},
		"User is found but error occurred while reading user from KV store": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathUserDisconnect),
				Body:   model.PostActionIntegrationRequest{},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID:                   "mock-userID",
			GetUserErr:               errors.New("error in getting the user from KVstore"),
			GetDisconnectUserPostErr: errors.New("failed to create disconnect post"),
		},
		"User not found and disconnect user post is created successfully": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathUserDisconnect),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						constants.DisconnectUserContextName: "mockContextName",
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
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathUserDisconnect),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						constants.DisconnectUserContextName: false,
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
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathUserDisconnect),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						constants.DisconnectUserContextName: true,
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID:                   "mock-userID",
			GetUserErr:               nil,
			GetDisconnectUserPostErr: nil,
			DisconnectUserErr:        errors.New("error in disconnecting the user"),
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

			mockAPI.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()

			mockAPI.On("LogError", testutils.GetMockArgumentsWithType("string", 6)...).Return()

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
			req.Header.Add(constants.HeaderMattermostUserID, test.userID)
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
				URL:    fmt.Sprintf("%s%s?secret=mockWebhookSecret", pathPrefix, constants.PathVirtualAgentWebhook),
				Body:   VirtualAgentResponse{},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
		},
		"Webhook secret is absent": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathVirtualAgentWebhook),
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
				URL:    fmt.Sprintf("%s%s?secret=mockWebhookSecret", pathPrefix, constants.PathVirtualAgentWebhook),
				Body:   nil,
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusInternalServerError,
			},
			isErrorExpected: true,
		},
		"OutputLink response received from Virtual Agent": {
			httpTest: httpTestString,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("%s%s?secret=mockWebhookSecret", pathPrefix, constants.PathVirtualAgentWebhook),
				Body: `{
					"requestId": "mock-requestId",
					"message": {
					  "text": "",
					  "typed": true
					},
					"userId": "mock-userId",
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
		"TopicPickerControl response received from Virtual Agent": {
			httpTest: httpTestString,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("%s%s?secret=mockWebhookSecret", pathPrefix, constants.PathVirtualAgentWebhook),
				Body: `{
					"requestId": "mock-requestId",
					"message": {
					  "text": "",
					  "typed": true
					},
					"userId": "mock-userId",
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
		"OutputText response received from Virtual Agent": {
			httpTest: httpTestString,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("%s%s?secret=mockWebhookSecret", pathPrefix, constants.PathVirtualAgentWebhook),
				Body: `{
					"requestId": "mock-requestId",
					"message": {
					  "text": "",
					  "typed": true
					},
					"userId": "mock-userId",
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
		"Picker response received from Virtual Agent": {
			httpTest: httpTestString,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("%s%s?secret=mockWebhookSecret", pathPrefix, constants.PathVirtualAgentWebhook),
				Body: `{
					"requestId": "mock-requestId",
					"message": {
					  "text": "",
					  "typed": true
					},
					"userId": "mock-userId",
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
		"GroupedPartsOutputControl response received from Virtual Agent": {
			httpTest: httpTestString,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("%s%s?secret=mockWebhookSecret", pathPrefix, constants.PathVirtualAgentWebhook),
				Body: `{
					"requestId": "mock-requestId",
					"message": {
					  "text": "",
					  "typed": true
					},
					"userId": "mock-userID",
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

			mockAPI.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()

			mockAPI.On("LogError", testutils.GetMockArgumentsWithType("string", 7)...).Return()

			mockAPI.On("DM", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil, nil)

			mockAPI.On("DMWithAttachments", mock.AnythingOfType("string"), &model.SlackAttachment{}).Return(nil, nil)

			p.SetAPI(mockAPI)

			p.initializeAPI()

			mockCtrl := gomock.NewController(t)
			mockedStore := mock_plugin.NewMockStore(mockCtrl)

			if !test.isErrorExpected {
				mockedStore.EXPECT().LoadUserWithSysID(gomock.Any()).Return(&serializer.User{}, nil)
				mockedStore.EXPECT().LoadPostIDs(gomock.Any()).Return([]string{}, nil)
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
		httpTest               testutils.HTTPTest
		request                testutils.Request
		isCarousel             bool
		expectedResponse       testutils.ExpectedResponse
		ParseAuthTokenErr      error
		LoadUserErr            error
		getDirectChannelError  *model.AppError
		callError              error
		getPostError           *model.AppError
		loadPostIDsReturnValue []string
		loadPostIDsError       error
		storePostIDsError      error
	}{
		"Picker is not carousel type and selected option is successfully sent to virtual Agent": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathActionOptions),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						constants.ContextKeySelectedOption: "mockOption",
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
		},
		"Picker is carousel type and selected option is successfully sent to virtual Agent": {
			httpTest:   httpTestJSON,
			isCarousel: true,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathActionOptions),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						constants.ContextKeySelectedLabel: "mockLabel",
						constants.ContextKeySelectedValue: "mockValue",
						constants.StyleCarousel:           true,
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			loadPostIDsReturnValue: []string{},
		},
		"Error while decoding response body": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathActionOptions),
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
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathActionOptions),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						constants.ContextKeySelectedOption: "mockOption",
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
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathActionOptions),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						constants.ContextKeySelectedOption: "mockOption",
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			LoadUserErr: errors.New("error in loading the user from KVstore"),
		},
		"Error occurs while parsing OAuth token": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathActionOptions),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						constants.ContextKeySelectedOption: "mockOption",
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
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathActionOptions),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						constants.ContextKeySelectedOption: "mockOption",
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
		"Picker is carousel type and error occurs while getting post": {
			httpTest:   httpTestJSON,
			isCarousel: true,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathActionOptions),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						constants.ContextKeySelectedLabel: "mockLabel",
						constants.ContextKeySelectedValue: "mockValue",
						constants.StyleCarousel:           true,
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			getPostError:           testutils.GetAppError("error while getting post"),
			loadPostIDsReturnValue: []string{},
		},
		"Picker is carousel type and error occurs while loading postIDs": {
			httpTest:   httpTestJSON,
			isCarousel: true,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathActionOptions),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						constants.ContextKeySelectedLabel: "mockLabel",
						constants.ContextKeySelectedValue: "mockValue",
						constants.StyleCarousel:           true,
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			loadPostIDsError: errors.New("error in loading post IDs"),
		},
		"Picker is carousel type and error occurs while storing postIDs": {
			httpTest:   httpTestJSON,
			isCarousel: true,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathActionOptions),
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						constants.ContextKeySelectedLabel: "mockLabel",
						constants.ContextKeySelectedValue: "mockValue",
						constants.StyleCarousel:           true,
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			loadPostIDsReturnValue: []string{testutils.GetID()},
			storePostIDsError:      errors.New("error in storing post IDs"),
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

			mockAPI.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()

			mockAPI.On("LogError", testutils.GetMockArgumentsWithType("string", 7)...).Return()

			mockAPI.On("GetDirectChannel", mock.Anything, mock.Anything).Return(&model.Channel{
				Id: "mock-channelID",
			}, test.getDirectChannelError)

			p.SetAPI(mockAPI)

			p.initializeAPI()

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "ParseAuthToken", func(_ *Plugin, _ string) (*oauth2.Token, error) {
				return &oauth2.Token{}, test.ParseAuthTokenErr
			})
			monkey.PatchInstanceMethod(reflect.TypeOf(p), "ScheduleJob", func(_ *Plugin, _ string) error {
				return nil
			})

			var c client
			monkey.PatchInstanceMethod(reflect.TypeOf(&c), "Call", func(_ *client, _, _, _ string, _ io.Reader, _ interface{}, _ url.Values) (responseData []byte, err error) {
				return nil, test.callError
			})

			mockCtrl := gomock.NewController(t)
			mockedStore := mock_plugin.NewMockStore(mockCtrl)

			mockedStore.EXPECT().LoadUser("mock-userID").Return(&serializer.User{}, test.LoadUserErr)
			if test.isCarousel {
				mockAPI.On("GetPost", mock.AnythingOfType("string")).Return(testutils.GetPostWithAttachments(2), test.getPostError)
				mockedStore.EXPECT().LoadPostIDs("").Return(test.loadPostIDsReturnValue, test.loadPostIDsError)
				if test.loadPostIDsReturnValue != nil && len(test.loadPostIDsReturnValue) > 0 {
					mockedStore.EXPECT().StorePostIDs("", []string{}).Return(test.storePostIDsError)
					mockAPI.On("DeletePost", mock.AnythingOfType("string")).Return(testutils.GetAppError("error in deleting the post"))
				}
			}
			p.store = mockedStore

			req := test.httpTest.CreateHTTPRequest(test.request)
			req.Header.Add(constants.HeaderMattermostUserID, "mock-userID")
			rr := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, rr, req)
			test.httpTest.CompareHTTPResponse(rr, test.expectedResponse)

			if test.ParseAuthTokenErr != nil || test.callError != nil || test.LoadUserErr != nil {
				mockAPI.AssertNumberOfCalls(t, "LogError", 1)
			}
		})
	}
}

func getHandleDateTimeSelectionRequestBody(date, time, dialogType string) *model.SubmitDialogRequest {
	return &model.SubmitDialogRequest{
		CallbackId: fmt.Sprintf("mockPostID__%s", dialogType),
		Submission: map[string]interface{}{
			"date": date,
			"time": time,
		},
		ChannelId: "mockChannelID",
	}
}

func Test_handleDateTimeSelection(t *testing.T) {
	defer monkey.UnpatchAll()

	httpTestJSON := testutils.HTTPTest{
		T:       t,
		Encoder: testutils.EncodeJSON,
	}

	for name, test := range map[string]struct {
		httpTest          testutils.HTTPTest
		request           testutils.Request
		expectedResponse  testutils.ExpectedResponse
		userID            string
		ParseAuthTokenErr error
	}{
		"User is unauthorized": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", constants.PathSetDateTime),
				Body:   getHandleDateTimeSelectionRequestBody("2022-09-23", "", "Date"),
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusUnauthorized,
			},
		},
		"Error parsing OAuth token": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", constants.PathSetDateTime),
				Body:   getHandleDateTimeSelectionRequestBody("2022-09-23", "", "Date"),
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID:            "mock-userID",
			ParseAuthTokenErr: errors.New("mockError"),
		},
		"Selected date is successfully sent to virtual Agent": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", constants.PathSetDateTime),
				Body:   getHandleDateTimeSelectionRequestBody("2022-09-23", "", "Date"),
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode:   http.StatusOK,
				Body:         &model.SubmitDialogResponse{},
				ResponseType: "application/json",
			},
			userID: "mock-userID",
		},
		"Selected date is invalid": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", constants.PathSetDateTime),
				Body:   getHandleDateTimeSelectionRequestBody("2022-23-23", "", "Date"),
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
				Body: &model.SubmitDialogResponse{
					Errors: map[string]string{
						"date": "Please enter a valid date",
					},
				},
				ResponseType: "application/json",
			},
			userID: "mock-userID",
		},
		"Selected time is successfully sent to virtual Agent": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", constants.PathSetDateTime),
				Body:   getHandleDateTimeSelectionRequestBody("", "22:12", "Time"),
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode:   http.StatusOK,
				Body:         &model.SubmitDialogResponse{},
				ResponseType: "application/json",
			},
			userID: "mock-userID",
		},
		"Selected time is invalid": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", constants.PathSetDateTime),
				Body:   getHandleDateTimeSelectionRequestBody("", "25:12", "Time"),
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
				Body: &model.SubmitDialogResponse{
					Errors: map[string]string{
						"time": "Please enter a valid time",
					},
				},
				ResponseType: "application/json",
			},
			userID: "mock-userID",
		},
		"Selected date-time is successfully sent to virtual Agent": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", constants.PathSetDateTime),
				Body:   getHandleDateTimeSelectionRequestBody("2022-09-23", "22:12", "DateTime"),
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
				Body: &model.SubmitDialogResponse{
					Errors: map[string]string{},
				},
				ResponseType: "application/json",
			},
			userID: "mock-userID",
		},
		"Selected date-time is invalid": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", constants.PathSetDateTime),
				Body:   getHandleDateTimeSelectionRequestBody("2022-13-23", "24:12", "DateTime"),
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
				Body: &model.SubmitDialogResponse{
					Errors: map[string]string{
						"date": "Please enter a valid date",
						"time": "Please enter a valid time",
					},
				},
				ResponseType: "application/json",
			},
			userID: "mock-userID",
		},
	} {
		t.Run(name, func(t *testing.T) {
			mockInterval := int64(1000)
			p := new(Plugin)

			mockAPI := &plugintest.API{}
			mockAPI.On("GetBundlePath").Return("mockString", nil)
			mockAPI.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return("LogDebug error")
			mockAPI.On("LogError", testutils.GetMockArgumentsWithType("string", 5)...).Return()
			mockAPI.On("UpdatePost", mock.AnythingOfType("*model.Post")).Return(nil, nil)

			mockAPI.On("GetConfig").Return(&model.Config{
				ServiceSettings: model.ServiceSettings{
					TimeBetweenUserTypingUpdatesMilliseconds: &mockInterval,
				},
			})

			p.SetAPI(mockAPI)

			p.initializeAPI()

			var c client
			monkey.PatchInstanceMethod(reflect.TypeOf(&c), "SendMessageToVirtualAgentAPI", func(_ *client, _, _ string, _ bool, _ *serializer.MessageAttachment) error {
				return nil
			})

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "ParseAuthToken", func(_ *Plugin, _ string) (*oauth2.Token, error) {
				return &oauth2.Token{}, test.ParseAuthTokenErr
			})

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "ScheduleJob", func(_ *Plugin, _ string) error {
				return nil
			})

			if test.userID != "" {
				mockCtrl := gomock.NewController(t)
				mockedStore := mock_plugin.NewMockStore(mockCtrl)

				mockedStore.EXPECT().LoadUser(test.userID).Return(&serializer.User{}, nil)

				p.store = mockedStore
			}

			req := test.httpTest.CreateHTTPRequest(test.request)
			req.Header.Add(constants.HeaderMattermostUserID, test.userID)
			resp := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, resp, req)
			test.httpTest.CompareHTTPResponse(resp, test.expectedResponse)
		})
	}
}

func Test_handleDateTimeSelectionDialog(t *testing.T) {
	defer monkey.UnpatchAll()

	httpTestJSON := testutils.HTTPTest{
		T:       t,
		Encoder: testutils.EncodeJSON,
	}

	for name, test := range map[string]struct {
		httpTest             testutils.HTTPTest
		request              testutils.Request
		expectedResponse     testutils.ExpectedResponse
		userID               string
		parseAuthTokenErr    error
		openDialogRequestErr error
	}{
		"User is unauthorized": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", constants.PathSetDateTimeDialog),
				Body: model.PostActionIntegrationRequest{
					TriggerId: "mockTriggerId",
					PostId:    "mockPostId",
					Context: map[string]interface{}{
						"type": "Date",
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusUnauthorized,
			},
		},
		"Selected date is successfully sent to virtual Agent": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", constants.PathSetDateTimeDialog),
				Body: model.PostActionIntegrationRequest{
					TriggerId: "mockTriggerId",
					PostId:    "mockPostId",
					Context: map[string]interface{}{
						"type": "Date",
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode:   http.StatusOK,
				Body:         &model.PostActionIntegrationResponse{},
				ResponseType: "application/json",
			},
			userID: "mock-userID",
		},
		"Error in opening data/time selection dialog": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", constants.PathSetDateTimeDialog),
				Body: model.PostActionIntegrationRequest{
					TriggerId: "mockTriggerId",
					PostId:    "mockPostId",
					Context: map[string]interface{}{
						"type": "Date",
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusInternalServerError,
				Body: &serializer.APIErrorResponse{
					StatusCode: http.StatusInternalServerError,
					Message:    "Error in opening date-time selection dialog.",
				},
				ResponseType: "application/json",
			},
			userID:               "mock-userID",
			openDialogRequestErr: errors.New("request failed to open date-/time selction dialog"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := new(Plugin)

			mockAPI := &plugintest.API{}
			mockAPI.On("GetBundlePath").Return("mockString", nil)
			mockAPI.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return("LogDebug error")
			mockAPI.On("LogError", testutils.GetMockArgumentsWithType("string", 6)...).Return("LogError error")
			p.SetAPI(mockAPI)

			p.initializeAPI()

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "ParseAuthToken", func(_ *Plugin, _ string) (*oauth2.Token, error) {
				return &oauth2.Token{}, test.parseAuthTokenErr
			})

			c := client{}
			monkey.PatchInstanceMethod(reflect.TypeOf(&c), "OpenDialogRequest", func(_ *client, _ *model.OpenDialogRequest) error {
				return test.openDialogRequestErr
			})

			if test.userID != "" {
				mockCtrl := gomock.NewController(t)
				mockedStore := mock_plugin.NewMockStore(mockCtrl)

				mockedStore.EXPECT().LoadUser(test.userID).Return(&serializer.User{}, nil)

				p.store = mockedStore
			}

			req := test.httpTest.CreateHTTPRequest(test.request)
			req.Header.Add(constants.HeaderMattermostUserID, test.userID)
			resp := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, resp, req)
			test.httpTest.CompareHTTPResponse(resp, test.expectedResponse)
		})
	}
}

func TestPlugin_handleFileAttachments(t *testing.T) {
	defer monkey.UnpatchAll()

	httpTestJSON := testutils.HTTPTest{
		T:       t,
		Encoder: testutils.EncodeJSON,
	}

	for name, test := range map[string]struct {
		httpTest         testutils.HTTPTest
		request          testutils.Request
		expectedResponse testutils.ExpectedResponse
		decodeError      error
		decryptError     error
		unmarshalError   error
		getFileError     *model.AppError
		isErrorExpected  bool
		isExpired        bool
	}{
		"File data is written in response": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    fmt.Sprintf("%s/file/{%s}", pathPrefix, constants.PathParamEncryptedFileInfo),
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
		},
		"Error decoding encrypted file info": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    fmt.Sprintf("%s/file/{%s}", pathPrefix, constants.PathParamEncryptedFileInfo),
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusBadRequest,
				Body: serializer.APIErrorResponse{
					Message:    "Error occurred while decoding the file.",
					StatusCode: http.StatusBadRequest,
				},
				ResponseType: "application/json",
			},
			decodeError:     errors.New("error in decoding the file"),
			isErrorExpected: true,
		},
		"Error decrypting file info": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    fmt.Sprintf("%s/file/{%s}", pathPrefix, constants.PathParamEncryptedFileInfo),
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusInternalServerError,
				Body: serializer.APIErrorResponse{
					Message:    "Error occurred while decrypting the file.",
					StatusCode: http.StatusBadRequest,
				},
				ResponseType: "application/json",
			},
			decryptError:    errors.New("error in decrypting the file"),
			isErrorExpected: true,
		},
		"Error unmarshaling file info": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    fmt.Sprintf("%s/file/{%s}", pathPrefix, constants.PathParamEncryptedFileInfo),
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusInternalServerError,
				Body: serializer.APIErrorResponse{
					Message:    "Error occurred while unmarshaling the file.",
					StatusCode: http.StatusBadRequest,
				},
				ResponseType: "application/json",
			},
			unmarshalError:  errors.New("error in unmarshaling the file"),
			isErrorExpected: true,
		},
		"Error getting file data": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    fmt.Sprintf("%s/file/{%s}", pathPrefix, constants.PathParamEncryptedFileInfo),
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusInternalServerError,
				Body: serializer.APIErrorResponse{
					Message:    "Couldn't get the file data.",
					StatusCode: http.StatusBadRequest,
				},
				ResponseType: "application/json",
			},
			getFileError: &model.AppError{
				Message: "error in getting file data",
			},
			isErrorExpected: true,
		},
		"File link is expired": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    fmt.Sprintf("%s/file/{%s}", pathPrefix, constants.PathParamEncryptedFileInfo),
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusNotFound,
			},
			isExpired: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := new(Plugin)
			p.setConfiguration(&configuration{EncryptionSecret: "mockEncryptionSecret"})

			mockAPI := &plugintest.API{}
			mockAPI.On("GetBundlePath").Return("mockString", nil)
			mockAPI.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
			mockAPI.On("LogError", testutils.GetMockArgumentsWithType("string", 6)...).Return()
			mockAPI.On("GetFile", mock.AnythingOfType("string")).Return([]byte{}, test.getFileError)
			p.SetAPI(mockAPI)

			p.initializeAPI()

			monkey.Patch(decode, func(_ string) ([]byte, error) {
				return []byte{}, test.decodeError
			})
			monkey.Patch(decrypt, func(_, _ []byte) ([]byte, error) {
				return []byte{}, test.decryptError
			})
			monkey.Patch(json.Unmarshal, func(_ []byte, _ interface{}) error {
				return test.unmarshalError
			})

			currentTime := time.Now().UTC()
			monkey.PatchInstanceMethod(reflect.TypeOf(currentTime), "After", func(_ time.Time, _ time.Time) bool {
				return test.isExpired
			})

			req := test.httpTest.CreateHTTPRequest(test.request)
			req.Header.Add(constants.HeaderMattermostUserID, "mock-userID")
			resp := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, resp, req)
			test.httpTest.CompareHTTPResponse(resp, test.expectedResponse)

			if test.isErrorExpected {
				mockAPI.AssertNumberOfCalls(t, "LogError", 1)
			}
		})
	}
}
