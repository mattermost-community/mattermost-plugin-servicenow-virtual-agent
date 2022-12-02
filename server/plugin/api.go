package plugin

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/serializer"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

// ServeHTTP demonstrates a plugin that handles HTTP requests.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.API.LogDebug("New request:", "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method)

	p.initializeAPI().ServeHTTP(w, r)
}

func (p *Plugin) initializeAPI() *mux.Router {
	r := mux.NewRouter()
	r.Use(p.withRecovery)
	p.handleStaticFiles(r)

	apiRouter := r.PathPrefix("/api/v1").Subrouter()

	// Add custom routes here
	apiRouter.HandleFunc(PathOAuth2Connect, p.checkAuth(p.httpOAuth2Connect)).Methods(http.MethodGet)
	apiRouter.HandleFunc(PathOAuth2Complete, p.checkAuth(p.httpOAuth2Complete)).Methods(http.MethodGet)
	apiRouter.HandleFunc(PathUserDisconnect, p.checkAuth(p.handleUserDisconnect)).Methods(http.MethodPost)
	apiRouter.HandleFunc(PathActionOptions, p.checkAuth(p.checkOAuth(p.handlePickerSelection))).Methods(http.MethodPost)
	apiRouter.HandleFunc(PathSetDateTimeDialog, p.checkAuth(p.checkOAuth(p.handleSetDateTimeDialog))).Methods(http.MethodPost)
	apiRouter.HandleFunc(PathSetDateTime, p.checkAuth(p.checkOAuth(p.handleSetDateTime))).Methods(http.MethodPost)
	apiRouter.HandleFunc(PathVirtualAgentWebhook, p.checkAuthBySecret(p.handleVirtualAgentWebhook)).Methods(http.MethodPost)
	apiRouter.HandleFunc(fmt.Sprintf("/file/{%s}", PathParamEncryptedFileInfo), p.handleFileAttachments).Methods(http.MethodGet)

	r.Handle("{anything:.*}", http.NotFoundHandler())

	return r
}

func (p *Plugin) handleAPIError(w http.ResponseWriter, apiErr *serializer.APIErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	errorBytes, err := json.Marshal(apiErr)
	if err != nil {
		p.API.LogError("Failed to marshal API error", "Error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(apiErr.StatusCode)
	if _, err = w.Write(errorBytes); err != nil {
		p.API.LogError("Failed to write JSON response", "Error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (p *Plugin) checkAuthBySecret(handleFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Replace all occurrences of " " with "+" in WebhookSecret.
		webhookSecret := strings.ReplaceAll(r.FormValue(SecretParam), " ", "+")
		if statusCode, err := verifyHTTPSecret(p.getConfiguration().WebhookSecret, webhookSecret); err != nil {
			p.API.LogError("Invalid secret", "Error", err.Error())
			p.handleAPIError(w, &serializer.APIErrorResponse{StatusCode: statusCode, Message: fmt.Sprintf("Invalid Secret. Error: %s", err.Error())})
			return
		}

		handleFunc(w, r)
	}
}

// Ref: mattermost plugin confluence(https://github.com/mattermost/mattermost-plugin-confluence/blob/3ee2aa149b6807d14fe05772794c04448a17e8be/server/controller/main.go#L97)
func verifyHTTPSecret(expected, got string) (status int, err error) {
	for {
		if subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1 {
			break
		}

		unescaped, _ := url.QueryUnescape(got)
		if unescaped == got {
			return http.StatusForbidden, errors.New("request URL: secret did not match")
		}
		got = unescaped
	}

	return 0, nil
}

// handleStaticFiles handles the static files under the assets directory.
func (p *Plugin) handleStaticFiles(r *mux.Router) {
	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		p.API.LogWarn("Failed to get bundle path.", "Error", err.Error())
		return
	}

	// This will serve static files from the 'assets' directory under '/static/<filename>'
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(bundlePath, "assets")))))
}

