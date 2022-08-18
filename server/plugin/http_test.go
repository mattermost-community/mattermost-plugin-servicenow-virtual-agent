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

			monkey.PatchInstanceMethod(reflect.TypeOf(&c), "Call", func(_ *client, _, _, _ string, _ io.Reader, _ interface{}, _ url.Values) (responseData []byte, err error) {
				return []byte("mock"), nil
			})

			res, err := c.CallJSON("mockMethod", "mockPath", nil, nil, nil)

			require.NotNil(t, res)
			require.Nil(t, err)
		})
	}
}
