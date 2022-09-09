package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/pkg/errors"
)

type ErrorResponse struct {
	Error Error `json:"error"`
}

type Error struct {
	Detail  string `json:"detail"`
	Message string `json:"message"`
}

func (c *client) CallJSON(method, path string, in, out interface{}, params url.Values) (responseData []byte, err error) {
	contentType := "application/json"
	buf := &bytes.Buffer{}
	err = json.NewEncoder(buf).Encode(in)
	if err != nil {
		return nil, err
	}
	return c.Call(method, path, contentType, buf, out, params)
}

func (c *client) Call(method, path, contentType string, inBody io.Reader, out interface{}, params url.Values) (responseData []byte, err error) {
	errContext := fmt.Sprintf("serviceNow virtual agent: Call failed: method:%s, path:%s", method, path)
	pathURL, err := url.Parse(path)
	if err != nil {
		return nil, errors.WithMessage(err, errContext)
	}

	if pathURL.Scheme == "" || pathURL.Host == "" {
		var baseURL *url.URL
		baseURL, err = url.Parse(c.plugin.getConfiguration().ServiceNowURL)
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
	if params != nil {
		req.URL.RawQuery = params.Encode()
	}
	if contentType != "" {
		req.Header.Add("Content-Type", contentType)
	}
	resp, err := c.httpClient.Do(req)
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

func ReturnStatusOK(w io.Writer) {
	m := make(map[string]string)
	m[model.STATUS] = model.STATUS_OK
	_, _ = w.Write([]byte(model.MapToJson(m)))
}

func (p *Plugin) OpenDialogRequest(w http.ResponseWriter, body model.OpenDialogRequest) {
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(body); err != nil {
		p.API.LogError("Error encoding request body. Error: %s", err.Error())
		http.Error(w, "Error encoding request body.", http.StatusInternalServerError)
		return
	}

	postURL := fmt.Sprintf("%s%s", p.getConfiguration().MattermostSiteURL, PathOpenDialog)
	req, err := http.NewRequest(http.MethodPost, postURL, buf)
	if err != nil {
		p.API.LogError("Error creating a POST request to open date/time selection dialog. Error: %s", err.Error())
		http.Error(w, "Error creating a POST request to open date/time selection dialog.", http.StatusInternalServerError)
		return
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		p.API.LogError("Error sending request to open date/time selection dialog. Error: %s", err.Error())
		http.Error(w, "Error sending request to open date/time selection dialog.", http.StatusInternalServerError)
		return
	}

	defer res.Body.Close()

	post := &Post{}
	if err = json.NewDecoder(res.Body).Decode(post); err != nil {
		p.API.LogError("Error decoding response body. Error: %s", err.Error())
		http.Error(w, "Error decoding response body.", http.StatusInternalServerError)
		return
	}

	if res.StatusCode != http.StatusOK {
		p.API.LogInfo("Request failed with status code %s", res.StatusCode)
		http.Error(w, "Request failed", res.StatusCode)
		return
	}
}
