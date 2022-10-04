package plugin

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Brightscout/mattermost-plugin-servicenow-virtual-agent/server/serializer"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

func (p *Plugin) InitOAuth2(mattermostUserID string) (string, error) {
	if _, err := p.GetUser(mattermostUserID); err == nil {
		return "", fmt.Errorf("user is already connected to ServiceNow")
	}

	conf := p.NewOAuth2Config()
	state := fmt.Sprintf("%v_%v", model.NewId()[0:15], mattermostUserID)
	if err := p.store.StoreOAuth2State(state); err != nil {
		return "", err
	}

	return conf.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

func (p *Plugin) CompleteOAuth2(authedUserID, code, state string) error {
	if authedUserID == "" || code == "" || state == "" {
		return errors.New("missing user, code or state")
	}

	oconf := p.NewOAuth2Config()

	err := p.store.VerifyOAuth2State(state)
	if err != nil {
		return errors.WithMessage(err, "missing stored state")
	}

	mattermostUserID := strings.Split(state, "_")[1]
	if mattermostUserID != authedUserID {
		return errors.New("not authorized, user ID mismatch")
	}

	ctx := context.Background()
	tok, err := oconf.Exchange(ctx, code)
	if err != nil {
		return err
	}

	client := p.MakeClient(context.Background(), tok)
	serviceNowUser, err := client.GetMe(mattermostUserID)
	if err != nil {
		return err
	}

	encryptedToken, err := p.NewEncodedAuthToken(tok)
	if err != nil {
		return err
	}

	u := &serializer.User{
		MattermostUserID: mattermostUserID,
		OAuth2Token:      encryptedToken,
		ServiceNowUser:   *serviceNowUser,
	}

	err = p.store.StoreUser(u)
	if err != nil {
		return err
	}

	_, err = p.DM(mattermostUserID, ConnectSuccessMessage, serviceNowUser.Email)
	if err != nil {
		return err
	}

	err = client.StartConverstaionWithVirtualAgent(mattermostUserID)
	if err != nil {
		return err
	}

	return nil
}

func (p *Plugin) GetUser(mattermostUserID string) (*serializer.User, error) {
	storedUser, err := p.store.LoadUser(mattermostUserID)
	if err != nil {
		return nil, err
	}

	return storedUser, nil
}

func (p *Plugin) DisconnectUser(mattermostUserID string) error {
	if err := p.store.DeleteUser(mattermostUserID); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) CreateDisconnectUserAttachment() *model.SlackAttachment {
	disconnectUserPath := fmt.Sprintf("%s%s", p.GetPluginURLPath(), PathUserDisconnect)
	disconnectUserAttachment := &model.SlackAttachment{
		Title: DisconnectUserConfirmationMessge,
		Color: "#FF0000",
		Actions: []*model.PostAction{
			{
				Type: "button",
				Name: "Yes",
				Integration: &model.PostActionIntegration{
					URL: disconnectUserPath,
					Context: map[string]interface{}{
						DisconnectUserContextName: true,
					},
				},
			},
			{
				Type: "button",
				Name: "No",
				Integration: &model.PostActionIntegration{
					URL: disconnectUserPath,
					Context: map[string]interface{}{
						DisconnectUserContextName: false,
					},
				},
			},
		},
	}

	return disconnectUserAttachment
}

func (p *Plugin) GetDisconnectUserPost(mattermostUserID, message string) (*model.Post, error) {
	disconnectUserAttachment := &model.SlackAttachment{
		Title: message,
		Color: "#32CD32",
	}

	post, err := p.GetPostWithSlackAttachment(mattermostUserID, disconnectUserAttachment)
	if err != nil {
		return nil, err
	}

	return post, nil
}

func (c *client) GetMe(mattermostUserID string) (*serializer.ServiceNowUser, error) {
	mattermostUser, appErr := c.plugin.API.GetUser(mattermostUserID)
	if appErr != nil {
		return nil, errors.Wrap(appErr, fmt.Sprintf("failed to get user details by mattermostUserID. UserID: %s", mattermostUserID))
	}

	userDetails := &serializer.UserDetails{}
	path := fmt.Sprintf("%s%s", c.plugin.getConfiguration().ServiceNowURL, PathGetUser)
	params := url.Values{}
	params.Add(SysQueryParam, fmt.Sprintf("email=%s", mattermostUser.Email))

	_, err := c.CallJSON(http.MethodGet, path, nil, userDetails, params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user details")
	}
	if len(userDetails.UserDetails) == 0 {
		return nil, fmt.Errorf("user doesn't exist on ServiceNow with email %s", mattermostUser.Email)
	}
	if len(userDetails.UserDetails) > 1 {
		c.plugin.API.LogWarn("multiple users with the same email address exist on ServiceNow instance", "Email", mattermostUser.Email, "Instance", c.plugin.getConfiguration().ServiceNowURL)
	}

	return userDetails.UserDetails[0], nil
}
