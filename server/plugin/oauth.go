package plugin

import (
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
)

func (p *Plugin) httpOAuth2Connect(w http.ResponseWriter, r *http.Request) {
	mattermostUserID := r.Header.Get("Mattermost-User-ID")
	if mattermostUserID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	redirectURL, err := p.InitOAuth2(mattermostUserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (p *Plugin) httpOAuth2Complete(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "missing authorization state", http.StatusBadRequest)
		return
	}

	mattermostUserID := r.Header.Get(HeaderMattermostUserID)
	if mattermostUserID == "" {
		http.Error(w, "not authorized", http.StatusUnauthorized)
		return
	}

	err := p.CompleteOAuth2(mattermostUserID, code, state)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
