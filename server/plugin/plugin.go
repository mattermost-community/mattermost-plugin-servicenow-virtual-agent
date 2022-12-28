package plugin

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bluele/gcache"
	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-plugin-api/cluster"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/pkg/errors"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin
	backgroundJob *cluster.Job

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

func (p *Plugin) closeBackgroundJob() {
	if err := p.backgroundJob.Close(); err != nil {
		p.API.LogError("Failed to close background job", "Error", err.Error())
	}
}

func (p *Plugin) deactivateJob() {
	if p.backgroundJob != nil {
		p.closeBackgroundJob()
		if err := p.API.KVDelete(cronPrefix + PublishSeriveNowVAIsTypingJobName); err != nil {
			p.API.LogError("Failed to delete the job", "Error", err.Error())
		}
	}
}

func (p *Plugin) ScheduleJob(mattermostUserID string) error {
	interval := *p.API.GetConfig().ServiceSettings.TimeBetweenUserTypingUpdatesMilliseconds / 1000

	// Close the previous background job if exist.
	p.deactivateJob()

	channel, err := p.API.GetDirectChannel(mattermostUserID, p.botUserID)
	if err != nil {
		p.API.LogError("Couldn't get bot's DM channel", "UserID", mattermostUserID, "Error", err.Error())
		return err
	}

	intervalInSecond := time.Duration(interval) * time.Second

	// cluster.Schedule creates a scheduled job and stores job metadata in kv store using key "cron_<jobName>"
	job, cronErr := cluster.Schedule(
		p.API,
		PublishSeriveNowVAIsTypingJobName,
		cluster.MakeWaitForRoundedInterval(intervalInSecond),
		func() {
			if err = p.API.PublishUserTyping(p.botUserID, channel.Id, ""); err != nil {
				p.API.LogDebug("Failed to publish a user is typing WebSocket event", "Error", err.Error())
			}
		},
	)
	if cronErr != nil {
		p.API.LogError("Error while scheduling a job", "Error", err.Error())
		return cronErr
	}

	p.backgroundJob = job
	return nil
}

func (p *Plugin) initBotUser() error {
	botID, err := p.Helpers.EnsureBot(&model.Bot{
		Username:    BotUsername,
		DisplayName: BotDisplayName,
		Description: BotDescription,
	}, plugin.ProfileImagePath(filepath.Join("assets", "profile.png")))
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
