package plugin

import (
	"io"
	"net/url"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func Test_SendMessageToVirtualAgentAPI(t *testing.T) {
	for _, testCase := range []struct {
		description string
		userID      string
		message     string
		typed       bool
		errMessage  string
	}{
		{
			description: "Everthing works fine",
			userID:      "mockID",
			message:     "mockMessage",
			typed:       true,
		},
		{
			description: "Error in 'CallJSON' method",
			userID:      "mockID",
			message:     "mockMessage",
			typed:       true,
			errMessage:  "mockErrMessage",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			c := new(client)

			monkey.PatchInstanceMethod(reflect.TypeOf(c), "Call", func(_ *client, _, _, _ string, _ io.Reader, _ interface{}, _ url.Values) (responseData []byte, err error) {
				if testCase.errMessage != "" {
					return nil, errors.New(testCase.errMessage)
				} else {
					return nil, nil
				}
			})

			err := c.SendMessageToVirtualAgentAPI(testCase.userID, testCase.message, testCase.typed)

			if testCase.errMessage != "" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_StartConverstaionWithVirtualAgent(t *testing.T) {
	for _, testCase := range []struct {
		description string
		userID      string
		errMessage  string
	}{
		{
			description: "Everthing works fine",
			userID:      "mockID",
		},
		{
			description: "Error in 'CallJSON' method",
			userID:      "mockID",
			errMessage:  "mockErrMessage",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			c := new(client)

			monkey.PatchInstanceMethod(reflect.TypeOf(c), "Call", func(_ *client, _, _, _ string, _ io.Reader, _ interface{}, _ url.Values) (responseData []byte, err error) {
				if testCase.errMessage != "" {
					return nil, errors.New(testCase.errMessage)
				} else {
					return nil, nil
				}
			})

			err := c.StartConverstaionWithVirtualAgent(testCase.userID)

			if testCase.errMessage != "" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_CreateOutputLinkAttachment(t *testing.T) {
	for _, testCase := range []struct {
		description string
		body        *OutputLink
	}{
		{
			description: "Everthing works fine",
			body: &OutputLink{
				Header: "mockHeader",
				Label:  "mockLabel",
				Value: OutputLinkValue{
					Action: "mockAction",
				},
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}

			p.CreateOutputLinkAttachment(testCase.body)
		})
	}
}

func Test_CreateTopicPickerControlAttachment(t *testing.T) {
	for _, testCase := range []struct {
		description string
		body        *TopicPickerControl
	}{
		{
			description: "Everthing works fine",
			body: &TopicPickerControl{
				PromptMessage: "mockPrompt",
				Options: []Option{{
					Label: "mockLabel",
				}},
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}

			p.CreateTopicPickerControlAttachment(testCase.body)
		})
	}
}

func Test_CreatePickerAttachment(t *testing.T) {
	for _, testCase := range []struct {
		description string
		body        *Picker
	}{
		{
			description: "Everthing works fine",
			body: &Picker{
				Label: "mockLabel",
				Options: []Option{{
					Label: "mockLabel",
				}},
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}

			p.CreatePickerAttachment(testCase.body)
		})
	}
}
