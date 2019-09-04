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
