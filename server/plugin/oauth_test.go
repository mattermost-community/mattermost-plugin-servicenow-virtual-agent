package plugin

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/Brightscout/mattermost-plugin-servicenow-virtual-agent/server/testutils"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPlugin_httpOAuth2Connect(t *testing.T) {
	httpTestJSON := testutils.HTTPTest{
		T:       t,
		Encoder: testutils.EncodeJSON,
	}

	for name, test := range map[string]struct {
		httpTest         testutils.HTTPTest
		request          testutils.Request
		expectedResponse testutils.ExpectedResponse
		userID           string
		InitOAuth2Err    error
	}{
		"Everything works fine": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    "/api/v1/oauth2/connect",
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusFound,
			},
			userID:        "mockID",
			InitOAuth2Err: nil,
		},
		"When userId is absent": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    "/api/v1/oauth2/connect",
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusUnauthorized,
			},
			userID:        "",
			InitOAuth2Err: nil,
		},
		"When InitOAuth2 returns error": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    "/api/v1/oauth2/connect",
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusInternalServerError,
			},
			userID:        "mockUserID",
			InitOAuth2Err: errors.New("mockError"),
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

			monkey.PatchInstanceMethod(reflect.TypeOf(&p), "InitOAuth2", func(_ *Plugin, _ string) (string, error) {
				return "mockResponse", test.InitOAuth2Err
			})

			req := test.httpTest.CreateHTTPRequest(test.request)
			req.Header.Add("Mattermost-User-ID", test.userID)
			rr := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, rr, req)
			test.httpTest.CompareHTTPResponse(rr, test.expectedResponse)
		})
	}
}

func TestPlugin_httpOAuth2Complete(t *testing.T) {
	httpTestJSON := testutils.HTTPTest{
		T:       t,
		Encoder: testutils.EncodeJSON,
	}

	for name, test := range map[string]struct {
		httpTest         testutils.HTTPTest
		request          testutils.Request
		expectedResponse testutils.ExpectedResponse
		userID           string
		CompleteOAuthErr error
	}{
		"Everything works fine": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    "/api/v1/oauth2/complete?code=mockCode&state=mockState",
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
			userID:           "mockID",
			CompleteOAuthErr: nil,
		},
		"When query code is mossing": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    "/api/v1/oauth2/complete?state=mockState",
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusBadRequest,
			},
			userID:           "mockID",
			CompleteOAuthErr: nil,
		},
		"When query state is missing": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    "/api/v1/oauth2/complete?code=mockCode",
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusBadRequest,
			},
			userID:           "mockID",
			CompleteOAuthErr: nil,
		},
		"When CompleteOAuth returns error": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    "/api/v1/oauth2/complete?code=mockCode&state=mockState",
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusInternalServerError,
			},
			userID:           "mockID",
			CompleteOAuthErr: errors.New("mockError"),
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

			monkey.PatchInstanceMethod(reflect.TypeOf(&p), "CompleteOAuth2", func(_ *Plugin, _, _, _ string) error {
				return test.CompleteOAuthErr
			})

			req := test.httpTest.CreateHTTPRequest(test.request)
			req.Header.Add("Mattermost-User-ID", test.userID)
			rr := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, rr, req)
			test.httpTest.CompareHTTPResponse(rr, test.expectedResponse)
		})
	}
}

func TestPlugin_NewOAuth2Config(t *testing.T) {
	t.Run("Everything works fine", func(t *testing.T) {
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

		res := p.NewOAuth2Config()

		require.NotNil(t, res)
	})
}
