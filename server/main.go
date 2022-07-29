package main

import (
	"github.com/Brightscout/mattermost-plugin-servicenow-virtual-agent/server/plugin"

	mmplugin "github.com/mattermost/mattermost-server/v5/plugin"
)

func main() {
	mmplugin.ClientMain(&plugin.Plugin{})
}
