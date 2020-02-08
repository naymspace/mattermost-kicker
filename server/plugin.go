package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

// WantLevel defines how urgent a Player wants to play
type WantLevel int

const (
	trigger        = "kicker"
	botUserName    = "k̃̐̈̂҉̖̖̯̭͍̖i̾ͥc͋ke̺͉̻̰͆̉̀͊r̿͝ͅ"
	botDisplayName = "k̃̐̈̂҉̖̖̯̭͍̖i̾ͥc͋ke̺͉̻̰͆̉̀͊r̿͝ͅ"
	playerCount    = 4
	paramMaxHour   = 24
	paramMaxMinute = 60
	// warnDuration is used by a timer to notify, if there are not enough players
	warnDuration = time.Minute * time.Duration(15) // 15 Minutes
	// WLDecline means that this Player does not want to play
	WLDecline WantLevel = -1
	// WLVolunteer means that this Player wants to play only if there are not enough players
	WLVolunteer WantLevel = 0
	// WLParticipate means that this Player wants to play
	WLParticipate WantLevel = 1
)

// Player is the interface between Mattermost Users and a Kicker game
type Player struct {
	user      *model.User
	wantLevel WantLevel
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

	enabled      bool
	busy         bool
	pollPost     *model.Post
	cancelPost   *model.Post
	endTime      time.Time
	timer        *time.Timer
	timerWarning *time.Timer
	userID       string // user-ID of user who started a game
	channelID    string
	rootID       string

	participants []Player
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
		Description:      "",
		DisplayName:      botDisplayName,
		AutoComplete:     true,
		AutoCompleteDesc: "S͈̩͚t̼̲̟̱̟̮a̖͖r̲͍t͈̜̻̩e̩ͅt̯ ̱͇̹d͎̘e͇̗n͉̰ ̲̜̬̞̜͎" + botDisplayName + ", e̖̙̫.̤̭̼̙̟͔g̙̠̜̰̪͇̭.͙͇̼̗ /" + trigger + " -̝̟̼̖̻͍2̟14̼̱7̤̣̜ͅ4͍͍̮ͅ8̰͉͇̜̟̱3̳͇̪64̯̫̦͇͓8͉",
		AutoCompleteHint: "K̬̻̼̩̙I͎LL̞̮͙ ̭̱̟͕̖̘͇H̱̯Ṷ̪̝ͅM̠͈̠AN̩͇S̭̪̞̮",
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
	p.router.HandleFunc("/decline", p.DeclineHandler)
	p.router.HandleFunc("/cancel-game", p.CancelGameHandler)

	// serve static assets
	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		return appError("failed to get bundle path", err)
	}
	p.router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(http.Dir(filepath.Join(bundlePath, "assets")))))

	// initialize plugin
	p.enabled = true
	p.busy = false

	return nil
}

func (p *KickerPlugin) setUserWantLevel(userID string, wantLevel WantLevel) *model.AppError {
	// get user info from Mattermost API
	user, err := p.API.GetUser(userID)
	if err != nil {
		return appError("failed to get user data", err)
	}

	p.removeParticipantByID(user.Id)
	p.participants = append(p.participants, Player{
		user:      user,
		wantLevel: wantLevel,
	})

	p.updatePollPost()

	return nil
}

func (p *KickerPlugin) handleParticipationRequest(w http.ResponseWriter, r *http.Request, wantLevel WantLevel) {
	err := p.setUserWantLevel(r.Header.Get("Mattermost-User-Id"), wantLevel)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "{\"response\":\"Invalid User\"}\n")
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "{\"response\":\"OK\"}\n")
}

// ParticipateHandler handles participation requests
func (p *KickerPlugin) ParticipateHandler(w http.ResponseWriter, r *http.Request) {
	p.handleParticipationRequest(w, r, WLParticipate)
}

// VolunteerHandler handles volunteering requests
func (p *KickerPlugin) VolunteerHandler(w http.ResponseWriter, r *http.Request) {
	p.handleParticipationRequest(w, r, WLVolunteer)
}

// DeclineHandler handles declining request
func (p *KickerPlugin) DeclineHandler(w http.ResponseWriter, r *http.Request) {
	p.handleParticipationRequest(w, r, WLDecline)
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

		p.removePollPost()
		p.removeCancelPost()
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "{\"response\":\"OK\"}\n")
}

