package plugin

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"bou.ke/monkey"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type mockBLock struct{}

func (b *mockBLock) BlockSize() int { return 0 }

func (b *mockBLock) Encrypt(_, _ []byte) {}

func (b *mockBLock) Decrypt(_, _ []byte) {}

type mockAesgcm struct{}

func (a *mockAesgcm) NonceSize() int { return 1 }

func (a *mockAesgcm) Overhead() int { return 1 }

func (a *mockAesgcm) Seal(dst, nonce, plaintext, additionalData []byte) []byte { return []byte("mock") }

func (a *mockAesgcm) Open(dst, nonce, ciphertext, additionalData []byte) ([]byte, error) {
	return []byte("mock"), nil
}

func Test_NewEncodedAuthToken(t *testing.T) {
	defer monkey.UnpatchAll()

	for _, testCase := range []struct {
		description    string
		expectedError  string
		newCipherError error
		newGCMError    error
		resdFullError  error
	}{
		{
			description: "OAuth token is encoded successfully",
		},
		{
			description:    "Failed to create oAuth token because aes.NewCipher gives error",
			expectedError:  "failed to create auth token: mockError",
			newCipherError: errors.New("mockError"),
		},
		{
			description:   "Failed to create oAuth token because cipher.NewGCM gives error",
			expectedError: "failed to create auth token: mockError",
			newGCMError:   errors.New("mockError"),
		},
		{
			description:   "Failed to create oAuth token because io.ReadFull gives error",
			expectedError: "failed to create auth token: mockError",
			resdFullError: errors.New("mockError"),
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}

			p.setConfiguration(
				&configuration{
					EncryptionSecret: "mockEncryptionSecret",
				})

			monkey.Patch(aes.NewCipher, func(a []byte) (cipher.Block, error) {
				return &mockBLock{}, testCase.newCipherError
			})

			monkey.Patch(cipher.NewGCM, func(_ cipher.Block) (cipher.AEAD, error) {
				return &mockAesgcm{}, testCase.newGCMError
			})

			monkey.Patch(io.ReadFull, func(_ io.Reader, _ []byte) (int, error) {
				return 1, testCase.resdFullError
			})

			tok := &oauth2.Token{}

			res, err := p.NewEncodedAuthToken(tok)

			if testCase.expectedError != "" {
				require.EqualError(t, err, testCase.expectedError)
				require.EqualValues(t, "", res)
			} else {
				require.Nil(t, err)
				require.NotEqualValues(t, "", res)
			}
		})
	}
}

func Test_ParseAuthToken(t *testing.T) {
	defer monkey.UnpatchAll()

	for _, testCase := range []struct {
		description    string
		expectedError  string
		newCipherError error
		newGCMError    error
		encodedToken   string
		unmarshalError error
	}{
		{
			description:  "OAuth2 token is parsed successfully",
			encodedToken: "mockEncodedToken",
		},
		{
			description:    "Failed to decode oAuth token because aes.NewCipher gives error",
			expectedError:  "mockError",
			newCipherError: errors.New("mockError"),
			encodedToken:   "mockEncodedToken",
		},
		{
			description:   "Failed to decode oAuth token because cipher.NewGCM gives error",
			expectedError: "mockError",
			newGCMError:   errors.New("mockError"),
			encodedToken:  "mockEncodedToken",
		},
		{
			description:   "Failed to decode oAuth token because token is too short",
			expectedError: "token too short",
			encodedToken:  "",
		},
		{
			description:    "Failed to decode oAuth token because json.Unmarshal gives error",
			expectedError:  "mockError",
			unmarshalError: errors.New("mockError"),
			encodedToken:   "mockEncodedToken",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			p := Plugin{}

			p.setConfiguration(
				&configuration{
					EncryptionSecret: "mockEncryptionSecret",
				})

			monkey.Patch(aes.NewCipher, func(a []byte) (cipher.Block, error) {
				return &mockBLock{}, testCase.newCipherError
			})

			monkey.Patch(cipher.NewGCM, func(_ cipher.Block) (cipher.AEAD, error) {
				return &mockAesgcm{}, testCase.newGCMError
			})

			monkey.Patch(json.Unmarshal, func(_ []byte, _ interface{}) error {
				return testCase.unmarshalError
			})

			res, err := p.ParseAuthToken(testCase.encodedToken)
			if testCase.expectedError != "" {
				require.EqualError(t, err, testCase.expectedError)
				require.Nil(t, res)
			} else {
				require.Nil(t, err)
			}
		})
	}
}
