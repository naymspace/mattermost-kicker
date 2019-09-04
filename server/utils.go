package main

func (p *KickerPlugin) filterParticipantsByWantlevel(wantLevel int) []Player {
	var players []Player

	for _, player := range p.participants {
		if player.wantLevel == wantLevel {
			players = append(players, player)
		}
	}

	return players
}