// handleFileAttachments returns the data of the fileID passed in the request URL.
func (p *Plugin) handleFileAttachments(w http.ResponseWriter, r *http.Request) {
	pathParams := mux.Vars(r)
	encryptedFileInfo := pathParams[PathParamEncryptedFileInfo]

	decoded, err := decode(encryptedFileInfo)
	if err != nil {
		p.API.LogError("Error occurred while decoding the file. Error: %s", err.Error())
		p.handleAPIError(w, &serializer.APIErrorResponse{StatusCode: http.StatusBadRequest, Message: "Error occurred while decoding the file."})
		return
	}

	jsonBytes, err := decrypt(decoded, []byte(p.getConfiguration().EncryptionSecret))
	if err != nil {
		p.API.LogError("Error occurred while decrypting the file. Error: %s", err.Error())
		p.handleAPIError(w, &serializer.APIErrorResponse{StatusCode: http.StatusInternalServerError, Message: "Error occurred while decrypting the file."})
		return
	}

	fileInfo := FileStruct{}
	if err = json.Unmarshal(jsonBytes, &fileInfo); err != nil {
		p.API.LogError("Error occurred while unmarshaling the file. Error: %s", err.Error())
		p.handleAPIError(w, &serializer.APIErrorResponse{StatusCode: http.StatusInternalServerError, Message: "Error occurred while unmarshaling the file."})
		return
	}

	currentTime := time.Now().UTC()
	if currentTime.After(fileInfo.Expiry) {
		http.NotFound(w, r)
		return
	}

	data, appErr := p.API.GetFile(fileInfo.ID)
	if appErr != nil {
		p.API.LogError("Couldn't get file data. FileID: %s", fileInfo.ID)
		p.handleAPIError(w, &serializer.APIErrorResponse{StatusCode: http.StatusInternalServerError, Message: "Couldn't get the file data."})
		return
	}

	w.Header().Set("Content-Type", http.DetectContentType(data))
	if _, err = w.Write(data); err != nil {
		p.API.LogError("Error occurred writing the file content in response. Error: %s", err.Error())
		p.handleAPIError(w, &serializer.APIErrorResponse{StatusCode: http.StatusInternalServerError, Message: "Error occurred while writing the file content in response."})
		return
	}
}

func (p *Plugin) withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if x := recover(); x != nil {
				p.API.LogError("Recovered from a panic",
					"URL", r.URL.String(),
					"Error", x,
					"Stack", string(debug.Stack()))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (p *Plugin) checkAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get(HeaderMattermostUserID)
		if userID == "" {
			p.handleAPIError(w, &serializer.APIErrorResponse{StatusCode: http.StatusUnauthorized, Message: NotAuthorizedError})
			return
		}

		handler(w, r)
	}
}

func (p *Plugin) checkOAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get(HeaderMattermostUserID)
		user, err := p.store.LoadUser(userID)
		if err != nil {
			p.API.LogError("Error loading user from KV store.", "Error", err.Error())
			return
		}
		// Adding the ServiceNow User ID in the request headers to pass it to the next handler
		r.Header.Set(HeaderServiceNowUserID, user.UserID)

		token, err := p.ParseAuthToken(user.OAuth2Token)
		if err != nil {
			p.API.LogError("Error parsing OAuth2 token.", "Error", err.Error())
			return
		}

		ctx := context.WithValue(r.Context(), ContextTokenKey, token)
		r = r.Clone(ctx)
		handler(w, r)
	}
}

func (p *Plugin) handleUserDisconnect(w http.ResponseWriter, r *http.Request) {
	response := &model.PostActionIntegrationResponse{}
	decoder := json.NewDecoder(r.Body)
	postActionIntegrationRequest := &model.PostActionIntegrationRequest{}
	if err := decoder.Decode(&postActionIntegrationRequest); err != nil {
		p.API.LogError("Error decoding PostActionIntegrationRequest.", "Error", err.Error())
		p.returnPostActionIntegrationResponse(w, response)
		return
	}

	mattermostUserID := r.Header.Get(HeaderMattermostUserID)
	// Check if the user is connected to ServiceNow
	if _, err := p.GetUser(mattermostUserID); err != nil {
		if err != ErrNotFound {
			p.API.LogError("Error occurred while fetching user by ID. UserID: %s. Error: %s", mattermostUserID, err.Error())
		} else {
			var notConnectedPost *model.Post
			notConnectedPost, err = p.GetDisconnectUserPost(mattermostUserID, AlreadyDisconnectedMessage)
			if err != nil {
				p.API.LogError("Error occurred while creating user not connected post", "Error", err.Error())
			} else {
				response = &model.PostActionIntegrationResponse{
					Update: notConnectedPost,
				}
			}
		}
		p.returnPostActionIntegrationResponse(w, response)
		return
	}

	disconnectUser := postActionIntegrationRequest.Context[DisconnectUserContextName].(bool)
	if !disconnectUser {
		var rejectionPost *model.Post
		rejectionPost, err := p.GetDisconnectUserPost(mattermostUserID, DisconnectUserRejectedMessage)
		if err != nil {
			p.API.LogError("Error occurred while creating disconnect user rejection post.", "Error", err.Error())
		} else {
			response = &model.PostActionIntegrationResponse{
				Update: rejectionPost,
			}
		}
		p.returnPostActionIntegrationResponse(w, response)
		return
	}

	if err := p.DisconnectUser(mattermostUserID); err != nil {
		p.API.LogError("Error occurred while disconnecting user. UserID: %s. Error: %s", mattermostUserID, err.Error())
		p.returnPostActionIntegrationResponse(w, response)
		return
	}

	successPost, err := p.GetDisconnectUserPost(mattermostUserID, DisconnectUserSuccessMessage)
	if err != nil {
		p.API.LogError("Error occurred while creating disconnect user success post", "Error", err.Error())
	} else {
		response = &model.PostActionIntegrationResponse{
			Update: successPost,
		}
	}
	p.returnPostActionIntegrationResponse(w, response)
}

