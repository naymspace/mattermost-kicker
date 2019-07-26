package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	trigger = "kicker"
)

// KickerPlugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type KickerPlugin struct {
	plugin.MattermostPlugin
	botUserID string

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	enabled bool
}

// ServeHTTP demonstrates a plugin that handles HTTP requests by greeting the world.
func (p *KickerPlugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello, world!")
}

// See https://developers.mattermost.com/extend/plugins/server/reference/

// OnActivate ensures a configuration is set and initializes the API
func (p *KickerPlugin) OnActivate() error {
	p.enabled = true
	err := p.API.RegisterCommand(&model.Command{
		Trigger:          trigger,
		Description:      "TODO: describe me",
		DisplayName:      "TODO: Name me",
		AutoComplete:     true,
		AutoCompleteDesc: "TODO: AutoCompleteDesc",
		AutoCompleteHint: "TODO: AutoCompleteDesc",
	})
	if err != nil {
		return err
	}

	return nil
}

// OnDeactivate unregisters the command
func (p *KickerPlugin) OnDeactivate() error {
	p.enabled = false
	return nil
}

// ExecuteCommand returns a test string for now
func (p *KickerPlugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	if !p.enabled {
		return nil, appError("Cannot execute command while the plugin is disabled.", nil)
	}
	if p.API == nil {
		return nil, appError("Cannot access the plugin API.", nil)
	}
	if strings.HasPrefix(args.Command, "/"+trigger) {
		return p.executeCommand(args.Command)
	}

	return nil, appError("Command trigger "+args.Command+"is not supported by this plugin.", nil)
}

// executeCommand returns a sample text
func (p *KickerPlugin) executeCommand(command string) (*model.CommandResponse, *model.AppError) {
	text := "Blorg!"
	return &model.CommandResponse{ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, Text: text}, nil
}

func appError(message string, err error) *model.AppError {
	errorMessage := ""
	if err != nil {
		errorMessage = err.Error()
	}
	return model.NewAppError("Kicker Plugin", message, nil, errorMessage, http.StatusBadRequest)
}
