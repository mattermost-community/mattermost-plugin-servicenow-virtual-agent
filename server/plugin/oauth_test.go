package plugin

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/constants"
	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/testutils"
)

func TestPlugin_httpOAuth2Connect(t *testing.T) {
	defer monkey.UnpatchAll()

	httpTestJSON := testutils.HTTPTest{
		T:       t,
		Encoder: testutils.EncodeJSON,
	}

	for name, test := range map[string]struct {
		httpTest         testutils.HTTPTest
		request          testutils.Request
		expectedResponse testutils.ExpectedResponse
		InitOAuth2Err    error
	}{
		"httpOAuth2Connect works as expected": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    "/api/v1/oauth2/connect",
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusFound,
			},
		},
		"httpOAuth2Connect InitOAuth2 returns error": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    "/api/v1/oauth2/connect",
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusInternalServerError,
			},
			InitOAuth2Err: errors.New("error in initializing oAuth2"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			p, mockAPI := setupTestPlugin(&plugintest.API{}, nil)
			mockAPI.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
			mockAPI.On("LogError", testutils.GetMockArgumentsWithType("string", 6)...).Return()

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "InitOAuth2", func(_ *Plugin, _ string) (string, error) {
				return "mockResponse", test.InitOAuth2Err
			})

			req := test.httpTest.CreateHTTPRequest(test.request)
			req.Header.Add(constants.HeaderMattermostUserID, testutils.GetID())
			rr := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, rr, req)
			test.httpTest.CompareHTTPResponse(rr, test.expectedResponse)
		})
	}
}

func TestPlugin_httpOAuth2Complete(t *testing.T) {
	defer monkey.UnpatchAll()

	httpTestJSON := testutils.HTTPTest{
		T:       t,
		Encoder: testutils.EncodeJSON,
	}

	for name, test := range map[string]struct {
		httpTest         testutils.HTTPTest
		request          testutils.Request
		expectedResponse testutils.ExpectedResponse
		completeOAuthErr error
	}{
		"httpOAuth2Complete works as expected": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    "/api/v1/oauth2/complete?code=mockCode&state=mockState",
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusOK,
			},
		},
		"Missing query code": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    "/api/v1/oauth2/complete?state=mockState",
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusBadRequest,
			},
		},
		"Missing query state": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    "/api/v1/oauth2/complete?code=mockCode",
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusBadRequest,
			},
		},
		"httpOAuth2Complete CompleteOAuth returns error": {
			httpTest: httpTestJSON,
			request: testutils.Request{
				Method: http.MethodGet,
				URL:    "/api/v1/oauth2/complete?code=mockCode&state=mockState",
			},
			expectedResponse: testutils.ExpectedResponse{
				StatusCode: http.StatusInternalServerError,
			},
			completeOAuthErr: errors.New("error completing OAuth2"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			p, mockAPI := setupTestPlugin(&plugintest.API{}, nil)
			mockAPI.On("LogDebug", testutils.GetMockArgumentsWithType("string", 7)...).Return()
			mockAPI.On("LogError", testutils.GetMockArgumentsWithType("string", 6)...).Return()

			monkey.PatchInstanceMethod(reflect.TypeOf(p), "CompleteOAuth2", func(_ *Plugin, _, _, _ string) error {
				return test.completeOAuthErr
			})

			req := test.httpTest.CreateHTTPRequest(test.request)
			req.Header.Add(constants.HeaderMattermostUserID, testutils.GetID())
			rr := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, rr, req)
			test.httpTest.CompareHTTPResponse(rr, test.expectedResponse)
		})
	}
}

func TestPlugin_NewOAuth2Config(t *testing.T) {
	t.Run("NewOAuth2Config returns proper configuration", func(t *testing.T) {
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
