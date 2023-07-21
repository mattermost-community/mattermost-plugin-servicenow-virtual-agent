package plugin

import (
	"sync"
	"time"

	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/serializer"
	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/store/kvstore"
)

const (
	UserKeyPrefix   = "user_"
	OAuth2KeyPrefix = "oauth2_"
)

const (
	OAuth2KeyExpiration   = 15 * time.Minute
	oAuth2StateTimeToLive = 300 // seconds
)

var ErrNotFound = kvstore.ErrNotFound

type Store interface {
	UserStore
	OAuth2StateStore
}

type UserStore interface {
	LoadUser(mattermostUserID string) (*serializer.User, error)
	StoreUser(user *serializer.User) error
	DeleteUser(mattermostUserID string) error
	LoadUserWithSysID(mattermostUserID string) (*serializer.User, error)
	DeleteUserTokenOnEncryptionSecretChange()
}

// OAuth2StateStore manages OAuth2 state
type OAuth2StateStore interface {
	VerifyOAuth2State(state string) error
	StoreOAuth2State(state string) error
}

type pluginStore struct {
	plugin   *Plugin
	basicKV  kvstore.KVStore
	oauth2KV kvstore.KVStore
	userKV   kvstore.KVStore
}

func (p *Plugin) NewStore(api plugin.API) Store {
	basicKV := kvstore.NewPluginStore(api)
	return &pluginStore{
		plugin:   p,
		basicKV:  basicKV,
		userKV:   kvstore.NewHashedKeyStore(basicKV, UserKeyPrefix),
		oauth2KV: kvstore.NewHashedKeyStore(kvstore.NewOneTimePluginStore(api, OAuth2KeyExpiration), OAuth2KeyPrefix),
	}
}

func (s *pluginStore) LoadUser(mattermostUserID string) (*serializer.User, error) {
	user := serializer.User{}
	err := kvstore.LoadJSON(s.userKV, mattermostUserID, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *pluginStore) LoadUserWithSysID(userID string) (*serializer.User, error) {
	user := serializer.User{}
	err := kvstore.LoadJSON(s.userKV, userID, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *pluginStore) StoreUser(user *serializer.User) error {
	err := kvstore.StoreJSON(s.userKV, user.MattermostUserID, user)
	if err != nil {
		return err
	}

	err = kvstore.StoreJSON(s.userKV, user.UserID, user)
	if err != nil {
		return err
	}

	return nil
}

func (s *pluginStore) DeleteUser(mattermostUserID string) error {
	u, err := s.LoadUser(mattermostUserID)
	if err != nil {
		return err
	}
	err = s.userKV.Delete(u.MattermostUserID)
	if err != nil {
		return err
	}

	return nil
}

func (s *pluginStore) VerifyOAuth2State(state string) error {
	data, err := s.oauth2KV.Load(state)
	if err != nil {
		if err == ErrNotFound {
			return errors.New("authentication attempt expired, please try again")
		}
		return err
	}

	if string(data) != state {
		return errors.New("invalid oauth state, please try again")
	}
	return nil
}

func (s *pluginStore) StoreOAuth2State(state string) error {
	return s.oauth2KV.StoreTTL(state, []byte(state), oAuth2StateTimeToLive)
}

func (s *pluginStore) DeleteUserTokenOnEncryptionSecretChange() {
	page := 0
	wg := new(sync.WaitGroup)

	for {
		kvList, err := s.plugin.API.KVList(page, DefaultPerPage)
		if err != nil {
			s.plugin.API.LogError("Failed to get the users.", "Error", err.Error())
			return
		}

		// isUserDeleted flag is used to check the condition for increasing the page number.
		// When a key is deleted, the users get shifted to the previous page, so if we fetch the next page, some users are missed.
		// If a key is deleted we don't increase the page number, else we increase it by 1.
		isUserDeleted := false
		for _, key := range kvList {
			wg.Add(1)

			go func(key string) {
				defer wg.Done()

				userID, isValidUserKey := IsValidUserKey(key)
				if !isValidUserKey {
					return
				}

				isUserDeleted = true
				decodedKey, decodeErr := decodeKey(userID)
				if decodeErr != nil {
					s.plugin.API.LogError("Unable to decode key", "UserID", userID, "Error", decodeErr.Error())
					return
				}

				if err := s.DeleteUser(decodedKey); err != nil {
					s.plugin.API.LogError("Unable to delete a user", "UserID", userID, "Error", err.Error())
					return
				}
			}(key)
		}

		// Wait for all goroutines to complete before continuing.
		wg.Wait()

		if len(kvList) < DefaultPerPage {
			break
		}

		if !isUserDeleted {
			page++
		}
	}
}
