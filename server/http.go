package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

type ErrorResponse struct {
	Error Error `json:"error"`
}

type Error struct {
	Detail  string `json:"detail"`
	Message string `json:"message"`
}

func (p *Plugin) CallJSON(method, path string, in, out interface{}, httpClient *http.Client) (responseData []byte, err error) {
	contentType := "application/json"
	buf := &bytes.Buffer{}
	err = json.NewEncoder(buf).Encode(in)
	if err != nil {
		return nil, err
	}
	return p.call(method, path, contentType, buf, out, httpClient)
}

func (p *Plugin) call(method, path, contentType string, inBody io.Reader, out interface{}, httpClient *http.Client) (responseData []byte, err error) {
	errContext := fmt.Sprintf("serviceNow virtual agent: Call failed: method:%s, path:%s", method, path)
	pathURL, err := url.Parse(strings.TrimSpace(fmt.Sprintf("%s%s", p.getConfiguration().ServiceNowURL, path)))
	if err != nil {
		return nil, errors.WithMessage(err, errContext)
	}

	if pathURL.Scheme == "" || pathURL.Host == "" {
		var baseURL *url.URL
		baseURL, err = url.Parse(p.getConfiguration().ServiceNowURL)
		if err != nil {
			return nil, errors.WithMessage(err, errContext)
		}
		if path[0] != '/' {
			path = "/" + path
		}
		path = baseURL.String() + path
	}

	req, err := http.NewRequest(method, path, inBody)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Add("Content-Type", contentType)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.Body == nil {
		return nil, nil
	}
	defer resp.Body.Close()

	responseData, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		if out != nil {
			err = json.Unmarshal(responseData, out)
			if err != nil {
				return responseData, err
			}
		}
		return responseData, nil

	case http.StatusNoContent:
		return nil, nil
	}

	errResp := ErrorResponse{}
	err = json.Unmarshal(responseData, &errResp)
	if err != nil {
		return responseData, errors.WithMessagef(err, "status: %s", resp.Status)
	}
	return responseData, fmt.Errorf("errorMessage %s. errorDetail: %s", errResp.Error.Message, errResp.Error.Detail)
}
