package main

const (
	HeaderMattermostUserID = "Mattermost-User-ID"

	ConnectSuccessMessage = "Thanks for linking your ServiceNow account!\n" +
		"You've connected your Mattermost account `%s` to ServiceNow."
	WelcomePretextMessage = "Welcome to the Mattermost ServiceNow Virtual Agent.\n" +
		"I'm here to help you. Let's start by linking your ServiceNow account.\n[Link to ServiceNow](%s)"

	PathOAuth2Connect  = "/oauth2/connect"
	PathOAuth2Complete = "/oauth2/complete"

	BotUsername    = "servicenow-virtual-agent"
	BotDisplayName = "ServiceNow Virtual Agent"
	BotDescription = "A bot account created by the plugin servicenow virtual agent."
)