func (p *Plugin) handleSetDateTimeDialog(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	response := &model.PostActionIntegrationResponse{}
	postActionIntegrationRequest := &model.PostActionIntegrationRequest{}
	if err := decoder.Decode(&postActionIntegrationRequest); err != nil {
		p.API.LogError("Error decoding PostActionIntegrationRequest.", "Error", err.Error())
		p.handleAPIError(w, &serializer.APIErrorResponse{StatusCode: http.StatusBadRequest, Message: "Error in decoding PostActionIntegrationRequest."})
		p.returnPostActionIntegrationResponse(w, response)
		return
	}

	var elements []model.DialogElement
	date := model.DialogElement{
		DisplayName: "Date:",
		Name:        DateValue,
		Type:        "text",
		Placeholder: "YYYY-MM-DD",
		HelpText:    "Please enter the date in the format YYYY-MM-DD. Example: 2001-11-04",
		Optional:    false,
		MinLength:   10,
		MaxLength:   10,
	}

	time := model.DialogElement{
		DisplayName: "Time:",
		Name:        TimeValue,
		Type:        "text",
		Placeholder: "HH:MM",
		HelpText:    "Please enter the time in 24 hour format as HH:MM. Example: 20:04",
		Optional:    false,
		MinLength:   5,
		MaxLength:   5,
	}

	inputType := fmt.Sprintf("%v", postActionIntegrationRequest.Context[DateTimeDialogType])
	switch inputType {
	case DateUIType:
		elements = append(elements, date)
	case TimeUIType:
		elements = append(elements, time)
	case DateTimeUIType:
		elements = append(elements, date, time)
	}

	requestBody := model.OpenDialogRequest{
		TriggerId: postActionIntegrationRequest.TriggerId,
		URL:       fmt.Sprintf("%s%s", p.GetPluginURLPath(), PathSetDateTime),
		Dialog: model.Dialog{
			Title:       fmt.Sprintf("Set %s", inputType),
			CallbackId:  fmt.Sprintf("%s__%s", postActionIntegrationRequest.PostId, inputType),
			SubmitLabel: "Submit",
			Elements:    elements,
		},
	}

	ctx := r.Context()
	token := ctx.Value(ContextTokenKey).(*oauth2.Token)
	client := p.MakeClient(r.Context(), token)
	if err := client.OpenDialogRequest(&requestBody); err != nil {
		p.API.LogError("Error opening date-time selction dialog.", "Error", err.Error())
		p.handleAPIError(w, &serializer.APIErrorResponse{StatusCode: http.StatusInternalServerError, Message: "Error in opening date-time selection dialog."})
		return
	}
	p.returnPostActionIntegrationResponse(w, response)
}

