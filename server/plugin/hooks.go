package plugin

import (
	"context"
	"encoding/json"
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

	channel, appErr := p.API.GetChannel(post.ChannelId)
	if appErr != nil {
		p.API.LogError("error occurred while fetching channel by ID. ChannelID: %s. Error: %s", post.ChannelId, appErr.Error())
		return
	}

	if channel.Type != model.CHANNEL_DIRECT {
		return
	}

	channelName := strings.Split(channel.Name, "__")
	if channelName[0] != p.botUserID && channelName[1] != p.botUserID {
		return
	}

	mattermostUserID := post.UserId
	// Check if the user is connected to ServiceNow
	user, err := p.GetUser(mattermostUserID)
	if err != nil {
		if err == ErrNotFound {
			_, _ = p.DM(mattermostUserID, WelcomePretextMessage, fmt.Sprintf("%s%s", p.GetPluginURL(), PathOAuth2Connect))
		} else {
			p.logAndSendErrorToUser(mattermostUserID, channel.Id, fmt.Sprintf("error occurred while fetching user by ID. UserID: %s. Error: %s", mattermostUserID, err.Error()))
		}
		return
	}

	if strings.ToLower(post.Message) == DisconnectKeyword {
		_, _ = p.DMWithAttachments(post.UserId, p.CreateDisconnectUserAttachment())
		return
	}

	if len(post.FileIds) > 1 {
		p.logAndSendErrorToUser(mattermostUserID, channel.Id, "Cannot send more than one file attachment at a time.")
		return
	}

	token, err := p.ParseAuthToken(user.OAuth2Token)
	if err != nil {
		p.logAndSendErrorToUser(mattermostUserID, channel.Id, fmt.Sprintf("error occurred while decrypting token. Error: %s", err.Error()))
		return
	}

	var attachment *MessageAttachment
	if len(post.FileIds) == 1 {
		fileInfo, appErr := p.API.GetFileInfo(post.FileIds[0])
		if appErr != nil {
			p.logAndSendErrorToUser(mattermostUserID, channel.Id, fmt.Sprintf("Error getting file info. Error: %s", appErr.Message))
			return
		}

		date := time.Now()
		//TODO: Add a configuration setting for expiry time
		expiryTime := date.Add(time.Minute * 15)

		file := &FileStruct{
			ID:     post.FileIds[0],
			Expiry: expiryTime,
		}

		var jsonBytes []byte
		jsonBytes, err = json.Marshal(file)
		if err != nil {
			p.API.LogError("Error occurred while mashaling file. Error: %s", err.Error())
			return
		}

		var encrypted []byte
		encrypted, err = encrypt(jsonBytes, []byte(p.getConfiguration().EncryptionSecret))
		if err != nil {
			p.API.LogError("Error occurred while encrpting file. Error: %s", err.Error())
			return
		}

		attachment = &MessageAttachment{
			URL:         p.GetPluginURL() + "/file/" + encode(encrypted),
			ContentType: fileInfo.MimeType,
			FileName:    fileInfo.Name,
		}
	}

	client := p.MakeClient(context.Background(), token)
	if err = client.SendMessageToVirtualAgentAPI(user.UserID, post.Message, true, attachment); err != nil {
		p.logAndSendErrorToUser(mattermostUserID, channel.Id, err.Error())
	}
}
