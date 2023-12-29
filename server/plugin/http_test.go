package plugin

import (
	"io"
	"net/url"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest/mock"
	"github.com/stretchr/testify/require"
)

func Test_CallJSON(t *testing.T) {
	defer monkey.UnpatchAll()

	for _, testCase := range []struct {
		description        string
		callMethodResponse []byte
	}{
		{
			description:        "Request is sent successfully",
			callMethodResponse: []byte("mockResponse"),
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			c := client{}

			monkey.PatchInstanceMethod(reflect.TypeOf(&c), "Call", func(_ *client, _, _, _ string, _ io.Reader, _ interface{}, _ url.Values) (responseData []byte, err error) {
				return testCase.callMethodResponse, nil
			})

			res, err := c.CallJSON(string(mock.AnythingOfType("string")), string(mock.AnythingOfType("string")), mock.AnythingOfType("io.Reader"), mock.AnythingOfType("interface{}"), nil)

			require.EqualValues(t, res, testCase.callMethodResponse)
			require.Nil(t, err)
		})
	}
}
