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
				URL:    "/api/v1/user/disconnect",
				Body:   model.PostActionIntegrationRequest{},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusUnauthorized,
			},
			userID:                   "",
			GetUserErr:               ErrNotFound,
			GetDisconnectUserPostErr: errors.New("mockErr"),
			DisconnectUserErr:        nil,
		},
		"User is disconnected successfully": {
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
			userID:                   "mock-userID",
			GetUserErr:               nil,
			GetDisconnectUserPostErr: nil,
			DisconnectUserErr:        nil,
		},
		"User not found and failed to create disconnect post": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    "/api/v1/user/disconnect",
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
				URL:    "/api/v1/user/disconnect",
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
			userID:                   "mock-userID",
			GetUserErr:               ErrNotFound,
			GetDisconnectUserPostErr: nil,
			DisconnectUserErr:        nil,
		},
		"DisconnectUserContextName is false": {
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
			userID:                   "mock-userID",
			GetUserErr:               nil,
			GetDisconnectUserPostErr: nil,
			DisconnectUserErr:        nil,
		},
		"Error occur while disconnecting user": {
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
	}{
		"Webhook secret is absent": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    "/api/v1/nowbot/processResponse",
				Body:   VirtualAgentResponse{},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusForbidden,
			},
		},
		"Webhook secret is present": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    "/api/v1/nowbot/processResponse?secret=mockWebhookSecret",
				Body:   VirtualAgentResponse{},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
		},
		"handleVirtualAgentWebhook empty body": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    "/api/v1/nowbot/processResponse?secret=mockWebhookSecret",
				Body:   nil,
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusInternalServerError,
			},
		},
		"Proper response is received from Virtual Agent": {
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
		httpTest          testutils.HTTPTest
		request           testutils.Request
		expectedResponse  testutils.ExpectedResponse
		userID            string
		ParseAuthTokenErr error
		LoadUserErr       error
	}{
		"Selected option is successfully sent to virtual Agent": {
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
			userID:            "mock-userID",
			ParseAuthTokenErr: nil,
			LoadUserErr:       nil,
		},
		"User is not present in store": {
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
			userID:            "mock-userID",
			ParseAuthTokenErr: nil,
			LoadUserErr:       errors.New("mockErr"),
		},
		"Error occurs while parsing OAuth token": {
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
			userID:            "mock-userID",
			ParseAuthTokenErr: errors.New("mockErr"),
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

			p.SetAPI(mockAPI)

			p.initializeAPI()

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "ParseAuthToken", func(_ *Plugin, _ string) (*oauth2.Token, error) {
				return &oauth2.Token{}, test.ParseAuthTokenErr
			})

			var c client
			monkey.PatchInstanceMethod(reflect.TypeOf(&c), "Call", func(_ *client, _, _, _ string, _ io.Reader, _ interface{}, _ url.Values) (responseData []byte, err error) {
				return nil, nil
			})

			mockCtrl := gomock.NewController(t)
			mockedStore := mock_plugin.NewMockStore(mockCtrl)

			mockedStore.EXPECT().LoadUser(test.userID).Return(&serializer.User{}, test.LoadUserErr)

			p.store = mockedStore

			req := test.httpTest.CreateHTTPRequest(test.request)
			req.Header.Add(HeaderMattermostUserID, test.userID)
			rr := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, rr, req)
			test.httpTest.CompareHTTPResponse(rr, test.expectedResponse)
		})
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
		"Selected date is successfully sent to virtual Agent": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathDateTimeSelection),
				Body: model.SubmitDialogRequest{
					CallbackId: "mockPostID__Date",
					Submission: map[string]interface{}{
						"date": "2022-09-23",
					},
					ChannelId: "mockChannelID",
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode:   http.StatusOK,
				Body:         &model.SubmitDialogResponse{},
				ResponseType: "application/json",
			},
			userID:            "mock-userID",
			ParseAuthTokenErr: nil,
		},
		"Selected date is invalid": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathDateTimeSelection),
				Body: model.SubmitDialogRequest{
					CallbackId: "mockPostID__Date",
					Submission: map[string]interface{}{
						"date": "2022-23-23",
					},
					ChannelId: "mockChannelID",
				},
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
			userID:            "mock-userID",
			ParseAuthTokenErr: nil,
		},
		"Selected time is successfully sent to virtual Agent": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathDateTimeSelection),
				Body: model.SubmitDialogRequest{
					CallbackId: "mockPostID__Time",
					Submission: map[string]interface{}{
						"time": "22:12",
					},
					ChannelId: "mockChannelID",
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode:   http.StatusOK,
				Body:         &model.SubmitDialogResponse{},
				ResponseType: "application/json",
			},
			userID:            "mock-userID",
			ParseAuthTokenErr: nil,
		},
		"Selected time is invalid": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathDateTimeSelection),
				Body: model.SubmitDialogRequest{
					CallbackId: "mockPostID__Time",
					Submission: map[string]interface{}{
						"time": "25:12",
					},
					ChannelId: "mockChannelID",
				},
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
			userID:            "mock-userID",
			ParseAuthTokenErr: nil,
		},
		"Selected date-time is successfully sent to virtual Agent": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathDateTimeSelection),
				Body: model.SubmitDialogRequest{
					CallbackId: "mockPostID__DateTime",
					Submission: map[string]interface{}{
						"date": "2022-09-23",
						"time": "22:12",
					},
					ChannelId: "mockChannelID",
				},
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
				Body: &model.SubmitDialogResponse{
					Errors: map[string]string{},
				},
				ResponseType: "application/json",
			},
			userID:            "mock-userID",
			ParseAuthTokenErr: nil,
		},
		"Selected date-time is invalid": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathDateTimeSelection),
				Body: model.SubmitDialogRequest{
					CallbackId: "mockPostID__DateTime",
					Submission: map[string]interface{}{
						"date": "2022-13-23",
						"time": "24:12",
					},
					ChannelId: "mockChannelID",
				},
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
			userID:            "mock-userID",
			ParseAuthTokenErr: nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := new(Plugin)

			mockAPI := &plugintest.API{}

			mockAPI.On("GetBundlePath").Return("mockString", nil)

			mockAPI.On("LogDebug", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("Logdebug error")

			mockAPI.On("LogError", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("LogError error")

			mockAPI.On("UpdatePost", mock.AnythingOfType("*model.Post")).Return(nil, nil)

			p.SetAPI(mockAPI)

			p.initializeAPI()

			var c client
			monkey.PatchInstanceMethod(reflect.TypeOf(&c), "SendMessageToVirtualAgentAPI", func(_ *client, _, _ string, _ bool) error {
				return nil
			})

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "ParseAuthToken", func(_ *Plugin, _ string) (*oauth2.Token, error) {
				return &oauth2.Token{}, test.ParseAuthTokenErr
			})

			mockCtrl := gomock.NewController(t)
			mockedStore := mock_plugin.NewMockStore(mockCtrl)

			mockedStore.EXPECT().LoadUser(test.userID).Return(&serializer.User{}, nil)

			p.store = mockedStore

			req := test.httpTest.CreateHTTPRequest(test.request)
			req.Header.Add(HeaderMattermostUserID, test.userID)
			rr := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, rr, req)
			test.httpTest.CompareHTTPResponse(rr, test.expectedResponse)
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
		httpTest          testutils.HTTPTest
		request           testutils.Request
		expectedResponse  testutils.ExpectedResponse
		userID            string
		ParseAuthTokenErr error
	}{
		"Selected date is successfully sent to virtual Agent": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodPost,
				URL:    fmt.Sprintf("/api/v1%s", PathDateTimeSelectionDialog),
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
			userID:            "mock-userID",
			ParseAuthTokenErr: nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := new(Plugin)

			mockAPI := &plugintest.API{}

			mockAPI.On("GetBundlePath").Return("mockString", nil)

			mockAPI.On("LogDebug", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("Logdebug error")

			mockAPI.On("LogError", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("LogError error")

			p.SetAPI(mockAPI)

			p.initializeAPI()

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "ParseAuthToken", func(_ *Plugin, _ string) (*oauth2.Token, error) {
				return &oauth2.Token{}, test.ParseAuthTokenErr
			})

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "OpenDialogRequest", func(_ *Plugin, _ http.ResponseWriter, _ model.OpenDialogRequest) {})

			mockCtrl := gomock.NewController(t)
			mockedStore := mock_plugin.NewMockStore(mockCtrl)

			mockedStore.EXPECT().LoadUser(test.userID).Return(&serializer.User{}, nil)

			p.store = mockedStore

			req := test.httpTest.CreateHTTPRequest(test.request)
			req.Header.Add(HeaderMattermostUserID, test.userID)
			rr := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, rr, req)
			test.httpTest.CompareHTTPResponse(rr, test.expectedResponse)
		})
	}
}
