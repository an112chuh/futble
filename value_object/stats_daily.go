package valueobject

type DailyStats struct {
	Result        string  `json:"result"`
	NextGame      string  `json:"next_game"`
	LastAnswer    string  `json:"last_answer"`
	LastTime      string  `json:"last_time"`
	Total         int     `json:"total"`
	Success       int     `json:"success"`
	Percent       float64 `json:"percent"`
	Res1          int     `json:"res1"`
	Res2          int     `json:"res2"`
	Res3          int     `json:"res3"`
	Res4          int     `json:"res4"`
	Res5          int     `json:"res5"`
	Res6          int     `json:"res6"`
	Res7          int     `json:"res7"`
	Res8          int     `json:"res8"`
	CurrentStreak int     `json:"current_streak"`
	MaxStreak     int     `json:"max_streak"`
	BestTime      string  `json:"best_time"`
}