func (p *KickerPlugin) removeParticipantByID(id string) {
	var participants []Player
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

func (p *KickerPlugin) removePollPost() {
	p.API.DeletePost(p.pollPost.Id)
}

func (p *KickerPlugin) removeCancelPost() {
	p.API.DeleteEphemeralPost(p.userID, p.cancelPost.Id)
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
	sassyResponseText := fmt.Sprintf("![](%s/plugins/%s/assets/sassy.webp)", p.siteURL, manifest.ID)
	busyResponsetext := fmt.Sprintf("![](%s/plugins/%s/assets/busy.webp)", p.siteURL, manifest.ID)

	// check if kicker is busy
	if p.busy {
		return &model.CommandResponse{ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL, Text: busyResponsetext}, nil
	}

	// flag busy
	p.busy = true

	// clear participants
	p.participants = []Player{}

	// set user, channel and root ID
	p.userID = args.UserId
	p.channelID = args.ChannelId
	p.rootID = args.RootId

	// parse Args
	parsedArgs, parseError := ParseArgs(args.Command)
	if parseError != nil {
		p.busy = false
		return &model.CommandResponse{ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL, Text: sassyResponseText}, nil
	}
	// get the wait-duration until poll ends
	loc, _ := time.LoadLocation("Europe/Berlin")
	p.endTime = getEndTime(parsedArgs...)
	duration := p.endTime.Sub(time.Now().In(loc))
	warnDur := duration - warnDuration

	// if invalid, return sassy response
	if duration <= 0 {
		p.busy = false
		return &model.CommandResponse{ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL, Text: sassyResponseText}, nil
	}

	// Set timerWarning if we have at least 15 minutes before starting
	if warnDur > 0 {
		p.timerWarning = time.AfterFunc(warnDur, p.CheckEnoughPlayer)
	}

	// delay execution until endTime is reached
	p.timer = time.AfterFunc(duration, p.CreateEndPollPost)

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

	// create bot-post for canceling the poll (only visible to poll creator)
	cancelPost := &model.Post{
		UserId:    p.botUserID,
		ChannelId: p.channelID,
		Message:   "",
		RootId:    p.rootID,
		Type:      model.POST_DEFAULT,
	}
	model.ParseSlackAttachment(cancelPost, p.buildCancelGameAttachment())
	p.cancelPost = p.API.SendEphemeralPost(p.userID, cancelPost)

	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         "",
	}, nil
}

// ChoosePlayers returns 4 random Player (if possible).
// Participants are prefered over Volunteers.
func (p *KickerPlugin) ChoosePlayers() []Player {
	var returnPlayer []Player
	participants := p.GetParticipants()
	volunteers := p.GetVolunteers()

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

func (p *KickerPlugin) buildSlackAttachments() []*model.SlackAttachment {
	actions := []*model.PostAction{}

	actions = append(actions, &model.PostAction{
		Name: "O̤̅ͯ̅̉K̥͙̮͌ͧ̐͋̂",
		Type: model.POST_ACTION_TYPE_BUTTON,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/participate", p.siteURL, manifest.ID),
		},
	})

	actions = append(actions, &model.PostAction{
		Name: "N0̬͉̞̮̮T̳̼̮̙̙̺ ̻̺̹̬B̭̗̗̮1̻̗̘̞n4̤̳̬̣r̦͖̲Y͙",
		Type: model.POST_ACTION_TYPE_BUTTON,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/volunteer", p.siteURL, manifest.ID),
		},
	})

	actions = append(actions, &model.PostAction{
		Name: "C͔on͕̥̠̜͈5̜͙Ṷ̺̭m̩̖͙̥̝̻͓e͇͈̱̰ ͍̞͎̘͈S̟̥̩͕0̞uL",
		Type: model.POST_ACTION_TYPE_BUTTON,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/decline", p.siteURL, manifest.ID),
		},
	})

	return []*model.SlackAttachment{{
		AuthorName: botDisplayName,
		Title:      "D̻̻̞̖͖̝̘e̝͔̖͍̤̩r̖̻̞̱̞̬ " + botDisplayName + " h̺̮4̼̣͇t͚ ͎e͓u̼̫̻͍̝̤̫ch͕̝͎̰͇ g̬͔̲̠̖̻E͎͓f̼͚u̫̙͕̝̮ͅn̘̟d̥̫̤e͔̯n̥̯͉̻!̲͍͕̦̺͈ͅ ̲̹̮͎d̮̪0̜̞̰0M͎̩̭̺̻ ͙͓͔̹ḥ̪Ṳ̠̘m̰̦̠4̭̟͕̤̺n1͈͚̙t̘͎y̞̞͈͓̫̯̱!͎̤͈̣͔",
		Text:       fmt.Sprintf("E̻̤̗u̳̣r̭e̳̥͓̘̦͈ We̲̥̯̼̣l͎̯t͎̹͚ ̫̠̱̣e̤̜ͅnd͓̬̘̰ͅe̝̰̤̻̣̹̱t̹̺͇̱̘̹ ͇̞ṳ̮̰͍m̜ %02d:%02d G̲̥̭̳̼̗̩T̰M̘̯̖͓̺̠+̮̜0̯̖͖̹͚̳1", p.endTime.Hour(), p.endTime.Minute()),
		Actions:    actions,
	}, p.buildParticipantsAttachment()}
}

