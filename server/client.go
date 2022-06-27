package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

type Client interface {
	GetMe(mattermostUserID string) (*ServiceNowUser, error)
}

type client struct {
	ctx        context.Context
	httpClient *http.Client
	plugin     *Plugin
}

func (p *Plugin) MakeClient(ctx context.Context, token *oauth2.Token) Client {
	httpClient := p.NewOAuth2Config().Client(ctx, token)
	c := &client{
		ctx:        ctx,
		httpClient: httpClient,
		plugin:     p,
	}
	return c
}

func (c *client) GetMe(mattermostUserID string) (*ServiceNowUser, error) {
	mattermostUser, appErr := c.plugin.API.GetUser(mattermostUserID)
	if appErr != nil {
		return nil, errors.Wrap(appErr, fmt.Sprintf("failed to get user details by mattermostUserID. UserID: %s", mattermostUserID))
	}

	userDetails := &UserDetails{}
	path := fmt.Sprintf("%s%s", c.plugin.getConfiguration().ServiceNowURL, PathGetUser)
	params := url.Values{}
	params.Add(SysQueryParam, fmt.Sprintf("email=%s", mattermostUser.Email))
	_, err := c.CallJSON(http.MethodGet, path, nil, userDetails, params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user details")
	}
	if len(userDetails.UserDetails) == 0 {
		return nil, errors.New(fmt.Sprintf("user doesn't exist on ServiceNow with email %s", mattermostUser.Email))
	}

	return userDetails.UserDetails[0], nil
}
