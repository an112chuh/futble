package aggregate

import "futble/entity"

type Game struct {
	Info    entity.GameBasic `json:"game_info"`
	Guesses []entity.Guess   `json:"guesses"`
}
