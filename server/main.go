package main

import (
	"github.com/mattermost/mattermost-plugin-servicenow-virtual-agent/server/plugin"

	mmplugin "github.com/mattermost/mattermost/server/public/plugin"
)

func main() {
	mmplugin.ClientMain(&plugin.Plugin{})
}
