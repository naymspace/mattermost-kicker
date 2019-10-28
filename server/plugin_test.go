package main

import (
	"testing"

	"github.com/mattermost/mattermost-server/model"
)

var horst = &Player{
	user: &model.User{
		Username: "horst",
		Id:       "1",
	},
	wantLevel: WLParticipate,
}

var baerbel = &Player{
	user: &model.User{
		Username: "bärbel",
		Id:       "2",
	},
	wantLevel: WLParticipate,
}

var etienne = &Player{
	user: &model.User{
		Username: "etienne",
		Id:       "3",
	},
	wantLevel: WLParticipate,
}

var ingebork = &Player{
	user: &model.User{
		Username: "ingebork",
		Id:       "4",
	},
	wantLevel: WLParticipate,
}

var kay = &Player{
	user: &model.User{
		Username: "kay",
		Id:       "5",
	},
	wantLevel: WLVolunteer,
}

var oke = &Player{
	user: &model.User{
		Username: "oke",
		Id:       "6",
	},
	wantLevel: WLVolunteer,
}

var mable = &Player{
	user: &model.User{
		Username: "mable",
		Id:       "7",
	},
	wantLevel: WLVolunteer,
}

var uwe = &Player{
	user: &model.User{
		Username: "uwe",
		Id:       "8",
	},
	wantLevel: WLVolunteer,
}

var dieder = &Player{
	user: &model.User{
		Username: "dieder",
		Id:       "9",
	},
	wantLevel: WLDecline,
}

func SetupTestKickerPlugin(player []Player) *KickerPlugin {
	return &KickerPlugin{
		participants: player,
	}
}

func TestGetParticipants(t *testing.T) {
	p := SetupTestKickerPlugin([]Player{*horst, *baerbel, *kay})

	players := p.GetParticipants()

	if len(players) != 2 {
		t.Errorf("Number of participants was incorrect, got: %d, want: %d", len(players), 2)
	}

	if players[0].user.Username != "horst" {
		t.Errorf("First participant name was incorrect, got: %s, want: %s", players[0].user.Username, "horst")
	}

	if players[1].user.Username != "bärbel" {
		t.Errorf("Second participant name was incorrect, got: %s, want: %s", players[1].user.Username, "bärbel")
	}
}

func TestGetVolunteers(t *testing.T) {
	p := SetupTestKickerPlugin([]Player{*horst, *baerbel, *kay})

	players := p.GetVolunteers()

	if len(players) != 1 {
		t.Errorf("Number of volunteers was incorrect, got: %d, want: %d", len(players), 1)
	}

	if players[0].user.Username != "kay" {
		t.Errorf("First volunteer name was incorrect, got: %s, want: %s", players[0].user.Username, "kay")
	}
}

func TestGetDecliners(t *testing.T) {
	p := SetupTestKickerPlugin([]Player{*horst, *baerbel, *dieder})

	players := p.GetDecliners()

	if len(players) != 1 {
		t.Errorf("Number of decliners was incorrect, got: %d, want: %d", len(players), 1)
	}

	if players[0].user.Username != "dieder" {
		t.Errorf("First decliner name was incorrect, got: %s, want: %s", players[0].user.Username, "dieder")
	}
}

func TestJoinPlayerNames(t *testing.T) {
	tables := []struct {
		Players []Player
		Result  string
	}{
		{
			Players: []Player{},
			Result:  "",
		},
		{
			Players: []Player{*baerbel},
			Result:  "bärbel",
		},
		{
			Players: []Player{*horst, *baerbel},
			Result:  "horst, bärbel",
		},
	}

	for _, table := range tables {
		r := JoinPlayerNames(table.Players)
		if r != table.Result {
			t.Errorf("Concatenated usernames were incorrect, got: %s, want: %s", r, table.Result)
		}
	}
}

func sliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func TestParseArgs(t *testing.T) {
	errorTables := []struct {
		Args string
	}{
		{
			Args: "",
		},
		{
			Args: "/kicker pfnort",
		},
		{
			Args: "/kicker pfnort troz",
		},
		{
			Args: "/kicker pfnort 12",
		},
		{
			Args: "/kicker 12 pfnort",
		},
		{
			Args: "/kicker 12 pfnort",
		},
		{
			Args: "/kicker 12 20 pfnort",
		},
		{
			Args: "/kicker 1220",
		},
		{
			Args: "/kicker 24",
		},
		{
			Args: "/kicker 24 1",
		},
		{
			Args: "/kicker 24 60",
		},
		{
			Args: "/kicker 24 -1",
		},
		{
			Args: "/kicker -1",
		},
		{
			Args: "/kicker -1 1",
		},
		{
			Args: "/kicker -1 60",
		},
		{
			Args: "/kicker -1 -1",
		},
		{
			Args: "/kicker 12 60",
		},
		{
			Args: "/kicker 12 -1",
		},
		{
			Args: "/kicker 12.5 0",
		},
		{
			Args: "/kicker 1.8446744e+19 0",
		},
	}

	for _, table := range errorTables {
		ints, err := ParseArgs(table.Args)
		if len(ints) != 0 || err == nil {
			t.Errorf("Error handling was incorrect for args: '%s', parsed ints should be empty, was: %d, err should not be nil, was: nil", table.Args, ints)
		}
	}

	successTables := []struct {
		Args   string
		Result []int
	}{
		{
			Args:   "/kicker",
			Result: []int{},
		},
		{
			Args:   "/kicker 12",
			Result: []int{12},
		},
		{
			Args:   "/kicker 0",
			Result: []int{0},
		},
		{
			Args:   "/kicker 23",
			Result: []int{23},
		},
		{
			Args:   "/kicker 12 30",
			Result: []int{12, 30},
		},
		{
			Args:   "/kicker 12 0",
			Result: []int{12, 0},
		},
		{
			Args:   "/kicker 12 59",
			Result: []int{12, 59},
		},
	}

	for _, table := range successTables {
		ints, err := ParseArgs(table.Args)
		if !sliceEqual(ints, table.Result) || err != nil {
			t.Errorf("Success handling was incorrect for args: '%s', parsed ints should be: %d, was: %d, err should be nil, was: %s", table.Args, table.Result, ints, err)
		}
	}
}

func TestChoosePlayers(t *testing.T) {
	singleResultTables := []struct {
		Players []Player
		Result  []Player
	}{
		// Test with 3 or less player, should return same players again
		{
			Players: []Player{*horst, *baerbel, *kay},
			Result:  []Player{*kay, *horst, *baerbel},
		},
		// Test with 4 participants and 1 volunteer, should return 4 player, which we know
		{
			Players: []Player{*horst, *baerbel, *etienne, *ingebork, *kay},
			Result:  []Player{*horst, *baerbel, *etienne, *ingebork},
		},
	}

	for _, table := range singleResultTables {
		player := SetupTestKickerPlugin(table.Players).ChoosePlayers()
		if !playerEqual(player, table.Result) {
			t.Errorf("ChoosePlayers returns unexpected results")
		}
	}

	// Since ChoosePlayer create randIndexes, we need to check multiple possible results
	multiResultTables := []struct {
		Players []Player
		Results [][]Player
	}{
		// Test with 3 participants and 2 volunteer, should return 4 player, 3 of which we know
		// But the fourth is decided by coinflip (kay || oke), so we need to check both
		{
			Players: []Player{*horst, *baerbel, *etienne, *kay, *oke},
			Results: [][]Player{{*horst, *baerbel, *etienne, *kay}, {*horst, *baerbel, *etienne, *oke}},
		},
		// Test with 1 participants and 4 volunteer, should return 4 player, 1 of which we know
		// But the other are decided random, so we need to check 4 cases.
		{
			Players: []Player{*horst, *kay, *oke, *mable, *uwe},
			Results: [][]Player{{*horst, *kay, *oke, *mable},
				{*horst, *kay, *oke, *uwe},
				{*horst, *kay, *mable, *uwe},
				{*horst, *oke, *mable, *uwe}},
		},
	}

	for _, table := range multiResultTables {
		player := SetupTestKickerPlugin(table.Players).ChoosePlayers()
		oneResultOccured := false
		for _, result := range table.Results {
			if playerEqual(player, result) {
				oneResultOccured = true
				break
			}
		}
		if !oneResultOccured {
			t.Errorf("ChoosePlayers returns unexpected results")
		}
	}
}

func TestFilterParticipantsByWantLevel(t *testing.T) {
	players := []Player{*horst, *baerbel, *kay, *oke, *mable, *uwe, *etienne, *dieder, *ingebork}
	p := SetupTestKickerPlugin(players)

	if len(p.filterParticipantsByWantlevel(WLDecline)) != 1 {
		t.Errorf("filterParticipantsByWantlevel return unexpected results")
	}
	if len(p.filterParticipantsByWantlevel(WLParticipate)) != 4 {
		t.Errorf("filterParticipantsByWantlevel return unexpected results")
	}
	if len(p.filterParticipantsByWantlevel(WLVolunteer)) != 4 {
		t.Errorf("filterParticipantsByWantlevel return unexpected results")
	}
}

func TestRemoveParticipantByID(t *testing.T) {
	players := []Player{*horst, *baerbel, *kay, *oke, *mable, *uwe, *etienne, *dieder, *ingebork}
	p := SetupTestKickerPlugin(players)

	p.removeParticipantByID("1")
	if len(p.participants) != 8 {
		t.Errorf("removeParticipantByID return unexpected results")
	}

	p.removeParticipantByID("8")
	if len(p.participants) != 7 {
		t.Errorf("removeParticipantByID return unexpected results")
	}
}

// compares Player-slices, ignores order
func playerEqual(a, b []Player) bool {
	if len(a) != len(b) {
		return false
	}
	for _, va := range a {
		foundAinB := false
		for _, vb := range b {
			if va == vb {
				foundAinB = true
			}
		}

		if !foundAinB {
			return false
		}
	}
	return true
}
