package plugin

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
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
	var dateMatched [][]string

	if date != "" {
		dateRegex := regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$`)
		dateMatched = dateRegex.FindAllStringSubmatch(date, -1)
	}

	if date != "" && dateMatched == nil {
		return "Please enter a valid date"
	}

	year, err := strconv.Atoi(strings.Split(date, "-")[0])
	if err != nil {
		return "Please enter a valid date"
	}

	currentYear := time.Now().Year()
	if year < currentYear-100 || year > currentYear+100 {
		return fmt.Sprintf("Year must be from %d to %d", currentYear-100, currentYear+100)
	}

	return ""
}

func (p *Plugin) validateTime(time string) string {
	var timeMatched [][]string

	if time != "" {
		timeRegex := regexp.MustCompile(`^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$`)
		timeMatched = timeRegex.FindAllStringSubmatch(time, -1)
	}

	if time != "" && timeMatched == nil {
		return "Please enter a valid time"
	}

	return ""
}
