package plugin

import (
	"io"
	"net/url"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/stretchr/testify/require"
)

func Test_CallJSON(t *testing.T) {
	defer monkey.UnpatchAll()

	for _, testCase := range []struct {
		description string
	}{
		{
			description: "Request is sent successfully",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			c := client{}
			// p := Plugin{}

			// var ctx context.Context
			// httpClient := p.NewOAuth2Config().Client(ctx, &oauth2.Token{})
			// httpClient := http.Client{}

			monkey.PatchInstanceMethod(reflect.TypeOf(&c), "Call", func(_ *client, _, _, _ string, _ io.Reader, _ interface{}, _ url.Values) (responseData []byte, err error) {
				return []byte("mock"), nil
			})

			res, err := c.CallJSON("mockMethod", "mockPath", nil, nil, nil)

			// monkey.PatchInstanceMethod(reflect.TypeOf(&httpClient), "Do", func(_ *http.Client, _ *http.Request) (*http.Response, error) {
			// 	fmt.Printf("\n\n\ncalled mock do\n\n\n")
			// 	return &http.Response{}, nil
			// })
			// c.httpClient = &httpClient

			// p.setConfiguration(
			// 	&configuration{
			// 		ServiceNowURL: "mockServiceNowURL",
			// 	})

			// c.plugin = &p
			// requestBody := &VirtualAgentRequestBody{
			// 	Message: &MessageBody{
			// 		Text:  "messageText",
			// 		Typed: true,
			// 	},
			// 	RequestID: c.plugin.generateUUID(),
			// 	UserID:    "userID",
			// }
			// res, err := c.CallJSON(http.MethodPost, PathVirtualAgentBotIntegration, requestBody, nil, nil)

			// fmt.Printf("\n\n\n%+v\n\n\n%+v\n\n\n", res, err)

			require.NotNil(t, res)
			require.Nil(t, err)
		})
	}
}
