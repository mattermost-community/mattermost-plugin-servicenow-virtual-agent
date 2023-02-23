// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package kvstore

import (
	"crypto/sha512"
	"encoding/base64"
	"fmt"

	"github.com/mattermost/mattermost-server/v5/model"
)

type hashedKeyStore struct {
	store  KVStore
	prefix string
}

var _ KVStore = (*hashedKeyStore)(nil)

const OAuth2KeyPrefix = "oauth2_"

func NewHashedKeyStore(s KVStore, prefix string) KVStore {
	return &hashedKeyStore{
		store:  s,
		prefix: prefix,
	}
}

func (s hashedKeyStore) Load(key string) ([]byte, error) {
	if s.prefix == OAuth2KeyPrefix {
		return s.store.Load(hashKey(s.prefix, key))
	}

	return s.store.Load(encodeKey(s.prefix, key))
}

func (s hashedKeyStore) Store(key string, data []byte) error {
	if s.prefix == OAuth2KeyPrefix {
		return s.store.Store(hashKey(s.prefix, key), data)
	}

	return s.store.Store(encodeKey(s.prefix, key), data)
}

func (s hashedKeyStore) StoreTTL(key string, data []byte, ttlSeconds int64) error {
	if s.prefix == OAuth2KeyPrefix {
		return s.store.StoreTTL(hashKey(s.prefix, key), data, ttlSeconds)
	}

	return s.store.StoreTTL(encodeKey(s.prefix, key), data, ttlSeconds)
}

func (s hashedKeyStore) StoreWithOptions(key string, value []byte, opts model.PluginKVSetOptions) (bool, error) {
	if s.prefix == OAuth2KeyPrefix {
		return s.store.StoreWithOptions(hashKey(s.prefix, key), value, opts)
	}

	return s.store.StoreWithOptions(encodeKey(s.prefix, key), value, opts)
}

func (s hashedKeyStore) Delete(key string) error {
	if s.prefix == OAuth2KeyPrefix {
		return s.store.Delete(hashKey(s.prefix, key))
	}

	return s.store.Delete(encodeKey(s.prefix, key))
}

func encodeKey(prefix, key string) string {
	if key == "" {
		return prefix
	}

	encodedKey := base64.StdEncoding.EncodeToString([]byte(key))
	encodedKey = fmt.Sprintf("%s%s", prefix, encodedKey)
	return encodedKey
}

func hashKey(prefix, hashableKey string) string {
	if hashableKey == "" {
		return prefix
	}

	h := sha512.New()
	_, _ = h.Write([]byte(hashableKey))
	hashedKey := fmt.Sprintf("%s%x", prefix, h.Sum(nil))
	return hashedKey[:50]
}
