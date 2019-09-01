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

	enabled  bool
	busy     bool
	pollPost *model.Post
	endTime  time.Time

	participants []player
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

	// initialize plugin
	p.enabled = true
	p.busy = false

	return nil
}

// ParticipateHandler handles participation requests
func (p *KickerPlugin) ParticipateHandler(w http.ResponseWriter, r *http.Request) {
	// get user info from Mattermost API
	user, _ := p.API.GetUser(r.Header.Get("Mattermost-User-Id"))

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

	var participants []player
	for _, participant := range p.participants {
		if user.Id != participant.user.Id {
			participants = append(participants, participant)
		}
	}
	p.participants = participants

	p.updatePollPost()

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "{\"response\":\"OK\"}\n")
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
	p.endTime = getEndTime(parsedArgs...)
	duration := p.endTime.Sub(time.Now().In(loc))

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
			p.API.CreatePost(&model.Post{
				UserId:    p.botUserID,
				ChannelId: args.ChannelId,
				Message:   "Nicht genug Spieler!",
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
	model.ParseSlackAttachment(post, p.buildSlackAttachments())
	p.pollPost, _ = p.API.CreatePost(post)

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
		if element.wantLevel == wantLevelParticipant {
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
		// not enough participants
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

func (p *KickerPlugin) buildSlackAttachments() []*model.SlackAttachment {
	actions := []*model.PostAction{}

	actions = append(actions, &model.PostAction{
		Name: "Bin dabei 👍",
		Type: model.POST_ACTION_TYPE_BUTTON,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("plugins/%s/participate", manifest.ID),
		},
	})

	actions = append(actions, &model.PostAction{
		Name: "Wenn sich sonst keiner traut 👉",
		Type: model.POST_ACTION_TYPE_BUTTON,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("plugins/%s/volunteer", manifest.ID),
		},
	})

	actions = append(actions, &model.PostAction{
		Name: "Och nö 👎",
		Type: model.POST_ACTION_TYPE_BUTTON,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("plugins/%s/delete-participation", manifest.ID),
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
