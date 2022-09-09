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

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

type Post struct {
	Body string `json:"body"`
}

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
	apiRouter.HandleFunc(PathDateTimeSelectionDialog, p.checkOAuth(p.handleDateTimeSelectionDialog)).Methods(http.MethodPost)
	apiRouter.HandleFunc(PathDateTimeSelection, p.checkOAuth(p.handleDateTimeSelection)).Methods(http.MethodPost)
	apiRouter.HandleFunc(PathVirtualAgentWebhook, p.checkAuthBySecret(p.handleVirtualAgentWebhook)).Methods(http.MethodPost)
	r.Handle("{anything:.*}", http.NotFoundHandler())

	return r
}

func (p *Plugin) checkAuthBySecret(handleFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if status, err := verifyHTTPSecret(p.getConfiguration().WebhookSecret, r.FormValue("secret")); err != nil {
			p.API.LogError("Invalid secret", "Error", err.Error())
			http.Error(w, fmt.Sprintf("Invalid Secret. Error: %s", err.Error()), status)
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
			http.Error(w, "Not authorized", http.StatusUnauthorized)
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

func (p *Plugin) handleDateTimeSelectionDialog(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	postActionIntegrationRequest := &model.PostActionIntegrationRequest{}
	if err := decoder.Decode(&postActionIntegrationRequest); err != nil {
		p.API.LogError("Error decoding PostActionIntegrationRequest.", "Error", err.Error())
		http.Error(w, "Error decoding PostActionIntegrationRequest.", http.StatusBadRequest)
		return
	}

	var elements []model.DialogElement

	date := model.DialogElement{
		DisplayName: "Date:",
		Name:        "date",
		Type:        "text",
		Placeholder: "YYYY-MM-DD",
		HelpText:    "Please enter the date in the format YYYY-MM-DD. Example: 2001-11-04",
		Optional:    false,
		MinLength:   10,
		MaxLength:   10,
	}

	time := model.DialogElement{
		DisplayName: "Time:",
		Name:        "time",
		Type:        "text",
		Placeholder: "HH:MM",
		HelpText:    "Please enter the time in 24 hour format as HH:MM. Example: 20:04",
		Optional:    false,
		MinLength:   5,
		MaxLength:   5,
	}

	inputType := fmt.Sprintf("%v", postActionIntegrationRequest.Context["type"])
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
		URL:       fmt.Sprintf("%s%s", p.GetPluginURLPath(), PathDateTimeSelection),
		Dialog: model.Dialog{
			Title:       fmt.Sprintf("Select %s", inputType),
			CallbackId:  fmt.Sprintf("%s__%s", postActionIntegrationRequest.PostId, inputType),
			SubmitLabel: "Submit",
			Elements:    elements,
		},
	}

	p.OpenDialogRequest(w, requestBody)

	response := &model.PostActionIntegrationResponse{}
	p.returnPostActionIntegrationResponse(w, response)
}

func (p *Plugin) handleDateTimeSelection(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	response := &model.SubmitDialogResponse{}
	submitRequest := &model.SubmitDialogRequest{}
	if err := decoder.Decode(&submitRequest); err != nil {
		p.API.LogError("Error decoding SubmitDialogRequest params.", "Error", err.Error())
		p.returnSubmitDialogResponse(w, response)
		return
	}

	ctx := r.Context()
	token := ctx.Value(ContextTokenKey).(*oauth2.Token)
	userID := r.Header.Get(HeaderServiceNowUserID)
	var selectedOption string

	if len(strings.Split(submitRequest.CallbackId, "__")) != 2 {
		p.API.LogError("Invalid callback ID.")
		response.Error = "Invalid callback ID"
		p.returnSubmitDialogResponse(w, response)
		return
	}

	postID := strings.Split(submitRequest.CallbackId, "__")[0]
	inputType := strings.Split(submitRequest.CallbackId, "__")[1]

	switch inputType {
	case DateTimeUIType:
		selectedOption = fmt.Sprintf("%v %v:00", submitRequest.Submission["date"], submitRequest.Submission["time"])

		dateValidationError := p.validateDate(fmt.Sprintf("%v", submitRequest.Submission["date"]))

		response.Errors = map[string]string{}

		if dateValidationError != "" {
			response.Errors["date"] = dateValidationError
		}

		timeValidationError := p.validateTime(fmt.Sprintf("%v", submitRequest.Submission["time"]))

		if timeValidationError != "" {
			response.Errors["time"] = timeValidationError
		}

		if dateValidationError != "" || timeValidationError != "" {
			p.returnSubmitDialogResponse(w, response)
			return
		}
	case DateUIType:
		selectedOption = fmt.Sprintf("%v", submitRequest.Submission["date"])

		dateValidationError := p.validateDate(fmt.Sprintf("%v", submitRequest.Submission["date"]))

		if dateValidationError != "" {
			response.Errors = map[string]string{
				"date": dateValidationError,
			}
			p.returnSubmitDialogResponse(w, response)
			return
		}
	case TimeUIType:
		selectedOption = fmt.Sprintf("%v:00", submitRequest.Submission["time"])

		timeValidationError := p.validateTime(fmt.Sprintf("%v", submitRequest.Submission["time"]))

		if timeValidationError != "" {
			response.Errors = map[string]string{
				"time": timeValidationError,
			}
			p.returnSubmitDialogResponse(w, response)
			return
		}
	}

	client := p.MakeClient(r.Context(), token)
	if err := client.SendMessageToVirtualAgentAPI(userID, selectedOption, true); err != nil {
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

	client := p.MakeClient(r.Context(), token)
	if err := client.SendMessageToVirtualAgentAPI(userID, selectedOption, true); err != nil {
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
		http.Error(w, "Error occurred while reading webhook body.", http.StatusInternalServerError)
		return
	}

	if err = p.ProcessResponse(data); err != nil {
		p.API.LogError("Error occurred while processing response body.", "Error", err.Error())
		http.Error(w, "Error occurred while processing response body.", http.StatusInternalServerError)
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
