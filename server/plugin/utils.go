package plugin

import "github.com/google/uuid"

func (p *Plugin) logAndSendErrorToUser(mattermostUserID, channelID, errorMessage string) {
	p.API.LogError(errorMessage)
	p.Ephemeral(mattermostUserID, channelID, GenericErrorMessage)
}

func (p *Plugin) generateUUID() string {
	return uuid.New().String()
}
