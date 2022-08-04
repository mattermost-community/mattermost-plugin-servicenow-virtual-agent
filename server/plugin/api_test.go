package plugin

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"bou.ke/monkey"
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
		"Everything works fine": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    "/api/v1/user/disconnect",
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						DisconnectUserContextName: true,
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID:                   "mockUserID",
			GetUserErr:               nil,
			GetDisconnectUserPostErr: nil,
			DisconnectUserErr:        nil,
		},
		"When user not found and failed to create disconnect post": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    "/api/v1/user/disconnect",
				Body:   model.PostActionIntegrationRequest{},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID:                   "mockUserID",
			GetUserErr:               ErrNotFound,
			GetDisconnectUserPostErr: errors.New("mockErr"),
			DisconnectUserErr:        nil,
		},
		"When user is found but error occured while reading user from KV store": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    "/api/v1/user/disconnect",
				Body:   model.PostActionIntegrationRequest{},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID:                   "mockUserID",
			GetUserErr:               errors.New("mockError"),
			GetDisconnectUserPostErr: errors.New("mockError"),
			DisconnectUserErr:        nil,
		},
		"When user not found and disconnect user post is created successfully": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    "/api/v1/user/disconnect",
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						DisconnectUserContextName: "mockContextName",
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID:                   "mockUserID",
			GetUserErr:               ErrNotFound,
			GetDisconnectUserPostErr: nil,
			DisconnectUserErr:        nil,
		},
		"When DisconnectUserContextName is false": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    "/api/v1/user/disconnect",
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						DisconnectUserContextName: false,
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID:                   "mockUserID",
			GetUserErr:               nil,
			GetDisconnectUserPostErr: nil,
			DisconnectUserErr:        nil,
		},
		"When error occur while disconnecting user": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    "/api/v1/user/disconnect",
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						DisconnectUserContextName: true,
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID:                   "mockUserID",
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

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "GetUser", func(_ *Plugin, _ string) (*User, error) {
				return &User{}, test.GetUserErr
			})

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "GetDisconnectUserPost", func(_ *Plugin, _, _ string) (*model.Post, error) {
				return &model.Post{}, test.GetDisconnectUserPostErr
			})

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "DisconnectUser", func(_ *Plugin, _ string) error {
				return test.DisconnectUserErr
			})

			req := test.httpTest.CreateHTTPRequest(test.request)
			req.Header.Add("Mattermost-User-ID", test.userID)
			rr := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, rr, req)
			test.httpTest.CompareHTTPResponse(rr, test.expectedResponse)
		})
	}
}

func TestPlugin_handleVirtualAgentWebhook(t *testing.T) {
	httpTestJSON := testutils.HTTPTest{
		T:       t,
		Encoder: testutils.EncodeJSON,
	}

	for name, test := range map[string]struct {
		httpTest         testutils.HTTPTest
		request          testutils.Request
		expectedResponse testutils.ExpectedResponse
		userID           string
	}{
		"When webhook secret is absent": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    "/api/v1/nowbot/processResponse",
				Body:   VirtualAgentResponse{},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusForbidden,
			},
			userID: "",
		},
		"When webhook secret is present": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    "/api/v1/nowbot/processResponse?secret=mockWebhookSecret",
				Body:   VirtualAgentResponse{},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID: "",
		},
		"When body is empty": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    "/api/v1/nowbot/processResponse?secret=mockWebhookSecret",
				Body:   nil,
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusInternalServerError,
			},
			userID: "",
		},
		"When body is proper": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    "/api/v1/nowbot/processResponse?secret=mockWebhookSecret",
				Body: VirtualAgentResponse{
					VirtualAgentRequestBody: VirtualAgentRequestBody{
						Action:    "mockAction",
						RequestID: "mockRequestID",
						UserID:    "mockUserID",
						Message: &MessageBody{
							Text:  "mockText",
							Typed: true,
						},
					},
					Body: []MessageResponseBody{
						{
							Value: OutputLink{
								UIType: "mockUIType",
								Group:  "mockGroup",
								Label:  "mockLabel",
								Header: "mockHeader",
								Type:   "mockType",
								Value: OutputLinkValue{
									Action: "mockAction",
								},
							},
						},
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID: "",
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

			p.SetAPI(mockAPI)

			p.initializeAPI()

			req := test.httpTest.CreateHTTPRequest(test.request)
			req.Header.Add("Mattermost-User-ID", test.userID)
			rr := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, rr, req)
			test.httpTest.CompareHTTPResponse(rr, test.expectedResponse)
		})
	}
}

func TestPlugin_handlePickerSelection(t *testing.T) {
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
		"Everything works fine": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    "/api/v1/action_options",
				Body: model.PostActionIntegrationRequest{
					Context: map[string]interface{}{
						"selected_option": "mockOption",
					},
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID:                   "mockUserID",
			GetUserErr:               nil,
			GetDisconnectUserPostErr: nil,
			DisconnectUserErr:        nil,
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

			// mockCtrl := gomock.NewController(t)
			// mockedStore := mock_plugin.NewMockStore(mockCtrl)

			// p.store = mockedStore

			req := test.httpTest.CreateHTTPRequest(test.request)
			req.Header.Add("Mattermost-User-ID", test.userID)
			rr := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, rr, req)
			test.httpTest.CompareHTTPResponse(rr, test.expectedResponse)
		})
	}
}
