package entity

import "time"

type Player struct {
	ID          int
	Name        string
	Surname     string
	Club        string
	ClubShort   string
	League      string
	Nation      string
	NationShort string
	Position    string
	Price       int
	Birth       time.Time
}

type PlayerBasic struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type SearchPlayers struct {
	Players []PlayerBasic `json:"players"`
}
