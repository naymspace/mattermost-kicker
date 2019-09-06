package main

import (
	"testing"

	"github.com/mattermost/mattermost-server/model"
)

var horst = &Player{
	user: &model.User{
		Username: "horst",
	},
	wantLevel: wantLevelParticipant,
}

var baerbel = &Player{
	user: &model.User{
		Username: "bärbel",
	},
	wantLevel: wantLevelParticipant,
}

var kay = &Player{
	user: &model.User{
		Username: "kay",
	},
	wantLevel: wantLevelVolunteer,
}

func SetupTestKickerPlugin() *KickerPlugin {
	return &KickerPlugin{
		participants: []Player{*horst, *baerbel, *kay},
	}
}

func TestGetParticipants(t *testing.T) {
	p := SetupTestKickerPlugin()

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
	p := SetupTestKickerPlugin()

	players := p.GetVolunteers()

	if len(players) != 1 {
		t.Errorf("Number of volunteers was incorrect, got: %d, want: %d", len(players), 1)
	}

	if players[0].user.Username != "kay" {
		t.Errorf("First volunteer name was incorrect, got: %s, want: %s", players[0].user.Username, "kay")
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
			Args: "/kicker", // TODO: this is valid!
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