func (p *KickerPlugin) buildParticipantsAttachment() *model.SlackAttachment {
	participants := p.GetParticipants()
	volunteers := p.GetVolunteers()
	decliners := p.GetDecliners()

	if len(participants) == 0 && len(volunteers) == 0 && len(decliners) == 0 {
		return nil
	}

	text := ""

	if len(participants) > 0 {
		text += "🤬̭̭̻̪͚̣��͙: " + JoinPlayerNames(participants) + "\n"
	}

	if len(volunteers) > 0 {
		text += "�͚̤�͚̳̟̯̳̣💩̜̲͖: " + JoinPlayerNames(volunteers) + "\n"
	}

	if len(decliners) > 0 {
		text += "�̹͇͉͙͎�̻͈̳☠̱̰̭ͅ: " + JoinPlayerNames(decliners) + "\n"
	}

	return &model.SlackAttachment{
		Text: text,
	}
}

func (p *KickerPlugin) buildCancelGameAttachment() []*model.SlackAttachment {
	actions := []*model.PostAction{}

	actions = append(actions, &model.PostAction{
		Name: "Y̫̙͕0̥̭͉͈͓u̥̯̮͕̩ ̪̣̘̬͔c͎̺4̖n͔͍͚͈'̫͔̬̭͇̬̲T̤̗̝͍̬̬ ͅs͓̩͓͓͍̙t͚̼̟̭̺0̝͇̝P̜̲̪̬̭͉ ṃ̗͎3̮",
		Type: model.POST_ACTION_TYPE_BUTTON,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/cancel-game", p.siteURL, manifest.ID),
		},
	})

	return []*model.SlackAttachment{{
		AuthorName: botDisplayName,
		Title:      "W̬h̺̦̟͈̝0͎̻̞ ̲̘̠͉̻͚s͙̜̳̪ṵ̱̥͇͇̮M̞̘̣̻̰ṃ̯̪̬̝0̹̮̹̝̫̟n3̗̖̯̙͚̳̹d̘̼̼̣̰̰͕ ̜̺̗̻m̥̞͔̼ͅE̱͖?͓",
		Text:       "1l͓͔ͅ ̭͖͇͉͉̳h͚̘͎̣ͅ0p̬̪̭̻͕E 1͕͎̺̲5̩̜̭̙̪̘̗ g̲̠̲̻̪o̰̬̮̤͕̺N̙͕e͖͎̞̫̲̠👹͓̠̳̹̘",
		Actions:    actions,
	}}
}

// GetParticipants returns all Players with the "participant" want level
func (p *KickerPlugin) GetParticipants() []Player {
	return p.filterParticipantsByWantlevel(WLParticipate)
}

// GetVolunteers returns all Players with the "volunteer" want level
func (p *KickerPlugin) GetVolunteers() []Player {
	return p.filterParticipantsByWantlevel(WLVolunteer)
}

// GetDecliners returns all Players with the "decline" want level
func (p *KickerPlugin) GetDecliners() []Player {
	return p.filterParticipantsByWantlevel(WLDecline)
}

func (p *KickerPlugin) filterParticipantsByWantlevel(wantLevel WantLevel) []Player {
	var players []Player

	for _, player := range p.participants {
		if player.wantLevel == wantLevel {
			players = append(players, player)
		}
	}

	return players
}

// CreateEndPollPost creates a post with the result of selected players
func (p *KickerPlugin) CreateEndPollPost() {
	p.removePollPost()
	p.removeCancelPost()

	chosenPlayer := p.ChoosePlayers()
	// not enough player
	if len(chosenPlayer) < playerCount {
		p.API.CreatePost(&model.Post{
			UserId:    p.botUserID,
			ChannelId: p.channelID,
			Message: `
			W͎͍͕̖R̹̟̰̈́͒͛̊͗Y̸̠̯̣̲̘̞̫̑͛͆ͥ̂́̂Y̳͔͉͎ͦͦ̑ͥ̆͆̍Ỳ̮͔͈̹̓̓̊͠Y̻̽̔͌ͣYYͭ̌ͤ̏ͦ͏̗̹̙̤̥̥̗Y̟͞Y͈̜̳̾͂͆Y̞͍̦͖̠̽̽ͧͬͯ̃̂Y̭ͨYͥ̄̆͛́Y͕͍̯̥̮͓͇Y͕̙̺̫͕̰ͮ́̿ͅŸ̲̬̩̺́̃̉Y̪̜͉̼̪̫ͣͦͯ̒̑ͩ̚?̦̟͚͉̪̐ͬ͟!͌͏͙͔?͈͓͚̤̟͓ͧͯ̾̃!͕̪̲̝͖ͬ͗̉ͥͯͥ͡?͕̥̯̖̠͓ͫͮ?̶̤̩͍̤̲̝̈́͋̍ͅ!̝̱͚̭̫̺̄̇͡!́͊̅̒̃̀",
			`,
			RootId: p.rootID,
			Type:   model.POST_DEFAULT,
		})
		p.busy = false
		return
	}

	message := "F̸͍̻͍͆̽R̪̻͔̼͚̽̾E̗̗̽̈̍ͥE̳ͤ̎͌̅̅̆̑ ̴͍̰̦̓̀ͪ͛̉ͅḿ͖̟̭̘̞3͔͔̞̓̽ͭ͗ͅ: " + JoinPlayerNames(chosenPlayer)

	p.API.CreatePost(&model.Post{
		UserId:    p.botUserID,
		ChannelId: p.channelID,
		Message:   message,
		RootId:    p.rootID,
		Type:      model.POST_DEFAULT,
	})

	p.busy = false
}

