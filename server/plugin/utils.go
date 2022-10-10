package plugin

import (
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

func (p *Plugin) logAndSendErrorToUser(mattermostUserID, channelID, errorMessage string) {
	p.API.LogError(errorMessage)
	p.Ephemeral(mattermostUserID, channelID, GenericErrorMessage)
}

func (p *Plugin) generateUUID() string {
	return uuid.New().String()
}

func (p *Plugin) validateDate(date string) string {
	parsedDate, err := time.Parse(DateLayout, date)
	if err != nil {
		return DateValidationError
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
		return TimeValidationError
	}

	return ""
}
