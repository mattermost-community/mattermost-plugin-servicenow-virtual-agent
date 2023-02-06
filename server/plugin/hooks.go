package plugin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

type FileStruct struct {
	ID     string
	Expiry time.Time
}

func (p *Plugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
	// If the message is posted by bot simply return
	if post.UserId == p.botUserID {
		return
	}

	isBotDMChannel := true
	cacheVal, err := p.channelCache.Get(post.ChannelId)
	if err == nil {
		isBotDMChannel, _ = cacheVal.(bool)
	} else {
		channel, channelErr := p.API.GetChannel(post.ChannelId)
		if channelErr != nil {
			p.API.LogError("Error occurred while fetching the channel by ID. ChannelID: %s. Error: %s", post.ChannelId, channelErr.Error())
			return
		}

		channelNameArr := strings.Split(channel.Name, "__")
		if len(channelNameArr) != 2 || (channelNameArr[0] != p.botUserID && channelNameArr[1] != p.botUserID) {
			isBotDMChannel = false
		}

		if err = p.channelCache.SetWithExpire(post.ChannelId, isBotDMChannel, time.Minute*time.Duration(ChannelCacheTTL)); err != nil {
			p.API.LogDebug("Failed to add channel in cache", "Error", err.Error())
		}
	}

	if !isBotDMChannel {
		return
	}

	mattermostUserID := post.UserId
	// Check if the user is connected to ServiceNow
	user, err := p.GetUser(mattermostUserID)
	if err != nil {
		if err == ErrNotFound {
			_, _ = p.DM(mattermostUserID, WelcomePretextMessage, fmt.Sprintf("%s%s", p.GetPluginURL(), PathOAuth2Connect))
		} else {
			p.logAndSendErrorToUser(mattermostUserID, post.ChannelId, fmt.Sprintf("Error occurred while fetching user by ID. UserID: %s. Error: %s", mattermostUserID, err.Error()))
		}
		return
	}

	if strings.ToLower(post.Message) == DisconnectKeyword {
		_, _ = p.DMWithAttachments(post.UserId, p.CreateDisconnectUserAttachment())
		return
	}

	if len(post.FileIds) > 1 {
		p.logAndSendErrorToUser(mattermostUserID, post.ChannelId, "Cannot send more than one file attachment at a time.")
		return
	}

	token, err := p.ParseAuthToken(user.OAuth2Token)
	if err != nil {
		p.logAndSendErrorToUser(mattermostUserID, post.ChannelId, fmt.Sprintf("Error occurred while decrypting token. Error: %s", err.Error()))
		return
	}

	var attachment *MessageAttachment
	if len(post.FileIds) == 1 {
		attachment, err = p.CreateMessageAttachment(post.FileIds[0], mattermostUserID)
		if err != nil {
			p.logAndSendErrorToUser(mattermostUserID, post.ChannelId, err.Error())
			return
		}
	}

	client := p.MakeClient(context.Background(), token)
	if err = client.SendMessageToVirtualAgentAPI(user.UserID, post.Message, true, attachment); err != nil {
		p.logAndSendErrorToUser(mattermostUserID, post.ChannelId, err.Error())
	}
}
