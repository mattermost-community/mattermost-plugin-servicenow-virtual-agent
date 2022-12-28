package plugin

import (
	"fmt"
	"testing"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/serializer"
	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/store/kvstore"
)

func Test_LoadUser(t *testing.T) {
	defer monkey.UnpatchAll()

	for _, testCase := range []struct {
		description string
	}{
		{
			description: "User is loaded successfully from KV store using mattermostID",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			s := pluginStore{}

			monkey.Patch(kvstore.LoadJSON, func(_ kvstore.KVStore, _ string, _ interface{}) error {
				return nil
			})

			_, err := s.LoadUser("mock-userID")
			require.Nil(t, err)
		})
	}
}

func Test_LoadUserWithSysID(t *testing.T) {
	defer monkey.UnpatchAll()

	for _, testCase := range []struct {
		description string
	}{
		{
			description: "User is loaded successfully from KV store using ServiceNow ID",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			s := pluginStore{}

			monkey.Patch(kvstore.LoadJSON, func(_ kvstore.KVStore, _ string, _ interface{}) error {
				return nil
			})

			_, err := s.LoadUserWithSysID("mock-userID")

			require.Nil(t, err)
		})
	}
}

func Test_StoreUser(t *testing.T) {
	defer monkey.UnpatchAll()

	for _, testCase := range []struct {
		description string
	}{
		{
			description: "User is stored successfully in KV store",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			s := pluginStore{}

			monkey.Patch(kvstore.StoreJSON, func(_ kvstore.KVStore, _ string, _ interface{}) error {
				return nil
			})

			err := s.StoreUser(&serializer.User{})

			require.Nil(t, err)
		})
	}
}

func Test_LoadPostIDs(t *testing.T) {
	defer monkey.UnpatchAll()
	for _, testCase := range []struct {
		description string
		loadError   error
	}{
		{
			description: "Post IDs are loaded successfully from the KV store",
		},
		{
			description: "Post IDs are not loaded successfully from the KV store",
			loadError:   fmt.Errorf("error in loading post IDs"),
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			s := pluginStore{}

			monkey.Patch(kvstore.LoadJSON, func(_ kvstore.KVStore, _ string, _ interface{}) error {
				return testCase.loadError
			})

			_, err := s.LoadPostIDs("mock-userID")
			if testCase.loadError == nil {
				require.Nil(t, err)
			} else {
				assert.EqualError(t, err, testCase.loadError.Error())
			}
		})
	}
}
