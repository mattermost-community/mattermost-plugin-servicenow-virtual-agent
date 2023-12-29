package plugin

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/bluele/gcache"
	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	client *pluginapi.Client

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	router *mux.Router
	// user ID of the bot account
	botUserID string

	store Store

	channelCache gcache.Cache
}

func (p *Plugin) OnActivate() error {
	if err := p.OnConfigurationChange(); err != nil {
		return err
	}

	p.client = pluginapi.NewClient(p.API, p.Driver)
	p.store = p.NewStore(p.API)

	if err := p.initBotUser(); err != nil {
		return err
	}

	p.router = p.initializeAPI()
	p.channelCache = gcache.New(p.getConfiguration().ChannelCacheSize).ARC().Build()
	return nil
}

func (p *Plugin) OnDeactivate() error {
	if p.channelCache != nil {
		p.channelCache.Purge()
	}

	return nil
}

func (p *Plugin) initBotUser() error {
	botID, err := p.client.Bot.EnsureBot(&model.Bot{
		Username:    BotUsername,
		DisplayName: BotDisplayName,
		Description: BotDescription,
	}, pluginapi.ProfileImagePath(filepath.Join("assets", "profile.png")))
	if err != nil {
		return errors.Wrap(err, "can't ensure bot")
	}

	p.botUserID = botID
	return nil
}

func (p *Plugin) GetSiteURL() string {
	return p.getConfiguration().MattermostSiteURL
}

func (p *Plugin) GetPluginURLPath() string {
	return "/plugins/" + manifest.ID + "/api/v1"
}

func (p *Plugin) GetPluginURL() string {
	return strings.TrimRight(p.GetSiteURL(), "/") + p.GetPluginURLPath()
}
