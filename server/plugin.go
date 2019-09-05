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
	botDisplayName       = "kicker BOT"
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

	enabled   bool
	busy      bool
	pollPost  *model.Post
	endTime   time.Time
	timer     *time.Timer
	userID    string // user-ID of user who started a game
	channelID string
	rootID    string

	participants []player
	siteURL      string
}

// ServeHTTP delegates routing to the mux Router, which is configured in OnActivate
func (p *KickerPlugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.router.ServeHTTP(w, r)
}

// See https://developers.mattermost.com/extend/plugins/server/reference/

// OnActivate registers a command and a bot, sets up routing, and initializes the plugin
func (p *KickerPlugin) OnActivate() error {
	err := p.API.RegisterCommand(&model.Command{
		Trigger:          trigger,
		Description:      "TODO: describe me",
		DisplayName:      botDisplayName,
		AutoComplete:     true,
		AutoCompleteDesc: "Startet den " + botDisplayName + ", e.g. /" + trigger + " 12 30",
		AutoCompleteHint: "[hour] [minute]",
	})
	if err != nil {
		return err
	}

	// Get siteURL from config
	config := p.API.GetConfig()
	p.siteURL = *config.ServiceSettings.SiteURL

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
	p.router.HandleFunc("/cancel-game", p.CancelGameHandler)

	// initialize plugin
	p.enabled = true
	p.busy = false

	return nil
}

// ParticipateHandler handles participation requests
func (p *KickerPlugin) ParticipateHandler(w http.ResponseWriter, r *http.Request) {
	// get user info from Mattermost API
	user, _ := p.API.GetUser(r.Header.Get("Mattermost-User-Id"))

	p.removeParticipantByID(user.Id)
	p.participants = append(p.participants, player{
		user:      user,
		wantLevel: wantLevelParticipant,
	})

	p.updatePollPost()

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "{\"response\":\"OK\"}\n")
}

// VolunteerHandler handles volunteering requests
func (p *KickerPlugin) VolunteerHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: DRY
	// get user info from Mattermost API
	user, _ := p.API.GetUser(r.Header.Get("Mattermost-User-Id"))

	p.removeParticipantByID(user.Id)
	p.participants = append(p.participants, player{
		user:      user,
		wantLevel: wantLevelVolunteer,
	})

	p.updatePollPost()

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "{\"response\":\"OK\"}\n")
}

// DeleteParticipationHandler handles deleting participation request
func (p *KickerPlugin) DeleteParticipationHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := p.API.GetUser(r.Header.Get("Mattermost-User-Id"))

	p.removeParticipantByID(user.Id)

	p.updatePollPost()

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "{\"response\":\"OK\"}\n")
}

// CancelGameHandler handles canceling game requests
func (p *KickerPlugin) CancelGameHandler(w http.ResponseWriter, r *http.Request) {
	user, err := p.API.GetUser(r.Header.Get("Mattermost-User-Id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "{\"response\":\"Invalid User\"}\n")
		return
	}

	if user.Id != p.userID {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "{\"response\":\"Not Authorized\"}\n")
		return
	}

	if p.busy {
		p.timer.Stop()
		p.busy = false

		p.API.CreatePost(&model.Post{
			UserId:    p.botUserID,
			ChannelId: p.channelID,
			Message:   "Bot wurde gestoppt!",
			RootId:    p.rootID,
			Type:      model.POST_DEFAULT,
		})
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "{\"response\":\"OK\"}\n")
}

func (p *KickerPlugin) removeParticipantByID(id string) {
	var participants []player
	for _, participant := range p.participants {
		if id != participant.user.Id {
			participants = append(participants, participant)
		}
	}
	p.participants = participants
}

func (p *KickerPlugin) updatePollPost() {
	model.ParseSlackAttachment(p.pollPost, p.buildSlackAttachments())
	p.pollPost, _ = p.API.UpdatePost(p.pollPost)
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

// ExecuteCommand checks the given command and passes it to executeCommand if valid
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

// executeCommand checks the given arguments and the internal state, and returns the according message
func (p *KickerPlugin) executeCommand(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
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

	// set user, channel and root ID
	p.userID = args.UserId
	p.channelID = args.ChannelId
	p.rootID = args.RootId

	// parse Args
	parsedArgs := parseArgs(args.Command)

	// get the wait-duration until poll ends
	loc, _ := time.LoadLocation("Europe/Berlin")
	p.endTime = getEndTime(parsedArgs...)
	duration := p.endTime.Sub(time.Now().In(loc))

	// if invalid, return sassy response
	if duration <= 0 {
		p.busy = false
		return &model.CommandResponse{ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL, Text: sassyResponseText}, nil
	}

	// create bot-post for ending the poll
	createEndPollPost := func() {
		chosenPlayer := p.choosePlayers()
		// not enough player
		if len(chosenPlayer) < playerCount {
			p.API.CreatePost(&model.Post{
				UserId:    p.botUserID,
				ChannelId: p.channelID,
				Message:   "Nicht genug Spieler!",
				RootId:    p.rootID,
				Type:      model.POST_DEFAULT,
			})
			p.busy = false
			return
		}

		message := "Es nehmen teil: " + p.joinPlayers(chosenPlayer)

		p.API.CreatePost(&model.Post{
			UserId:    p.botUserID,
			ChannelId: p.channelID,
			Message:   message,
			RootId:    p.rootID,
			Type:      model.POST_DEFAULT,
		})

		p.busy = false
	}
	// delay execution until endTime is reached
	p.timer = time.AfterFunc(duration, createEndPollPost)

	// create bot-post for initiating the poll
	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: p.channelID,
		Message:   "",
		RootId:    p.rootID,
		Type:      model.POST_DEFAULT,
	}
	model.ParseSlackAttachment(post, p.buildSlackAttachments())
	p.pollPost, _ = p.API.CreatePost(post)

	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         "",
		Attachments:  p.buildCancelGameAttachment(),
	}, nil
}

