package plugin

const (
	HeaderMattermostUserID = "Mattermost-User-ID"
	HeaderServiceNowUserID = "ServiceNow-User-ID"
	// Used for storing the token in the request context to pass from one middleware to another
	// #nosec G101 -- This is a false positive. The below line is not a hardcoded credential
	ContextTokenKey ServiceNowOAuthToken = "ServiceNow-Oauth-Token"

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
	PathActionOptions              = "/action_options"

	SysQueryParam = "sysparm_query"

	BotUsername    = "servicenow-virtual-agent"
	BotDisplayName = "ServiceNow Virtual Agent"
	BotDescription = "A bot account created by the plugin ServiceNow Virtual Agent."

	DisconnectKeyword                = "disconnect"
	DisconnectUserContextName        = "Disconnect"
	DisconnectUserConfirmationMessge = "Are you sure you want to disconnect your ServiceNow account?"
	DisconnectUserRejectedMessage    = "You're still connected to your ServiceNow account."
	DisconnectUserSuccessMessage     = "Successfully disconnected your ServiceNow account."
	AlreadyDisconnectedMessage       = "You're already disconnected from your ServiceNow account."

	StartConversationAction         = "START_CONVERSATION"
	OutputTextUIType                = "OutputText"
	InputTextUIType                 = "InputText"
	FileUploadUIType                = "FileUpload"
	TopicPickerControlUIType        = "TopicPickerControl"
	PickerUIType                    = "Picker"
	BooleanUIType                   = "Boolean"
	OutputLinkUIType                = "OutputLink"
	GroupedPartsOutputControlUIType = "GroupedPartsOutputControl"
	OutputCardUIType                = "OutputCard"

	updatedPostBorderColor = "#74ccac"
)

// #nosec G101 -- This is a false positive. The below line is not a hardcoded credential
const (
	EmptyServiceNowURLErrorMessage               = "serviceNow URL should not be empty"
	EmptyServiceNowOAuthClientIDErrorMessage     = "serviceNow OAuth clientID should not be empty"
	EmptyServiceNowOAuthClientSecretErrorMessage = "serviceNow OAuth clientSecret should not be empty"
	EmptyEncryptionSecretErrorMessage            = "encryption secret should not be empty"
	EmptyWebhookSecretErrorMessage               = "webhook secret should not be empty"
)

type ServiceNowOAuthToken string
