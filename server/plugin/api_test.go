package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/stretchr/testify/assert"
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

func setupTestPlugin(api *plugintest.API, store *mock_plugin.MockStore) (*Plugin, *plugintest.API) {
	p := &Plugin{}
	path, _ := filepath.Abs("../..")
	api.On("GetBundlePath").Return(path, nil)
	p.SetAPI(api)
	if store != nil {
		p.store = store
	}

	p.router = p.initializeAPI()
	p.setConfiguration(&configuration{
		ServiceNowURL:               "mockURL",
		ServiceNowOAuthClientID:     "mockClientID",
		ServiceNowOAuthClientSecret: "mockClientSecret",
		EncryptionSecret:            "mockEncryptionSecret",
		WebhookSecret:               "mockWebhookSecret",
		MattermostSiteURL:           "mockSiteURL",
		PluginID:                    "mockPluginID",
		PluginURL:                   "mockPluginURL",
		PluginURLPath:               "mockPluginURLPath",
	})

	return p, api
}

func TestWithRecovery(t *testing.T) {
	defer func() {
		if x := recover(); x != nil {
			require.Fail(t, "got panic")
		}
	}()

	p, api := setupTestPlugin(&plugintest.API{}, nil)
	api.On("LogError", "Recovered from a panic", "URL", "http://random", "Error", "bad handler", "Stack", mock.Anything)

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

func setupPluginForCheckOAuthMiddleware(p *Plugin, s *mock_plugin.MockStore, c *mock_plugin.MockClient, t *testing.T) {
	s.EXPECT().LoadUser(testutils.GetID()).Return(testutils.GetSerializerUser(), nil)
	monkey.PatchInstanceMethod(reflect.TypeOf(p), "ParseAuthToken", func(_ *Plugin, _ string) (*oauth2.Token, error) {
		return nil, nil
	})

	monkey.PatchInstanceMethod(reflect.TypeOf(p), "GetClientFromRequest", func(_ *Plugin, _ *http.Request) Client {
		return c
	})
}

func TestPlugin_handleSkip(t *testing.T) {
	httpTestJSON := testutils.HTTPTest{
		T:       t,
		Encoder: testutils.EncodeJSON,
	}

	for name, test := range map[string]struct {
		httpTest         testutils.HTTPTest
		request          testutils.Request
		expectedResponse testutils.ExpectedResponse
		setupClient      func(c *mock_plugin.MockClient)
	}{
		"Skip message is successfully sent to virtual Agent": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathSkip),
				Body: model.PostActionIntegrationRequest{
					ChannelId: testutils.GetID(),
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			setupClient: func(c *mock_plugin.MockClient) {
				c.EXPECT().SendMessageToVirtualAgentAPI(testutils.GetServiceNowSysID(), constants.SkipInternal, true, &serializer.MessageAttachment{}).Return(nil)
			},
		},
		"Error while decoding response body": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathSkip),
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			setupClient: func(c *mock_plugin.MockClient) {},
		},
		"Error while sending skip message to Virtual Agent": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathSkip),
				Body: model.PostActionIntegrationRequest{
					ChannelId: testutils.GetID(),
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			setupClient: func(c *mock_plugin.MockClient) {
				c.EXPECT().SendMessageToVirtualAgentAPI(testutils.GetServiceNowSysID(), constants.SkipInternal, true, &serializer.MessageAttachment{}).Return(errors.New("error while sending skip message to Virtual Agent"))
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			mockedStore := mock_plugin.NewMockStore(mockCtrl)
			mockedClient := mock_plugin.NewMockClient(mockCtrl)

			p, mockAPI := setupTestPlugin(&plugintest.API{}, mockedStore)

			mockAPI.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
			mockAPI.On("LogError", testutils.GetMockArgumentsWithType("string", 7)...).Return()

			setupPluginForCheckOAuthMiddleware(p, mockedStore, mockedClient, t)
			test.setupClient(mockedClient)

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "ScheduleJob", func(_ *Plugin, _ string) error {
				return nil
			})

			req := test.httpTest.CreateHTTPRequest(test.request)
			req.Header.Add(constants.HeaderMattermostUserID, testutils.GetID())
			rr := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, rr, req)
			test.httpTest.CompareHTTPResponse(rr, test.expectedResponse)
		})
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
		GetUserErr               error
		GetDisconnectUserPostErr error
		DisconnectUserErr        error
	}{
		"Error while decoding request body": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("%s%s", pathPrefix, constants.PathUserDisconnect),
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
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
			GetUserErr: ErrNotFound,
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
		},
		"Error occurred while disconnecting user": {
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
			GetUserErr:               nil,
			GetDisconnectUserPostErr: nil,
			DisconnectUserErr:        errors.New("error in disconnecting the user"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			p, mockAPI := setupTestPlugin(&plugintest.API{}, nil)
			mockAPI.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
			mockAPI.On("LogError", testutils.GetMockArgumentsWithType("string", 6)...).Return()

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
			req.Header.Add(constants.HeaderMattermostUserID, testutils.GetID())
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
			mockCtrl := gomock.NewController(t)
			mockedStore := mock_plugin.NewMockStore(mockCtrl)
			p, mockAPI := setupTestPlugin(&plugintest.API{}, mockedStore)

			mockAPI.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()

			mockAPI.On("LogError", testutils.GetMockArgumentsWithType("string", 7)...).Return()

			mockAPI.On("DM", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil, nil)

			mockAPI.On("DMWithAttachments", mock.AnythingOfType("string"), &model.SlackAttachment{}).Return(nil, nil)

			if !test.isErrorExpected {
				mockedStore.EXPECT().LoadUserWithSysID(gomock.Any()).Return(&serializer.User{}, nil)
				mockedStore.EXPECT().LoadPostIDs(gomock.Any()).Return([]string{}, nil)
			}

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
		getPostError           *model.AppError
		loadPostIDsReturnValue []string
		loadPostIDsError       error
		storePostIDsError      error
		setupClient            func(c *mock_plugin.MockClient)
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
			setupClient: func(c *mock_plugin.MockClient) {
				c.EXPECT().SendMessageToVirtualAgentAPI(testutils.GetServiceNowSysID(), "mockOption", true, &serializer.MessageAttachment{}).Return(nil)
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
			setupClient: func(c *mock_plugin.MockClient) {
				c.EXPECT().SendMessageToVirtualAgentAPI(testutils.GetServiceNowSysID(), "mockValue", false, &serializer.MessageAttachment{}).Return(nil)
			},
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
			setupClient: func(c *mock_plugin.MockClient) {},
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
			setupClient: func(c *mock_plugin.MockClient) {
				c.EXPECT().SendMessageToVirtualAgentAPI(testutils.GetServiceNowSysID(), "mockOption", true, &serializer.MessageAttachment{}).Return(errors.New("error in sending message to VA"))
			},
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
			setupClient:            func(c *mock_plugin.MockClient) {},
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
			setupClient: func(c *mock_plugin.MockClient) {
				c.EXPECT().SendMessageToVirtualAgentAPI(testutils.GetServiceNowSysID(), "mockValue", false, &serializer.MessageAttachment{}).Return(nil)
			},
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
			setupClient: func(c *mock_plugin.MockClient) {
				c.EXPECT().SendMessageToVirtualAgentAPI(testutils.GetServiceNowSysID(), "mockValue", false, &serializer.MessageAttachment{}).Return(nil)
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			mockedStore := mock_plugin.NewMockStore(mockCtrl)
			mockedClient := mock_plugin.NewMockClient(mockCtrl)
			p, mockAPI := setupTestPlugin(&plugintest.API{}, mockedStore)

			mockAPI.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
			mockAPI.On("LogError", testutils.GetMockArgumentsWithType("string", 7)...).Return()

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "ScheduleJob", func(_ *Plugin, _ string) error {
				return nil
			})

			if test.isCarousel {
				mockAPI.On("GetPost", mock.AnythingOfType("string")).Return(testutils.GetPostWithAttachments(2), test.getPostError)
				mockedStore.EXPECT().LoadPostIDs(testutils.GetID()).Return(test.loadPostIDsReturnValue, test.loadPostIDsError)
				if test.loadPostIDsReturnValue != nil && len(test.loadPostIDsReturnValue) > 0 {
					mockedStore.EXPECT().StorePostIDs(testutils.GetID(), []string{}).Return(test.storePostIDsError)
					mockAPI.On("DeletePost", mock.AnythingOfType("string")).Return(testutils.GetAppError("error in deleting the post"))
				}
			}

			setupPluginForCheckOAuthMiddleware(p, mockedStore, mockedClient, t)
			test.setupClient(mockedClient)

			req := test.httpTest.CreateHTTPRequest(test.request)
			req.Header.Add(constants.HeaderMattermostUserID, testutils.GetID())
			rr := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, rr, req)
			test.httpTest.CompareHTTPResponse(rr, test.expectedResponse)
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
		httpTest         testutils.HTTPTest
		request          testutils.Request
		expectedResponse testutils.ExpectedResponse
		setupClient      func(c *mock_plugin.MockClient)
	}{
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
			setupClient: func(c *mock_plugin.MockClient) {
				c.EXPECT().SendMessageToVirtualAgentAPI(testutils.GetServiceNowSysID(), "2022-09-23", true, &serializer.MessageAttachment{}).Return(nil)
			},
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
			setupClient: func(c *mock_plugin.MockClient) {},
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
			setupClient: func(c *mock_plugin.MockClient) {
				c.EXPECT().SendMessageToVirtualAgentAPI(testutils.GetServiceNowSysID(), "22:12:00", true, &serializer.MessageAttachment{}).Return(nil)
			},
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
			setupClient: func(c *mock_plugin.MockClient) {},
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
			setupClient: func(c *mock_plugin.MockClient) {
				c.EXPECT().SendMessageToVirtualAgentAPI(testutils.GetServiceNowSysID(), gomock.Any(), true, &serializer.MessageAttachment{}).Return(nil)
			},
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
			setupClient: func(c *mock_plugin.MockClient) {},
		},
	} {
		t.Run(name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			mockedStore := mock_plugin.NewMockStore(mockCtrl)
			mockedClient := mock_plugin.NewMockClient(mockCtrl)
			p, mockAPI := setupTestPlugin(&plugintest.API{}, mockedStore)

			mockAPI.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return("LogDebug error")
			mockAPI.On("LogError", testutils.GetMockArgumentsWithType("string", 5)...).Return()
			mockAPI.On("UpdatePost", mock.AnythingOfType("*model.Post")).Return(nil, nil)

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "ScheduleJob", func(_ *Plugin, _ string) error {
				return nil
			})

			setupPluginForCheckOAuthMiddleware(p, mockedStore, mockedClient, t)
			test.setupClient(mockedClient)

			req := test.httpTest.CreateHTTPRequest(test.request)
			req.Header.Add(constants.HeaderMattermostUserID, testutils.GetID())
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
		openDialogRequestErr error
	}{
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
		},
		"Error in opening date/time selection dialog": {
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
			openDialogRequestErr: errors.New("request failed to open date-/time selction dialog"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			mockedStore := mock_plugin.NewMockStore(mockCtrl)
			mockedClient := mock_plugin.NewMockClient(mockCtrl)
			p, mockAPI := setupTestPlugin(&plugintest.API{}, mockedStore)

			mockAPI.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return("LogDebug error")
			mockAPI.On("LogError", testutils.GetMockArgumentsWithType("string", 6)...).Return("LogError error")

			setupPluginForCheckOAuthMiddleware(p, mockedStore, mockedClient, t)
			mockedClient.EXPECT().OpenDialogRequest(gomock.Any()).Return(test.openDialogRequestErr)

			req := test.httpTest.CreateHTTPRequest(test.request)
			req.Header.Add(constants.HeaderMattermostUserID, testutils.GetID())
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
			p, mockAPI := setupTestPlugin(&plugintest.API{}, nil)
			mockAPI.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
			mockAPI.On("LogError", testutils.GetMockArgumentsWithType("string", 6)...).Return()
			mockAPI.On("GetFile", mock.AnythingOfType("string")).Return([]byte{}, test.getFileError)

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
			req.Header.Add(constants.HeaderMattermostUserID, testutils.GetID())
			resp := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, resp, req)
			test.httpTest.CompareHTTPResponse(resp, test.expectedResponse)

			if test.isErrorExpected {
				mockAPI.AssertNumberOfCalls(t, "LogError", 1)
			}
		})
	}
}

