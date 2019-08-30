package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	trigger              = "kicker"
	botUserName          = "kicker"
	botDisplayName       = "kicker"
	playerCount          = 4
	wantLevelParticipant = 1
	wantLevelVolunteer   = 0
)

type player struct {
	user      *model.User
	wantLevel int
}

// KickerPlugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type KickerPlugin struct {
	plugin.MattermostPlugin
	botUserID string
	router    *mux.Router

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	enabled bool
	busy    bool

	participants []player
}

// ServeHTTP demonstrates a plugin that handles HTTP requests by greeting the world.
func (p *KickerPlugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.router.ServeHTTP(w, r)
}

// See https://developers.mattermost.com/extend/plugins/server/reference/

// OnActivate ensures a configuration is set and initializes the API
func (p *KickerPlugin) OnActivate() error {
	p.enabled = true
	err := p.API.RegisterCommand(&model.Command{
		Trigger:          trigger,
		Description:      "TODO: describe me",
		DisplayName:      "Kicker BOT",
		AutoComplete:     true,
		AutoCompleteDesc: "Startet den Kicker BOT, e.g. /kicker 12 30",
		AutoCompleteHint: "[hour] [minute]",
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

	// setup routing
	p.router = mux.NewRouter()
	p.router.HandleFunc("/participate", p.ParticipateHandler)
	p.router.HandleFunc("/volunteer", p.VolunteerHandler)
	p.router.HandleFunc("/delete-participation", p.DeleteParticipationHandler)
	p.busy = false
	return nil
}

// ParticipateHandler handles participation requests. orly?
// https://i.kym-cdn.com/photos/images/masonry/000/004/734/348799688_l.jpg
func (p *KickerPlugin) ParticipateHandler(w http.ResponseWriter, r *http.Request) {
	// vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	// fmt.Fprintf(w, "Category: %v\n", vars["category"])
	p.API.LogDebug("ParticipateHandler called", "user_id", r.Header.Get("Mattermost-User-Id"))

	// out, _ := json.Marshal(r)
	// p.API.LogDebug("ParticipateHandler called", "r", string(out))

	// get user info from Mattermost API
	user, _ := p.API.GetUser(r.Header.Get("Mattermost-User-Id"))
	p.API.LogDebug("ParticipateHandler", "user.Username", user.Username)

	p.participants = append(p.participants, player{
		user:      user,
		wantLevel: wantLevelParticipant,
	})

	fmt.Fprintf(w, "{\"response\":\"OKAY\"}\n")
}

// VolunteerHandler handles volunteering requests. ORLY?!
func (p *KickerPlugin) VolunteerHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: DRY
	// get user info from Mattermost API
	user, _ := p.API.GetUser(r.Header.Get("Mattermost-User-Id"))
	p.API.LogDebug("VolunteerHandler", "user.Username", user.Username)

	p.participants = append(p.participants, player{
		user:      user,
		wantLevel: wantLevelVolunteer,
	})

	fmt.Fprintf(w, "{\"response\":\"OKAY\"}\n")
}

// DeleteParticipationHandler handles deleting participation. YES FCKING RLY
func (p *KickerPlugin) DeleteParticipationHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := p.API.GetUser(r.Header.Get("Mattermost-User-Id"))
	p.API.LogDebug("DeleteParticipationHandler", "user.Username", user.Username)

	for index, participant := range p.participants {
		if user.Id == participant.user.Id {
			p.participants = remove(p.participants, index)
		}
	}
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
	// Log Command:
	// p.API.LogInfo(args.Command, nil)
	responseText := "Der Kicker wurde gestartet."
	sassyResponseText := "![](https://media3.giphy.com/media/utmZFnsMhUHqU/giphy.gif?cid=790b76115d3b59e1417459456b2425e4&rid=giphy.gif)"
	busyResponsetext := "![](https://media3.giphy.com/media/cOFLK7ZbliXW21RfmE/giphy.gif?cid=790b7611f21be7df606604f241cada0852c238865fccab98&rid=giphy.gif)"
	// check if kicker is busy
	if p.busy {
		return &model.CommandResponse{ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL, Text: busyResponsetext}, nil
	}

	// flag busy
	p.busy = true

	// clear participants
	p.participants = []player{}

	// parse Args
	parsedArgs := parseArgs(args.Command)

	// get the wait-duration until poll ends
	loc, _ := time.LoadLocation("Europe/Berlin")
	endTime := getEndTime(parsedArgs...)
	duration := endTime.Sub(time.Now().In(loc))

	// if invalid, return sassy response
	if duration <= 0 {
		p.busy = false
		return &model.CommandResponse{ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL, Text: sassyResponseText}, nil
	}

	// create bot-post for ending the poll
	createEndPollPost := func() {
		chosenPlayer := choosePlayer(p.participants)
		// not enough player
		if len(chosenPlayer) < playerCount {
			message := "Nicht genug Spieler, es wollen spielen: "
			for _, player := range chosenPlayer {
				message += player.user.Username + ", "
				// TODO: do we want to show the wantLevel?
			}
			p.API.CreatePost(&model.Post{
				UserId:    p.botUserID,
				ChannelId: args.ChannelId,
				Message:   message,
				RootId:    args.RootId,
				Type:      model.POST_DEFAULT,
			})
			p.busy = false
			return
		}

		message := "Es nehmen teil: "
		for _, element := range chosenPlayer {
			message += element.user.Username + ", "
		}

		p.API.CreatePost(&model.Post{
			UserId:    p.botUserID,
			ChannelId: args.ChannelId,
			Message:   message,
			RootId:    args.RootId,
			Type:      model.POST_DEFAULT,
		})

		p.busy = false
	}
	// delay execution until endTime is reached
	time.AfterFunc(duration, createEndPollPost)

	// create bot-post for initiating the poll
	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: args.ChannelId,
		Message:   "",
		RootId:    args.RootId,
		Type:      model.POST_DEFAULT,
	}
	model.ParseSlackAttachment(post, p.buildSlackAttachments(endTime))
	p.API.CreatePost(post)

	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         responseText,
	}, nil
}

