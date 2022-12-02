package plugin

import (
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/serializer"
	"golang.org/x/oauth2"
)

func (p *Plugin) httpOAuth2Connect(w http.ResponseWriter, r *http.Request) {
	mattermostUserID := r.Header.Get(HeaderMattermostUserID)
	if mattermostUserID == "" {
		p.handleAPIError(w, &serializer.APIErrorResponse{StatusCode: http.StatusUnauthorized, Message: NotAuthorizedError})
		return
	}

	redirectURL, err := p.InitOAuth2(mattermostUserID)
	if err != nil {
		p.handleAPIError(w, &serializer.APIErrorResponse{StatusCode: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (p *Plugin) httpOAuth2Complete(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		p.handleAPIError(w, &serializer.APIErrorResponse{StatusCode: http.StatusBadRequest, Message: "Missing authorization code"})
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" {
		p.handleAPIError(w, &serializer.APIErrorResponse{StatusCode: http.StatusBadRequest, Message: "Missing authorization state"})
		return
	}

	mattermostUserID := r.Header.Get(HeaderMattermostUserID)
	if mattermostUserID == "" {
		p.handleAPIError(w, &serializer.APIErrorResponse{StatusCode: http.StatusUnauthorized, Message: NotAuthorizedError})
		return
	}

	err := p.CompleteOAuth2(mattermostUserID, code, state)
	if err != nil {
		p.handleAPIError(w, &serializer.APIErrorResponse{StatusCode: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	html := `
<!DOCTYPE html>
<html>
	<head>
		<script>
			window.close();
		</script>
	</head>
	<body>
		<p>Completed connecting to ServiceNow. Please close this window.</p>
	</body>
</html>
`

	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(html)); err != nil {
		p.handleAPIError(w, &serializer.APIErrorResponse{StatusCode: http.StatusInternalServerError, Message: err.Error()})
	}
}

func (p *Plugin) NewOAuth2Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.getConfiguration().ServiceNowOAuthClientID,
		ClientSecret: p.getConfiguration().ServiceNowOAuthClientSecret,
		RedirectURL:  fmt.Sprintf("%s%s", p.GetPluginURL(), PathOAuth2Complete),
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("%s/oauth_auth.do", p.getConfiguration().ServiceNowURL),
			TokenURL: fmt.Sprintf("%s/oauth_token.do", p.getConfiguration().ServiceNowURL),
		},
	}
}