// TODO: DRY
func (p *KickerPlugin) choosePlayers() []player {
	var returnPlayer []player
	participants := p.getParticipants()
	volunteers := p.getVolunteers()

	if len(participants)+len(volunteers) < playerCount {
		// not enough players! return all that wanted to play
		return append(participants, volunteers...)
	}

	// generate seed depending on server-time
	rand.Seed(time.Now().UnixNano())

	if len(participants) >= playerCount {
		// enough participants
		for i := 0; i < playerCount; i++ {
			// add random participants
			randIndex := rand.Intn(len(participants))
			returnPlayer = append(returnPlayer, participants[randIndex])
			participants = remove(participants, randIndex)
		}
	} else {
		// not enough participants
		// take all participants
		returnPlayer = append(returnPlayer, participants...)
		// add random volunteers
		restPlayerCount := playerCount - len(returnPlayer)
		for i := 0; i < restPlayerCount; i++ {
			randIndex := rand.Intn(len(volunteers))
			returnPlayer = append(returnPlayer, volunteers[randIndex])
			volunteers = remove(volunteers, randIndex)
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

func (p *KickerPlugin) buildSlackAttachments() []*model.SlackAttachment {
	actions := []*model.PostAction{}

	actions = append(actions, &model.PostAction{
		Name: "Bin dabei 👍",
		Type: model.POST_ACTION_TYPE_BUTTON,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/participate", p.siteURL, manifest.ID),
		},
	})

	actions = append(actions, &model.PostAction{
		Name: "Wenn sich sonst keiner traut 👉",
		Type: model.POST_ACTION_TYPE_BUTTON,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/volunteer", p.siteURL, manifest.ID),
		},
	})

	actions = append(actions, &model.PostAction{
		Name: "Och nö 👎",
		Type: model.POST_ACTION_TYPE_BUTTON,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/delete-participation", p.siteURL, manifest.ID),
		},
	})

	return []*model.SlackAttachment{{
		AuthorName: botDisplayName,
		Title:      "Der " + botDisplayName + " hat euch herausgefordert! Wer möchte teilnehmen?",
		Text:       fmt.Sprintf("Kickern startet um %02d:%02d Uhr.", p.endTime.Hour(), p.endTime.Minute()),
		Actions:    actions,
	}, p.buildParticipantsAttachment()}
}

func (p *KickerPlugin) buildParticipantsAttachment() *model.SlackAttachment {
	participants := p.getParticipants()
	volunteers := p.getVolunteers()

	if len(participants) == 0 && len(volunteers) == 0 {
		return nil
	}

	text := ""

	if len(participants) > 0 {
		text += "👍: " + p.joinPlayers(participants) + "\n"
	}

	if len(volunteers) > 0 {
		text += "👉: " + p.joinPlayers(volunteers) + "\n"
	}

	return &model.SlackAttachment{
		Text: text,
	}
}

func (p *KickerPlugin) buildCancelGameAttachment() []*model.SlackAttachment {
	actions := []*model.PostAction{}

	actions = append(actions, &model.PostAction{
		Name: "Stop Bot",
		Type: model.POST_ACTION_TYPE_BUTTON,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/cancel-game", p.siteURL, manifest.ID),
		},
	})

	return []*model.SlackAttachment{{
		AuthorName: botDisplayName,
		Title:      "Der Kicker wurde gestartet.",
		Text:       "Zum stoppen kannst du diesen Button benutzen:",
		Actions:    actions,
	}}
}

func (p *KickerPlugin) getParticipants() []player {
	var players []player

	for index, element := range p.participants {
		if element.wantLevel == wantLevelParticipant {
			players = append(players, p.participants[index])
		}
	}

	return players
}

func (p *KickerPlugin) getVolunteers() []player {
	var players []player

	for index, element := range p.participants {
		if element.wantLevel == wantLevelVolunteer {
			players = append(players, p.participants[index])
		}
	}

	return players
}

func (p *KickerPlugin) joinPlayers(players []player) string {
	result := ""
	for index, element := range players {
		result += element.user.Username
		if index+1 < len(players) {
			result += ", "
		}
	}
	return result
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
