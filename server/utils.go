package main

import (
	"strconv"
	"strings"

	"github.com/mattermost/mattermost-server/model"
)

func (p *KickerPlugin) filterParticipantsByWantlevel(wantLevel int) []Player {
	var players []Player

	for _, player := range p.participants {
		if player.wantLevel == wantLevel {
			players = append(players, player)
		}
	}

	return players
}

/*
ParseArgs parses the arguments entered by the user.

Takes the first 3 arguments:
- the command "kicker" itself
- a given hour
- a given minute

Returns the parsed hour and minute as integers on success.
*/
func ParseArgs(args string) ([]int, *model.AppError) {
	emptyParams := []int{}
	str := strings.SplitN(args, " ", 3)

	if len(str) == 3 {
		i1, err1 := strconv.Atoi(str[1])
		i2, err2 := strconv.Atoi(str[2])
		if err1 != nil || err2 != nil {
			return emptyParams, appError("Parsing failed", nil)
		}
		// Check for Limits
		if i1 >= paramMaxHour || i1 < 0 {
			return emptyParams, appError("Parsing failed", nil)
		}
		if i2 >= paramMaxMinute || i2 < 0 {
			return emptyParams, appError("Parsing failed", nil)
		}
		return []int{
			i1,
			i2,
		}, nil
	}

	if len(str) == 2 {
		i1, err1 := strconv.Atoi(str[1])
		if err1 != nil {
			return emptyParams, appError("Parsing failed", nil)
		}
		if i1 >= paramMaxHour || i1 < 0 {
			return emptyParams, appError("Parsing failed", nil)
		}
		return []int{
			i1,
		}, nil
	}

	if len(str) == 1 && str[0] != "" {
		// only the trigger was passed, no hour or minute; this is a valid command,
		// the default hour and minute will be used
		return emptyParams, nil
	}

	return emptyParams, appError("Parsing failed", nil)
}
