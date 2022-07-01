package main

import (
	"context"
	"net/http"

	"golang.org/x/oauth2"
)

type Client interface {
	GetMe(mattermostUserID string) (*ServiceNowUser, error)
	StartConverstaionWithVirtualAgent(userID string) error
	SendMessageToVirtualAgentAPI(userID, messageText string) error
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
