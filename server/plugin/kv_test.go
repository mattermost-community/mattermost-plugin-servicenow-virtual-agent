package plugin

import (
	"testing"

	"bou.ke/monkey"
	"github.com/Brightscout/mattermost-plugin-servicenow-virtual-agent/server/serializer"
	"github.com/Brightscout/mattermost-plugin-servicenow-virtual-agent/server/store/kvstore"
	"github.com/stretchr/testify/require"
)

func Test_LoadUser(t *testing.T) {
	defer monkey.UnpatchAll()

	for _, testCase := range []struct {
		description string
	}{
		{
			description: "User is loaded successfully from KV store usind mattermostID",
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

// func Test_DeleteUser(t *testing.T) {
// 	for _, testCase := range []struct {
// 		description string
// 	}{
// 		{
// 			description: "Ephemeral post is successfully created",
// 		},
// 	} {
// 		t.Run(testCase.description, func(t *testing.T) {
// 			s := pluginStore{}

// 			monkey.PatchInstanceMethod(reflect.TypeOf(&s), "LoadUser", func(_ *pluginStore, _ string) (*serializer.User, error) {
// 				return &serializer.User{}, nil
// 			})

// 			monkey.Patch(kvstore.StoreJSON, func(_ kvstore.KVStore, _ string, _ interface{}) error {
// 				return nil
// 			})

// 			_, err := s.LoadUserWithSysID("mock-userID")

// 			require.Nil(t, err)
// 		})
// 	}
// }
