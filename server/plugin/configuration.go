package plugin

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

// configuration captures the plugin's external configuration as exposed in the Mattermost server
// configuration, as well as values computed from the configuration. Any public fields will be
// deserialized from the Mattermost server configuration in OnConfigurationChange.
//
// As plugins are inherently concurrent (hooks being called asynchronously), and the plugin
// configuration can change at any time, access to the configuration must be synchronized. The
// strategy used in this plugin is to guard a pointer to the configuration, and clone the entire
// struct whenever it changes. You may replace this with whatever strategy you choose.
//
// If you add non-reference types to your configuration struct, be sure to rewrite Clone as a deep
// copy appropriate for your types.
type configuration struct {
	ServiceNowURL                string `json:"ServiceNowURL"`
	ServiceNowOAuthClientID      string `json:"ServiceNowOAuthClientID"`
	ServiceNowOAuthClientSecret  string `json:"ServiceNowOAuthClientSecret"`
	EncryptionSecret             string `json:"EncryptionSecret"`
	WebhookSecret                string `json:"WebhookSecret"`
	ChannelCacheSize             int    `json:"ChannelCacheSize"`
	AttachmentLinkExpirationTime int    `json:"AttachmentLinkExpirationTime"`
	MattermostSiteURL            string
	PluginID                     string
	PluginURL                    string
	PluginURLPath                string
}

// Clone shallow copies the configuration. Your implementation may require a deep copy if
// your configuration has reference types.
func (c *configuration) Clone() *configuration {
	var clone = *c
	return &clone
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (p *Plugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &configuration{}
	}

	return p.configuration
}

// setConfiguration replaces the active configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex is not
// reentrant. In particular, avoid using the plugin API entirely, as this may in turn trigger a
// hook back into the plugin. If that hook attempts to acquire this lock, a deadlock may occur.
//
// This method panics if setConfiguration is called with the existing configuration. This almost
// certainly means that the configuration was modified without being cloned and may result in
// an unsafe access.
func (p *Plugin) setConfiguration(configuration *configuration) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if configuration != nil && p.configuration == configuration {
		// Ignore assignment if the configuration struct is empty. Go will optimize the
		// allocation for same to point at the same memory address, breaking the check
		// above.
		if reflect.ValueOf(*configuration).NumField() == 0 {
			return
		}

		panic("setConfiguration called with the existing configuration")
	}

	p.configuration = configuration
}

// IsValid checks if all needed fields are set.
func (c *configuration) IsValid() error {
	if c.ServiceNowURL == "" {
		return fmt.Errorf(EmptyServiceNowURLErrorMessage)
	}
	if c.ServiceNowOAuthClientID == "" {
		return fmt.Errorf(EmptyServiceNowOAuthClientIDErrorMessage)
	}
	if c.ServiceNowOAuthClientSecret == "" {
		return fmt.Errorf(EmptyServiceNowOAuthClientSecretErrorMessage)
	}
	if c.EncryptionSecret == "" {
		return fmt.Errorf(EmptyEncryptionSecretErrorMessage)
	}
	if c.WebhookSecret == "" {
		return fmt.Errorf(EmptyWebhookSecretErrorMessage)
	}
	if c.ChannelCacheSize <= 0 {
		return fmt.Errorf(InvalidChannelCacheSizeErrorMessage)
	}
	if c.AttachmentLinkExpirationTime <= 0 {
		return fmt.Errorf(InvalidAttachmentLinkExpirationTimeErrorMessage)
	}
	return nil
}

func (c *configuration) sanitize() {
	c.ServiceNowURL = strings.TrimRight(strings.TrimSpace(c.ServiceNowURL), "/")
	c.ServiceNowOAuthClientID = strings.TrimSpace(c.ServiceNowOAuthClientID)
	c.ServiceNowOAuthClientSecret = strings.TrimSpace(c.ServiceNowOAuthClientSecret)
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	var configuration = new(configuration)

	// Load the public configuration fields from the Mattermost server configuration.
	if err := p.API.LoadPluginConfiguration(configuration); err != nil {
		return errors.Wrap(err, "failed to load plugin configuration")
	}

	configuration.sanitize()

	mattermostSiteURL := p.API.GetConfig().ServiceSettings.SiteURL
	if mattermostSiteURL == nil {
		return errors.New("plugin requires Mattermost Site URL to be set")
	}

	configuration.MattermostSiteURL = *mattermostSiteURL
	configuration.PluginURL = p.GetPluginURL()
	configuration.PluginURLPath = p.GetPluginURLPath()
	configuration.PluginID = manifest.ID

	p.setConfiguration(configuration)

	return nil
}