func TestCheckAuth(t *testing.T) {
	requestURL := fmt.Sprintf("%s%s", pathPrefix, constants.PathOAuth2Connect)
	t.Run("user id not present", func(t *testing.T) {
		assert := assert.New(t)
		p, mockAPI := setupTestPlugin(&plugintest.API{}, nil)
		mockAPI.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, requestURL, nil)
		p.ServeHTTP(nil, w, r)

		result := w.Result()
		require.NotNil(t, result)
		defer result.Body.Close()

		assert.Equal(http.StatusUnauthorized, result.StatusCode)
		var resp *serializer.APIErrorResponse
		err := json.NewDecoder(result.Body).Decode(&resp)
		require.Nil(t, err)
		assert.Contains(resp.Message, constants.ErrorNotAuthorized)
	})
}

func TestCheckOAuth(t *testing.T) {
	requestURL := fmt.Sprintf("%s%s", pathPrefix, constants.PathSetDateTimeDialog)
	for name, test := range map[string]struct {
		SetupAPI             func(*plugintest.API)
		SetupPluginAndStore  func(p *Plugin, s *mock_plugin.MockStore)
		ExpectedStatusCode   int
		ExpectedErrorMessage string
	}{
		"failed to load the user": {
			SetupAPI: func(api *plugintest.API) {
				api.On("LogError", mock.AnythingOfType("string"), "Error", "load user error")
			},
			SetupPluginAndStore: func(p *Plugin, s *mock_plugin.MockStore) {
				s.EXPECT().LoadUser(testutils.GetID()).Return(nil, errors.New("load user error"))
			},
			ExpectedStatusCode:   http.StatusInternalServerError,
			ExpectedErrorMessage: "load user error",
		},
		"failed to parse auth token": {
			SetupAPI: func(api *plugintest.API) {
				api.On("LogError", mock.AnythingOfType("string"), "Error", "token error")
			},
			SetupPluginAndStore: func(p *Plugin, s *mock_plugin.MockStore) {
				s.EXPECT().LoadUser(testutils.GetID()).Return(testutils.GetSerializerUser(), nil)
				monkey.PatchInstanceMethod(reflect.TypeOf(p), "ParseAuthToken", func(_ *Plugin, _ string) (*oauth2.Token, error) {
					return nil, fmt.Errorf("token error")
				})
			},
			ExpectedStatusCode:   http.StatusInternalServerError,
			ExpectedErrorMessage: "token error",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			defer monkey.UnpatchAll()

			mockCtrl := gomock.NewController(t)
			mockedStore := mock_plugin.NewMockStore(mockCtrl)
			p, api := setupTestPlugin(&plugintest.API{}, mockedStore)
			api.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
			test.SetupAPI(api)
			test.SetupPluginAndStore(p, mockedStore)
			defer api.AssertExpectations(t)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, requestURL, nil)
			r.Header.Add(constants.HeaderMattermostUserID, testutils.GetID())
			p.ServeHTTP(nil, w, r)

			result := w.Result()
			require.NotNil(t, result)
			defer result.Body.Close()

			assert.Equal(test.ExpectedStatusCode, result.StatusCode)
			if test.ExpectedErrorMessage != "" {
				var resp *serializer.APIErrorResponse
				err := json.NewDecoder(result.Body).Decode(&resp)
				require.Nil(t, err)
				assert.Contains(resp.Message, test.ExpectedErrorMessage)
			}
		})
	}
}
