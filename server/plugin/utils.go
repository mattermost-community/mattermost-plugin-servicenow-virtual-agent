package plugin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/constants"
)

func (p *Plugin) logAndSendErrorToUser(mattermostUserID, channelID, errorMessage string) {
	p.API.LogError(errorMessage)
	p.Ephemeral(mattermostUserID, channelID, constants.GenericErrorMessage)
}

func (p *Plugin) generateUUID() string {
	return uuid.New().String()
}

func (p *Plugin) validateDate(date string) string {
	parsedDate, err := time.Parse(constants.DateLayout, date)
	if err != nil {
		return constants.ErrorDateValidation
	}

	year := parsedDate.Year()
	currentYear := time.Now().Year()
	if year < currentYear-100 || year > currentYear+100 {
		return fmt.Sprintf("Please enter year from %d to %d", currentYear-100, currentYear+100)
	}

	return ""
}

func (p *Plugin) validateTime(time string) string {
	var timeMatched [][]string

	timeRegex := regexp.MustCompile(`^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$`)
	timeMatched = timeRegex.FindAllStringSubmatch(time, -1)
	if timeMatched == nil {
		return constants.ErrorTimeValidation
	}

	return ""
}

func (p *Plugin) IsCharCountSafe(attachments []*model.SlackAttachment) bool {
	bytes, _ := json.Marshal(attachments)
	// 35 is the approx. length of one line added by the MM server for post action IDs and 100 is a buffer
	return utf8.RuneCountInString(string(bytes)) < model.POST_PROPS_MAX_RUNES-100-(len(attachments)*35)
}

func (p *Plugin) GetClientFromRequest(r *http.Request) Client {
	ctx := r.Context()
	token := ctx.Value(constants.ContextTokenKey).(*oauth2.Token)
	return p.MakeClient(ctx, token)
}
