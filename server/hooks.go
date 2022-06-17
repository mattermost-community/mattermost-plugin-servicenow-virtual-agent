package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
)

func (p *Plugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
	// If the message is posted by bot simply return
	if post.UserId == p.botUserID {
		return
	}

	channel, appErr := p.API.GetChannel(post.ChannelId)
	if appErr != nil {
		p.API.LogError("error occurred while fetching channel by ID. ChannelID: %s. Error: %s", post.ChannelId, appErr.Error())
		return
	}

	if channel.Type != model.ChannelTypeDirect {
		return
	}

	botID := strings.Split(channel.Name, "__")[0]
	if botID != p.botUserID {
		return
	}

	mattermostUserID := post.UserId
	// Check if the user is connected to serviceNow
	_, err := p.GetUser(mattermostUserID)
	if err == nil {
		// TODO: Send the user message to serviceNow for further computation
		return
	}

	_, _ = p.DM(mattermostUserID, WelcomePretextMessage, fmt.Sprintf("%s%s", p.GetPluginURL(), PathOAuth2Connect))
}
