package plugin

import (
	"context"
	"net/http"

	"github.com/Brightscout/mattermost-plugin-servicenow-virtual-agent/server/serializer"
	"golang.org/x/oauth2"
)

type Client interface {
	GetMe(mattermostUserID string) (*serializer.ServiceNowUser, error)
	StartConverstaionWithVirtualAgent(userID string) error
	SendMessageToVirtualAgentAPI(userID, messageText string, typed bool) error
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