// TODO: DRY
func choosePlayer(all []player) []player {
	var returnPlayer []player
	var participants []player
	var volunteers []player
	participantCount := 0
	volunteerCount := 0
	for index, element := range all {
		if element.wantLevel > 0 {
			participantCount++
			participants = append(participants, all[index])
		} else {
			volunteerCount++
			volunteers = append(volunteers, all[index])
		}
	}

	if participantCount+volunteerCount < playerCount {
		// not enough player! return all that wanted to play:
		for _, volunteer := range volunteers {
			returnPlayer = append(returnPlayer, volunteer)
		}
		for _, participant := range participants {
			returnPlayer = append(returnPlayer, participant)
		}
		return returnPlayer
	}

	if participantCount >= playerCount {
		// enough participants
		for i := 0; i < playerCount; i++ {
			// generate seed depending on server-time
			rand.Seed(time.Now().UnixNano())
			// generate index within length of participants
			randIndex := rand.Intn(len(participants))
			participant := participants[randIndex]
			returnPlayer = append(returnPlayer, participant)
			participants = remove(participants, randIndex)
		}
	} else {
		// not enough Pṕarticipants
		// take all participants
		for _, participant := range participants {
			returnPlayer = append(returnPlayer, participant)
		}
		restPlayerCount := playerCount - len(returnPlayer)
		for i := 0; i < restPlayerCount; i++ {
			rand.Seed(time.Now().UnixNano())
			randIndex := rand.Intn(len(volunteers))
			volunteer := volunteers[randIndex]
			returnPlayer = append(returnPlayer, volunteer)
			participants = remove(volunteers, randIndex)
		}

	}
	return returnPlayer
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

/*
  parses given args
  takes the first 3 arguments:
  - the command "kicker" itself
  - a given hour
  - a given minute
*/
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

/*
	func getStartMessage(endTime time.Time) string {
		// TODO: Needs random selected messages
		message := "Ich starte um %02d:%02d Uhr, ihr Maden. Also seht zu oder verreckt an Bewegungsmangel."
		return fmt.Sprintf(message, endTime.Hour(), endTime.Minute())
	}
*/

func (p *KickerPlugin) buildSlackAttachments(endTime time.Time) []*model.SlackAttachment {
	actions := []*model.PostAction{}

	actions = append(actions, &model.PostAction{
		Name: "Bin dabei",
		Type: model.POST_ACTION_TYPE_BUTTON,
		Integration: &model.PostActionIntegration{
			// TODO: where to get these values?
			URL: fmt.Sprintf("%s/plugins/%s/participate", "http://localhost:8065", "com.naymspace.mattermost-kicker"),
		},
	})

	actions = append(actions, &model.PostAction{
		Name: "Wenn sich sonst keiner traut 🤷",
		Type: model.POST_ACTION_TYPE_BUTTON,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/volunteer", "http://localhost:8065", "com.naymspace.mattermost-kicker"),
		},
	})

	actions = append(actions, &model.PostAction{
		Name: "Teilnahme zurückziehen",
		Type: model.POST_ACTION_TYPE_BUTTON,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/delete-participation", "http://localhost:8065", "com.naymspace.mattermost-kicker"),
		},
	})

	return []*model.SlackAttachment{{
		AuthorName: "kicker BOT",
		Title:      "Der kicker-BOT hat euch herausgefordert! Wer möchte teilnehmen?",
		Text:       fmt.Sprintf("Kickern startet um %02d:%02d Uhr.", endTime.Hour(), endTime.Minute()),
		Actions:    actions,
	}}
}

func appError(message string, err error) *model.AppError {
	errorMessage := ""
	if err != nil {
		errorMessage = err.Error()
	}
	return model.NewAppError("Kicker Plugin", message, nil, errorMessage, http.StatusBadRequest)
}

// remove element from array
// TODO: Move this to a utility-class
// from https://stackoverflow.com/questions/37334119/how-to-delete-an-element-from-a-slice-in-golang
func remove(s []player, i int) []player {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
