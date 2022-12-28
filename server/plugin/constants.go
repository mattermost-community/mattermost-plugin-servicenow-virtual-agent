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
	PathOpenDialog                 = "/api/v4/actions/dialogs/open"
	PathSetDateTimeDialog          = "/date_time"
	PathSetDateTime                = "/selected_date_time"

	SysQueryParam   = "sysparm_query"
	VideoQueryParam = "target_url"
	SecretParam     = "secret"

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
	OutputImageUIType               = "OutputImage"
	DateUIType                      = "Date"
	TimeUIType                      = "Time"
	DateTimeUIType                  = "DateTime"
	OutputCardSmallImageType        = "Small image with text"
	OutputCardLargeImageType        = "Large image with text"
	OutputCardVideoType             = "Youtube Video Card"
	OutputCardRecordType            = "Card"

	InvalidImageLinkError = "Invalid image link."
	ItemTypeImage         = "image"
	ItemTypeFile          = "file"
	ItemTypePicture       = "Picture"
	DateValue             = "date"
	TimeValue             = "time"
	DateTimeDialogType    = "type"
	DateLayout            = "2006-01-02"

	ContextKeySelectedLabel  = "selected_label"
	ContextKeySelectedValue  = "selected_value"
	ContextKeySelectedOption = "selected_option"

	StyleCarousel = "carousel"

	DateValidationError    = "Please enter a valid date"
	TimeValidationError    = "Please enter a valid time"
	InvalidCallbackIDError = "Invalid callback ID."
	NotAuthorizedError     = "Not authorized"

	UploadImageMessage = "\n(**Note:** Please upload an image using the Mattermost `Upload files` option OR use the shorthand `Ctrl+U`.)"
	UploadFileMessage  = "\n(**Note:** Please upload a file using the Mattermost `Upload files` option OR use the shorthand `Ctrl+U`.)"

	PathParamEncryptedFileInfo = "encryptedFileInfo"

	updatedPostBorderColor            = "#74ccac"
	AttachmentLinkExpiryTimeInMinutes = 15

	YoutubeURL = "https://www.youtube.com/watch?v=%s"

	PublishSeriveNowVAIsTypingJobName = "PublishSeriveNowVAIsTypingJob"
	cronPrefix                        = "cron_"
	// ChannelCacheTTL contains the value after which cache entries are expired. This value is in minutes.
	ChannelCacheTTL = 1440
)

// #nosec G101 -- This is a false positive. The below line is not a hardcoded credential
const (
	EmptyServiceNowURLErrorMessage               = "serviceNow URL should not be empty"
	EmptyServiceNowOAuthClientIDErrorMessage     = "serviceNow OAuth clientID should not be empty"
	EmptyServiceNowOAuthClientSecretErrorMessage = "serviceNow OAuth clientSecret should not be empty"
	EmptyEncryptionSecretErrorMessage            = "encryption secret should not be empty"
	EmptyWebhookSecretErrorMessage               = "webhook secret should not be empty"
	InvalidChannelCacheSizeErrorMessage          = "direct message channel cache size should be greater than zero"
)

type ServiceNowOAuthToken string
