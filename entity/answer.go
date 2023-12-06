package entity

type Guess struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Surname       string `json:"surname"`
	Age           int    `json:"age"`
	AgeColor      int    `json:"age_color"`
	Club          string `json:"club"`
	ClubColor     int    `json:"club_color"`
	League        string `json:"league"`
	LeagueColor   int    `json:"league_color"`
	Nation        string `json:"nation"`
	NationColor   int    `json:"nation_color"`
	Position      string `json:"position"`
	PositionColor int    `json:"position_color"`
	Price         int    `json:"price"`
	PriceColor    int    `json:"price_color"`
}

var GREY int = 0
var YELLOW int = 1
var GREEN int = 2
var RED int = 3
