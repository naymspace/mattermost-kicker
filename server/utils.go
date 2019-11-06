package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/model"
)

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
	defaultParams := []int{12}
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
		return defaultParams, nil
	}

	return emptyParams, appError("Parsing failed", nil)
}

// JoinPlayerNames concatenates the usernames of the players
func JoinPlayerNames(players []Player) string {
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
// from https://stackoverflow.com/questions/37334119/how-to-delete-an-element-from-a-slice-in-golang
func remove(s []Player, i int) []Player {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
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
