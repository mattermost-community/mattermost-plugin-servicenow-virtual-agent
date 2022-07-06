package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"runtime/debug"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
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
	apiRouter.HandleFunc(PathActionOptions, p.checkAuth(p.handlePickerSelection)).Methods(http.MethodPost)
	apiRouter.HandleFunc(PathVirtualAgentWebhook, p.handleVirtualAgentWebhook).Methods(http.MethodPost)
	r.Handle("{anything:.*}", http.NotFoundHandler())

	return r
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
					"url", r.URL.String(),
					"error", x,
					"stack", string(debug.Stack()))
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

func (p *Plugin) handleUserDisconnect(w http.ResponseWriter, r *http.Request) {
	response := &model.PostActionIntegrationResponse{}
	decoder := json.NewDecoder(r.Body)
	postActionIntegrationRequest := &model.PostActionIntegrationRequest{}
	if err := decoder.Decode(&postActionIntegrationRequest); err != nil {
		p.API.LogError("error decoding PostActionIntegrationRequest params.", "error", err.Error())
		p.returnPostActionIntegrationResponse(w, response)
		return
	}

	mattermostUserID := r.Header.Get(HeaderMattermostUserID)
	// Check if the user is connected to ServiceNow
	_, err := p.GetUser(mattermostUserID)
	if err != nil {
		if err != ErrNotFound {
			p.API.LogError("error occurred while fetching user by ID. UserID: %s. Error: %s", mattermostUserID, err.Error())
		} else {
			var notConnectedPost *model.Post
			notConnectedPost, err = p.GetDisconnectUserPost(mattermostUserID, AlreadyDisconnectedMessage)
			if err != nil {
				p.API.LogError("error occurred while creating user not connected post", "error", err.Error())
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
		rejectionPost, err = p.GetDisconnectUserPost(mattermostUserID, DisconnectUserRejectedMessage)
		if err != nil {
			p.API.LogError("error occurred while creating disconnect user rejection post.", "error", err.Error())
		} else {
			response = &model.PostActionIntegrationResponse{
				Update: rejectionPost,
			}
		}
		p.returnPostActionIntegrationResponse(w, response)
		return
	}

	if err = p.DisconnectUser(mattermostUserID); err != nil {
		p.API.LogError("error occurred while disconnecting user. UserID: %s. Error: %s", mattermostUserID, err.Error())
		p.returnPostActionIntegrationResponse(w, response)
		return
	}

	successPost, err := p.GetDisconnectUserPost(mattermostUserID, DisconnectUserSuccessMessage)
	if err != nil {
		p.API.LogError("error occurred while creating disconnect user success post", "error", err.Error())
	} else {
		response = &model.PostActionIntegrationResponse{
			Update: successPost,
		}
	}
	p.returnPostActionIntegrationResponse(w, response)
}

func (p *Plugin) handlePickerSelection(w http.ResponseWriter, r *http.Request) {
	response := &model.PostActionIntegrationResponse{}
	decoder := json.NewDecoder(r.Body)
	postActionIntegrationRequest := &model.PostActionIntegrationRequest{}
	if err := decoder.Decode(&postActionIntegrationRequest); err != nil {
		p.API.LogError("Error decoding PostActionIntegrationRequest params.", "error", err.Error())
		p.returnPostActionIntegrationResponse(w, response)
		return
	}

	mattermostUserID := r.Header.Get(HeaderMattermostUserID)
	user, err := p.store.LoadUser(mattermostUserID)
	if err != nil {
		p.API.LogError("Error loading user from KV store.", "error", err.Error())
	}

	token, err := p.ParseAuthToken(user.OAuth2Token)
	if err != nil {
		p.API.LogError("Error parsing OAuth2 token.", "error", err.Error())
		return
	}

	ctx := context.Background()
	client := p.MakeClient(ctx, token)
	err = client.SendMessageToVirtualAgentAPI(user.UserID, postActionIntegrationRequest.Context["selected_option"].(string), false)
	if err != nil {
		p.API.LogError("Error sending message to virtual agent API.", "error", err.Error())
	}
}

func (p *Plugin) handleVirtualAgentWebhook(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		p.API.LogError("error occurred while reading webhook body.", "error", err.Error())
		http.Error(w, "Error occurred while reading webhook body.", http.StatusInternalServerError)
		return
	}
	if data == nil {
		return
	}
	err = p.ProcessResponse(data)
	if err != nil {
		p.API.LogError("error occurred while processing response body.", "error", err.Error())
		http.Error(w, "Error occurred while processing response body.", http.StatusInternalServerError)
		return
	}
	ReturnStatusOK(w)
}

func (p *Plugin) returnPostActionIntegrationResponse(w http.ResponseWriter, res *model.PostActionIntegrationResponse) {
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(res.ToJson()); err != nil {
		p.API.LogWarn("failed to write PostActionIntegrationResponse", "Error", err.Error())
	}
}
