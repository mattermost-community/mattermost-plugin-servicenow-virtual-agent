package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

func (p *Plugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
	// If the message is posted by bot simply return
	// fmt.Printf("\n\n\n\n\npost %+v\n\n\n\n\n", post)
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
		fileInfo, err := p.API.GetFileInfo(post.FileIds[0])
		if err != nil {
			p.logAndSendErrorToUser(mattermostUserID, channel.Id, fmt.Sprintf("Error getting file info. Error: %s", err.Error()))
			return
		}

		fileLink, err := p.API.GetFileLink(post.FileIds[0])
		if err != nil {
			p.logAndSendErrorToUser(mattermostUserID, channel.Id, fmt.Sprintf("Error getting file link. Error: %s", err.Error()))
			return
		}

		fmt.Printf("\n\n\n\n link    %s \n\n\n\n\n", fileLink)
		fmt.Printf("\n\n\n\n type    %s \n\n\n\n\n", fileInfo.MimeType)
		fmt.Printf("\n\n\n\n name    %s \n\n\n\n\n", strings.Split(fileInfo.Name, ".")[0])

		attachment = &MessageAttachment{
			URL:         "https://www.google.com/imgres?imgurl=https%3A%2F%2Fwww.simplilearn.com%2Fice9%2Ffree_resources_article_thumb%2Fwhat_is_image_Processing.jpg&imgrefurl=https%3A%2F%2Fwww.simplilearn.com%2Fimage-processing-article&tbnid=Bx81dUgHmqLhzM&vet=12ahUKEwi5ubn3oYf5AhVdNLcAHQbgAawQMygCegUIARCMAQ..i&docid=NMmM-IXyCkU2hM&w=848&h=477&q=image&ved=2ahUKEwi5ubn3oYf5AhVdNLcAHQbgAawQMygCegUIARCMAQ",
			ContentType: "image/jpeg",
			FileName:    "what_is_image_Processing.jpeg",
		}
	}

	client := p.MakeClient(context.Background(), token)
	err = client.SendMessageToVirtualAgentAPI(user.UserID, post.Message, true, attachment)
	if err != nil {
		p.logAndSendErrorToUser(mattermostUserID, channel.Id, err.Error())
	}
}
