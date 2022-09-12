package testutils

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Request stores http Request basic data
type Request struct {
	Method string
	URL    string
	Header http.Header
	Body   interface{}
}

// ExpectedResponse stores expected response basic data
type ExpectedResponse struct {
	StatusCode   int
	ResponseType string
	Body         interface{}
}

// HTTPTest encapsulates data for testing needs
type HTTPTest struct {
	*testing.T
	Encoder func(interface{}) ([]byte, error)
}

// EncodeJSON encodes json data in bytes
func EncodeJSON(data interface{}) ([]byte, error) {
	if data == nil {
		return []byte{}, nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return []byte{}, errors.Wrap(err, "Error while encoding json")
	}

	return b, nil
}

// EncodeString encodes string data in bytes
func EncodeString(data interface{}) ([]byte, error) {
	if data == nil {
		return nil, nil
	}

	body, ok := data.(string)
	if !ok {
		return nil, errors.New("error while encoding string")
	}

	return []byte(body), nil
}

// CreateHTTPRequest creates http Request with basic data
func (test *HTTPTest) CreateHTTPRequest(request Request) *http.Request {
	tassert := assert.New(test.T)
	data, err := test.Encoder(request.Body)
	tassert.NoError(err)

	var body io.Reader = bytes.NewBuffer(data)

	req, err := http.NewRequest(request.Method, request.URL, body)
	tassert.NoError(err, "Error while creating Request")

	if request.Header != nil {
		req.Header = request.Header
	}

	return req
}

// CompareHTTPResponse compares expected response with actual response
func (test *HTTPTest) CompareHTTPResponse(resp *httptest.ResponseRecorder, expected ExpectedResponse) {
	testAssert := assert.New(test.T)
	testAssert.Equal(expected.StatusCode, resp.Code, "Http status codes are different")

	if expected.Body != nil {
		expectedBody, err := test.Encoder(expected.Body)
		testAssert.NoError(err)

		testAssert.Equal(expected.ResponseType, resp.Header().Get("Content-Type"))

		respBody := resp.Body.Bytes()

		testAssert.Equal(expectedBody, respBody)
	}
}

func GetMockArgumentsWithType(typeString string, num int) []interface{} {
	ret := make([]interface{}, num)
	for i := 0; i < len(ret); i++ {
		ret[i] = mock.AnythingOfTypeArgument(typeString)
	}
	return ret
}
