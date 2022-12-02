package plugin

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/serializer"
	"github.com/mattermost/mattermost-server/v5/model"
	"golang.org/x/oauth2"
)

type Client interface {
	GetMe(mattermostUserID string) (*serializer.ServiceNowUser, error)
	StartConverstaionWithVirtualAgent(userID string) error
	SendMessageToVirtualAgentAPI(userID, messageText string, typed bool, attachment *MessageAttachment) error
	OpenDialogRequest(body *model.OpenDialogRequest) error
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

func (c *client) OpenDialogRequest(body *model.OpenDialogRequest) error {
	postURL := fmt.Sprintf("%s%s", c.plugin.getConfiguration().MattermostSiteURL, PathOpenDialog)
	_, err := c.CallJSON(http.MethodPost, postURL, body, nil, nil)
	return err
}