func (p *Plugin) handleSetDateTime(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	response := &model.SubmitDialogResponse{}
	submitRequest := &model.SubmitDialogRequest{}
	if err := decoder.Decode(&submitRequest); err != nil {
		p.API.LogError("Error decoding SubmitDialogRequest.", "Error", err.Error())
		p.returnSubmitDialogResponse(w, response)
		return
	}

	ctx := r.Context()
	token := ctx.Value(ContextTokenKey).(*oauth2.Token)
	userID := r.Header.Get(HeaderServiceNowUserID)
	var selectedOption string

	if len(strings.Split(submitRequest.CallbackId, "__")) != 2 {
		p.API.LogError(InvalidCallbackIDError)
		response.Error = InvalidCallbackIDError
		p.returnSubmitDialogResponse(w, response)
		return
	}

	postID := strings.Split(submitRequest.CallbackId, "__")[0]
	inputType := strings.Split(submitRequest.CallbackId, "__")[1]

	var dateValidationError, timeValidationError string
	switch inputType {
	case DateTimeUIType:
		selectedOption = fmt.Sprintf("%v %v:00", submitRequest.Submission[DateValue], submitRequest.Submission[TimeValue])

		response.Errors = map[string]string{}

		dateValidationError = p.validateDate(fmt.Sprintf("%v", submitRequest.Submission[DateValue]))
		if dateValidationError != "" {
			response.Errors[DateValue] = dateValidationError
		}

		timeValidationError = p.validateTime(fmt.Sprintf("%v", submitRequest.Submission[TimeValue]))
		if timeValidationError != "" {
			response.Errors[TimeValue] = timeValidationError
		}
	case DateUIType:
		selectedOption = fmt.Sprintf("%v", submitRequest.Submission[DateValue])

		dateValidationError = p.validateDate(fmt.Sprintf("%v", submitRequest.Submission[DateValue]))
		if dateValidationError != "" {
			response.Errors = map[string]string{
				DateValue: dateValidationError,
			}
		}
	case TimeUIType:
		selectedOption = fmt.Sprintf("%v:00", submitRequest.Submission[TimeValue])

		timeValidationError = p.validateTime(fmt.Sprintf("%v", submitRequest.Submission[TimeValue]))

		if timeValidationError != "" {
			response.Errors = map[string]string{
				TimeValue: timeValidationError,
			}
		}
	}

	if dateValidationError != "" || timeValidationError != "" {
		p.returnSubmitDialogResponse(w, response)
		return
	}

	client := p.MakeClient(r.Context(), token)
	if err := client.SendMessageToVirtualAgentAPI(userID, selectedOption, true, &MessageAttachment{}); err != nil {
		p.API.LogError("Error sending message to VA.", "Error", err.Error())
		p.returnSubmitDialogResponse(w, response)
		return
	}

	newAttachment := []*model.SlackAttachment{}
	newAttachment = append(newAttachment, &model.SlackAttachment{
		Text:  fmt.Sprintf("You selected %s: %s", inputType, selectedOption),
		Color: updatedPostBorderColor,
	})

	newPost := &model.Post{
		Id:        postID,
		ChannelId: submitRequest.ChannelId,
		UserId:    p.botUserID,
	}

	model.ParseSlackAttachment(newPost, newAttachment)

	if _, appErr := p.API.UpdatePost(newPost); appErr != nil {
		p.API.LogError("Error updating the post.", "Error", appErr.Message)
		p.returnSubmitDialogResponse(w, response)
		return
	}

	p.returnSubmitDialogResponse(w, response)
}

func (p *Plugin) handlePickerSelection(w http.ResponseWriter, r *http.Request) {
	response := &model.PostActionIntegrationResponse{}
	decoder := json.NewDecoder(r.Body)
	postActionIntegrationRequest := &model.PostActionIntegrationRequest{}
	if err := decoder.Decode(&postActionIntegrationRequest); err != nil {
		p.API.LogError("Error decoding PostActionIntegrationRequest params.", "Error", err.Error())
		p.returnPostActionIntegrationResponse(w, response)
		return
	}

	ctx := r.Context()
	token := ctx.Value(ContextTokenKey).(*oauth2.Token)
	userID := r.Header.Get(HeaderServiceNowUserID)
	selectedOption := postActionIntegrationRequest.Context["selected_option"].(string)
	attachment := &MessageAttachment{}

	client := p.MakeClient(r.Context(), token)
	if err := client.SendMessageToVirtualAgentAPI(userID, selectedOption, true, attachment); err != nil {
		p.API.LogError("Error sending message to VA.", "Error", err.Error())
		p.returnPostActionIntegrationResponse(w, response)
		return
	}

	newAttachment := []*model.SlackAttachment{}
	newAttachment = append(newAttachment, &model.SlackAttachment{
		Text:  fmt.Sprintf("You selected: %s", selectedOption),
		Color: updatedPostBorderColor,
	})

	newPost := &model.Post{
		ChannelId: postActionIntegrationRequest.ChannelId,
		UserId:    p.botUserID,
	}

	model.ParseSlackAttachment(newPost, newAttachment)

	response = &model.PostActionIntegrationResponse{
		Update: newPost,
	}

	p.returnPostActionIntegrationResponse(w, response)
}

func (p *Plugin) handleVirtualAgentWebhook(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		p.API.LogError("Error occurred while reading webhook body.", "Error", err.Error())
		p.handleAPIError(w, &serializer.APIErrorResponse{StatusCode: http.StatusInternalServerError, Message: "Error occurred while reading webhook body."})
		return
	}

	if err = p.ProcessResponse(data); err != nil {
		p.API.LogError("Error occurred while processing response body.", "Error", err.Error())
		p.handleAPIError(w, &serializer.APIErrorResponse{StatusCode: http.StatusInternalServerError, Message: "Error occurred while processing response body."})
		return
	}
	ReturnStatusOK(w)
}

func (p *Plugin) returnPostActionIntegrationResponse(w http.ResponseWriter, res *model.PostActionIntegrationResponse) {
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(res.ToJson()); err != nil {
		p.API.LogWarn("Failed to write PostActionIntegrationResponse", "Error", err.Error())
	}
}

func (p *Plugin) returnSubmitDialogResponse(w http.ResponseWriter, res *model.SubmitDialogResponse) {
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(res.ToJson()); err != nil {
		p.API.LogWarn("Failed to write SubmitDialogResponse", "Error", err.Error())
	}
}
