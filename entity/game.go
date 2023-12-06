package entity

type GameBasic struct {
	ID         int     `json:"id"`
	GameMode   int     `json:"game_mode"`
	Started    *bool   `json:"started,omitempty"`
	TimeStart  *string `json:"start_time,omitempty"`
	TimeFinish *string `json:"finish_time,omitempty"`
	GameResult *string `json:"game_result,omitempty"`
}

var WIN int = 1
var LOSE int = -1
var NOTHING int = 0

var DAILY int = 1
var RATING int = 2
var UNLIMITED int = 3
