package main

import (
	"fmt"

	"github.com/mattermost/mattermost-server/v6/model"
)

// Ephemeral sends an ephemeral message to a user
func (p *Plugin) Ephemeral(userID, channelID, format string, args ...interface{}) {
	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: channelID,
		Message:   fmt.Sprintf(format, args...),
	}
	_ = p.API.SendEphemeralPost(userID, post)
}

// DM posts a simple Direct Message to the specified user
func (p *Plugin) DM(mattermostUserID, format string, args ...interface{}) (string, error) {
	postID, err := p.dm(mattermostUserID, &model.Post{
		Message: fmt.Sprintf(format, args...),
	})
	if err != nil {
		return "", err
	}
	return postID, nil
}

func (p *Plugin) dm(mattermostUserID string, post *model.Post) (string, error) {
	channel, err := p.API.GetDirectChannel(mattermostUserID, p.botUserID)
	if err != nil {
		p.API.LogInfo("Couldn't get bot's DM channel", "user_id", mattermostUserID)
		return "", err
	}
	post.ChannelId = channel.Id
	post.UserId = p.botUserID
	sentPost, err := p.API.CreatePost(post)
	if err != nil {
		return "", err
	}
	return sentPost.Id, nil
}
