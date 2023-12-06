package entity

type Score struct {
	Home int `json:"home"`
	Away int `json:"away"`
}

type RatingUserStats struct {
	ID       int    `json:"id"`
	Login    string `json:"login"`
	Trophies int    `json:"trophies"`
}

type RatingStats struct {
	Home RatingUserStats `json:"home"`
	Away RatingUserStats `json:"away"`
}

type RatingRecord struct {
	Place    int    `json:"place"`
	ID       int    `json:"id"`
	Login    string `json:"login"`
	Trophies int    `json:"trophies"`
	Online   string `json:"online"`
}

type Rating struct {
	Ratings  []RatingRecord `json:"ratings"`
	MyRating *RatingRecord  `json:"my_rating,omitempty"`
}

type InviteStruct struct {
	ID         int    `json:"id"`
	Login      string `json:"login"`
	SendTime   string `json:"send_time"`
	ExpiryTime string `json:"expiry_time"`
}

type NotificationGlobal struct {
	Invites []InviteStruct `json:"invites"`
}

type UserStruct struct {
	ID    int    `json:"id"`
	Login string `json:"login"`
}

type FindUserStruct struct {
	Users []UserStruct `json:"users"`
}

type ResultGameStruct struct {
	Result     string `json:"result"`
	Score      string `json:"score"`
	Rating     int    `json:"rating"`
	RatingDiff int    `json:"rating_diff"`
	AddMoney   int    `json:"add_money"`
}

type RatingHintPricesGlobal struct {
	Red    RatingHintPricesStruct `json:"red"`
	Yellow RatingHintPricesStruct `json:"yellow"`
	Green  RatingHintPricesStruct `json:"green"`
}

type RatingHintPricesStruct struct {
	Age      int  `json:"age"`
	Club     *int `json:"club,omitempty"`
	League   int  `json:"league"`
	Nation   int  `json:"nation"`
	Position *int `json:"position,omitempty"`
	Price    int  `json:"price"`
}

type Hint struct {
	Color int
	Type  int
}

type HintOpponent struct {
	Exist bool `json:"exist"`
	Color *int `json:"color,omitempty"`
}
