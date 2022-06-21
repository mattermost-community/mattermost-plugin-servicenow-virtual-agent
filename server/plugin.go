package main

import (
	"path/filepath"
	"sync"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/pkg/errors"
)

const (
	botUsername    = "servicenow-virtual-agent"
	botDisplayName = "ServiceNow Virtual Agent Plugin"
	botDescription = "A bot account created by the plugin ServiceNow Virtual Agent."
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	router *mux.Router
	// user ID of the bot account
	botUserID string
}

func (p *Plugin) OnActivate() error {
	if err := p.OnConfigurationChange(); err != nil {
		return err
	}

	if err := p.initBotUser(); err != nil {
		return err
	}

	p.router = p.initializeAPI()

	return nil
}

func (p *Plugin) initBotUser() error {
	botID, err := p.Helpers.EnsureBot(&model.Bot{
		Username:    botUsername,
		DisplayName: botDisplayName,
		Description: botDescription,
	}, plugin.ProfileImagePath(filepath.Join("assets", "profile.png")))
	if err != nil {
		return errors.Wrap(err, "can't ensure bot")
	}

	p.botUserID = botID
	return nil
}
