package main

const (
	HeaderMattermostUserID = "Mattermost-User-ID"

	ConnectSuccessMessage = "Thanks for linking your ServiceNow account!\n" +
		"Your ServiceNow account (*%s*) has been connected to Mattermost."
	WelcomePretextMessage = "Welcome to the Mattermost ServiceNow Virtual Agent.\n" +
		"I'm here to help you. Let's start by linking your ServiceNow account.\n[Link to ServiceNow](%s)"
	GenericErrorMessage = "Something went wrong. Please try again later."

	PathOAuth2Connect              = "/oauth2/connect"
	PathOAuth2Complete             = "/oauth2/complete"
	PathUserDisconnect             = "/user/disconnect"
	PathGetUser                    = "/api/now/table/sys_user"
	PathVirtualAgentWebhook        = "/nowbot/processResponse"
	PathVirtualAgentBotIntegration = "/api/sn_va_as_service/bot/integration"

	SysQueryParam = "sysparm_query"

	BotUsername    = "servicenow-virtual-agent"
	BotDisplayName = "ServiceNow Virtual Agent"
	BotDescription = "A bot account created by the plugin ServiceNow Virtual Agent."

	DisconnectKeyword                = "disconnect"
	DisconnectUserContextName        = "Disconnect"
	DisconnectUserConfirmationMessge = "Are you sure you want to disconnect your ServiceNow account?"
	DisconnectUserRejectedMessage    = "You're still connected to your ServiceNow account."
	DisconnectUserSuccessMessage     = "Successfully disconnected your ServiceNow account."
	AlreadyDisconnectedMessage       = "You're not connected to your ServiceNow account."
)
