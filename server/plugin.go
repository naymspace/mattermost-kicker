package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	trigger        = "kicker"
	botUserName    = "Kicker"
	botDisplayName = "Kicker"
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

	// Init bot
	bot := &model.Bot{
		Username:    botUserName,
		DisplayName: botDisplayName,
	}

	botUserID, appErr := p.Helpers.EnsureBot(bot)
	if appErr != nil {
		return appError("Failed to ensure bot user", nil)
	}

	p.botUserID = botUserID

	if err = p.setProfileImage(); err != nil {
		return appError("failed to set profile image", err)
	}

	return nil
}

// OnDeactivate unregisters the command
func (p *KickerPlugin) OnDeactivate() error {
	p.enabled = false
	return nil
}

// setProfileImage set the profile image of the bot account
func (p *KickerPlugin) setProfileImage() error {
	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		return appError("failed to get bundle path", err)
	}

	profileImage, err := ioutil.ReadFile(filepath.Join(bundlePath, "assets", "logo.png"))
	if err != nil {
		return appError("failed to read profile image", err)
	}
	if appErr := p.API.SetProfileImage(p.botUserID, profileImage); appErr != nil {
		return appError("failed to set profile image", appErr)
	}
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
		return p.executeCommand(args)
	}

	return nil, appError("Command trigger "+args.Command+"is not supported by this plugin.", nil)
}

// executeCommand returns a sample text
func (p *KickerPlugin) executeCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	p.API.LogInfo(args.Command, nil)
	text := "Großer Kicker, erhöre mein flehen. Sag uns, wer soll zum Kickertisch gehen!"
	textNo := "![](https://media3.giphy.com/media/utmZFnsMhUHqU/giphy.gif?cid=790b76115d3b59e1417459456b2425e4&rid=giphy.gif)"
	botText := "Nö."
	parsedArgs := parseArgs(args.Command)
	// get time to start
	loc, _ := time.LoadLocation("Europe/Berlin")
	endTime := getEndTime(parsedArgs...)
	duration := endTime.Sub(time.Now().In(loc))

	if duration <= 0 {
		return &model.CommandResponse{ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, Text: textNo}, nil
	}

	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: args.ChannelId,
		Message:   botText,
		RootId:    args.RootId,
		Type:      model.POST_DEFAULT,
	}

	createStartMessage := func() {
		p.API.CreatePost(post)
	}

	time.AfterFunc(duration, createStartMessage)

	return &model.CommandResponse{ResponseType: model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, Text: text}, nil
}

func getEndTime(params ...int) time.Time {
	// default values
	hour, minute := 12, 0

	if len(params) == 2 {
		hour, minute = params[0], params[1]
	}

	if len(params) == 1 {
		hour = params[0]
	}
	loc, _ := time.LoadLocation("Europe/Berlin")

	n := time.Now()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, loc).Add(time.Hour * time.Duration(hour)).Add(time.Minute * time.Duration(minute))
}

func parseArgs(args string) []int {
	str := strings.SplitN(args, " ", 3)

	if len(str) == 3 {
		i1, err1 := strconv.Atoi(str[1])
		i2, err2 := strconv.Atoi(str[2])
		if err1 != nil || err2 != nil {
			return []int{}
		}
		return []int{
			i1,
			i2,
		}
	}

	if len(str) == 2 {
		i1, err1 := strconv.Atoi(str[1])
		if err1 != nil {
			return []int{}
		}
		return []int{
			i1,
		}
	}

	return []int{}
}

func appError(message string, err error) *model.AppError {
	errorMessage := ""
	if err != nil {
		errorMessage = err.Error()
	}
	return model.NewAppError("Kicker Plugin", message, nil, errorMessage, http.StatusBadRequest)
}