// CheckEnoughPlayer creates a warning post, if we do not have enough players.
func (p *KickerPlugin) CheckEnoughPlayer() {
	players := p.ChoosePlayers()

	if len(players) < playerCount {
		p.API.CreatePost(&model.Post{
			UserId:    p.botUserID,
			ChannelId: p.channelID,
			Message: `
W̷̭̭̩͇͊ͫ̉͛͌ͦͬ́̕̕̕Rͮͭ͑͡͏̱̬̣͚̪͓̜̠̖̯Y̻̥̳̯͓̘̯͎͈̳̺̜͂̀̒ͧ͢͡Ŷ̨̜̲̝̱͇͕̠̰̰̟͉͙͓͙͔̼̠͇̓͗̃̓̍ͤ̑͋͐̋ͤ͑̄ͮͮ̚͘͢͢͡Ý̸̧̞͍͓͙̥͔̭̫̯̣̗̝̝̉͒͑̅̌́͘͡Y̆̏̾̉̃̾̊̓̎̏̈̇͒̓ͬ͏̥̱̦͓͚͘͜Y̸̠̥̭̳̙̜̖̗͒̀̉̓ͥ͂̋̾ͧ́͆͜Y̨̯̤̫̲̖̝̩ͧͦ͐̏̃ͮ̀̀ͅȲͬ̊̾҉̵̩̹̝̺̥̱͈̖͎Y̓̑̾́̏̆̔҉̴̢̫̟̞̱͇̫͇̥̩̗͙͘͟Ÿ͍̤̙̹̹̞̘̣͈͇̥̠̘̹̰ͬ͊̉̎ͦ͊̂̓ͥ͜͡ͅY̷̸̡̗̤̯̺̻̪̗̦͔̝̺̰̊̿͒ͥ̂͛ͤ̈́̒̊͐͑̀ͦ͆ͫ͋̍͢͞Y̶̨̲̬̼͚̥͖̳͕͕̭͉̊ͮ̐̓̀̏̀̈̿͒̌ͤ̾́̍ͪͧ̂́Y̡̲̺̱̟͓͇̝͖̌ͪ̈ͪ͒̇ͪͧ̌͆̓ͪͬͪ͌̍́̌̕̕͞ͅŸ́ͬ̐ͥ͛ͯ̂ͬ̃ͯ͛̄̀̚͘͏̘͓̖̗̗͔̬̫̞̥͙̼̤̱͕̠̯Y̍̾̀ͤ̐ͧ͐҉̶̧̬̬̥̦̘̼͘͝Y̻̤͉̝̖͊̋̃̍̈̓̽͐̇̔̔̊̔̚̕͞?̴̢̘̙͖̖̙̟̻̟̠̳̯̺̖͈̞͖̘͈͙͋̋ͩͮ̐ͤ̓̆ͥ͆̋ͥ̃͆ͩͧ̚!̶͍̮̥̟̦͙͖̮͉̤̤͚̍ͪ̂̏͛ͤ̐̔́̕?̴̨̤̻͖͎͉̤̬͎͚̞̳̗̳ͦ͋̓͗͊ͭ͗ͫ͂͟͠͞!̷̸̢͍̩̝̬̝̹̣͓͕̝̪̫̘̠̒̏̀̏̿͌ͅ?̳̤̘̱̗͔̻͊̿͂̌̍̓̂̄ͯ͂͂́̇́̚͜?̸̨̲͎͉̺̗͕͈̱͉̫̭͕̘͕̝̃̀͂̎̋͋͒̉ͨͮ̓̏ͯͩ̀͘ͅ!̶̶̢̡̙̻͎̱͔͔̜̈́ͪ̀̀̚ͅ!̙̱̟̦̮̩̆ͯ̅̅̔̿ͥ̀ͯͪ̿̀͘̕͟ͅ
			`,
			RootId: p.rootID,
			Type:   model.POST_DEFAULT,
		})
	}
}
