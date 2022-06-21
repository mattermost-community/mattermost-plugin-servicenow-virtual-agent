package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-plugin-api/experimental/command"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/pkg/errors"
)

const helpTextHeader = "###### Mattermost ServiceNow Virtual Agent Plugin - Slash Command Help\n"

const commonHelpText = "\n" +
	"* `/servicenow-va help` - Launch the ServiceNow Virtual Agent plugin command line help syntax\n"

var serviceNowVACommandHandler = CommandHandler{
	handlers: map[string]CommandHandlerFunc{
		"help": executeHelp,
	},
	defaultHandler: executeDefault,
}

type CommandHandlerFunc func(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse

type CommandHandler struct {
	handlers       map[string]CommandHandlerFunc
	defaultHandler CommandHandlerFunc
}

func (ch CommandHandler) Handle(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	for n := len(args); n > 0; n-- {
		h := ch.handlers[strings.Join(args[:n], "/")]
		if h != nil {
			return h(p, c, header, args[n:]...)
		}
	}
	return ch.defaultHandler(p, c, header, args...)
}

func executeHelp(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	return p.help(header)
}

// executeDefault is the default command if no other command fits. It defaults to help.
func executeDefault(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	return p.help(header)
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, commandArgs *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	args := strings.Fields(commandArgs.Command)
	if len(args) == 0 || args[0] != "/servicenow-va" {
		return p.help(commandArgs), nil
	}
	return serviceNowVACommandHandler.Handle(p, c, commandArgs, args[1:]...), nil
}

func (p *Plugin) getCommand(config *configuration) (*model.Command, error) {
	iconData, err := command.GetIconData(p.API, "assets/icon.svg")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get icon data")
	}

	return &model.Command{
		Trigger:              "servicenow-va",
		AutoComplete:         true,
		AutoCompleteDesc:     "Available commands: help",
		AutoCompleteHint:     "[command]",
		AutocompleteData:     getAutocompleteData(config),
		AutocompleteIconData: iconData,
	}, nil
}

func (p *Plugin) help(args *model.CommandArgs) *model.CommandResponse {
	p.postCommandResponse(args, fmt.Sprintf("%s%s", helpTextHeader, commonHelpText))
	return &model.CommandResponse{}
}

func (p *Plugin) postCommandResponse(args *model.CommandArgs, text string) {
	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: args.ChannelId,
		RootId:    args.RootId,
		Message:   text,
	}
	_ = p.API.SendEphemeralPost(args.UserId, post)
}

func getAutocompleteData(config *configuration) *model.AutocompleteData {
	serviceNowVA := model.NewAutocompleteData("servicenow-va", "[command]", "Available commands: help")

	help := model.NewAutocompleteData("help", "", "Display ServiceNow Virtual Agent Plugin Help.")
	serviceNowVA.AddCommand(help)

	return serviceNowVA
}
