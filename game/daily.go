package game

import (
	"database/sql"
	"futble/config"
	"futble/report"
	"futble/result"
	"net/http"
	"strconv"
	"time"
)

type Game struct {
	ID      int          `json:"id"`
	Answers []AnswerType `json:"answers"`
}

var WIN int = 1
var LOSE int = -1
var NOTHING int = 0

type AnswerType struct {
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

func DailyHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	IDGame := CheckTodayDailyGameExist(user)
	if IDGame == -1 {
		res = result.SetErrorResult(`Ошибка при поиске текущей игры`)
		result.ReturnJSON(w, &res)
		return
	}
	if IDGame == 0 {
		var err error
		IDGame, err = CreateGame(DAILY, user)
		if err != nil {
			res = result.SetErrorResult(`Ошибка при создании новой игры`)
			result.ReturnJSON(w, &res)
			return
		}
	}
	GameInfo, err := GameInfoCollect(IDGame)
	if err != nil {
		res = result.SetErrorResult(`Ошибка при получении данных об игре`)
		result.ReturnJSON(w, &res)
		return
	}
	res.Done = true
	res.Items = GameInfo
	result.ReturnJSON(w, &res)
}

func DailyTestHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	var g Game
	g.ID = 1
	g.Answers = make([]AnswerType, 8)
	for i := range g.Answers {
		g.Answers[i].Name = strconv.Itoa(i)
		g.Answers[i].Surname = strconv.Itoa(i)
		g.Answers[i].Age = 20 + i
		g.Answers[i].AgeColor = i % 3
		g.Answers[i].Club = `test` + strconv.Itoa(i)
		g.Answers[i].ClubColor = (i + 1) % 3
		g.Answers[i].League = `ENG`
		g.Answers[i].LeagueColor = (i + 2) % 3
		g.Answers[i].Nation = `POR`
		g.Answers[i].NationColor = i % 3
		g.Answers[i].Position = `RW`
		g.Answers[i].PositionColor = (i + 1) % 3
		g.Answers[i].Price = 1000000 * i
		g.Answers[i].Price = i / 3
	}
	res.Done = true
	res.Items = g
	result.ReturnJSON(w, &res)
}

func DailyAnswerHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	IDGame := CheckTodayDailyGameExist(user)
	if IDGame == -1 {
		res = result.SetErrorResult(`Ошибка при поиске текущей игры`)
		result.ReturnJSON(w, &res)
		return
	}
	if IDGame == 0 {
		res = result.SetErrorResult(`Данной игры не существует`)
		result.ReturnJSON(w, &res)
		return
	}
	keys := r.URL.Query()
	IDGuess, err := GetIDBySurname(keys[`name`][0])
	if err != nil {
		res = result.SetErrorResult(`Данного игрока не существует`)
		result.ReturnJSON(w, &res)
		return
	}
	res, GameResult, err := PutGuess(IDGuess, IDGame)
	if err != nil {
		res = result.SetErrorResult(`Ошибка при вставлении результата`)
		result.ReturnJSON(w, &res)
		return
	}
	if GameResult == -10 {
		result.ReturnJSON(w, &res)
		return
	}
	if GameResult == LOSE {
		res.Done = true
		res.Items = `Game lost`
		AddDailyStatLose(user.ID)
	}
	if GameResult == WIN {
		res.Done = true
		res.Items = `Game won`
		AddDailyStatWin(user.ID, IDGame)
	}
	if GameResult == NOTHING {
		res.Done = true
		res.Items = `Game continue`
	}
	result.ReturnJSON(w, &res)
}

func CheckTodayDailyGameExist(user config.User) int {
	db := config.ConnectDB()
	var ID int
	query := `SELECT id_game FROM games.daily WHERE id_user = $1 AND end_time > $2`
	params := []any{user.ID, time.Now()}
	err := db.QueryRow(query, params...).Scan(&ID)
	if err != nil {
		if err != sql.ErrNoRows {
			report.ErrorServer(nil, err)
			return -1
		} else {
			return 0
		}
	}
	return ID
}

func AddDailyStatWin(IDUser int, IDGame int) {
	db := config.ConnectDB()
	var NumOfGuesses int
	query := `SELECT count(*) FROM games.guess WHERE id_game = $1`
	params := []any{IDGame}
	err := db.QueryRow(query, params...).Scan(&NumOfGuesses)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	var exists bool
	query = `SELECT EXISTS(SELECT 1 FROM users.daily WHERE id_user = $1)`
	params = []any{IDUser}
	err = db.QueryRow(query, params...).Scan(&exists)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	if !exists {
		query = `INSERT INTO users.daily (id_user, total, success, percent, res1, res2, res3, res4, res5, res6, res7, res8) VALUES ($1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)`
		params = []any{IDUser}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			return
		}
	}
	query = `UPDATE users.daily SET total = total + 1, success = success + 1, percent = (success+1)/(total+1), res` + strconv.Itoa(NumOfGuesses) + ` = res` + strconv.Itoa(NumOfGuesses) + ` + 1 WHERE id_user = $1`
	params = []any{IDUser}
	_, err = db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
}

func AddDailyStatLose(IDUser int) {
	db := config.ConnectDB()
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users.daily WHERE id_user = $1)`
	params := []any{IDUser}
	err := db.QueryRow(query, params...).Scan(&exists)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	if !exists {
		query = `INSERT INTO users.daily (id_user, total, success, percent, res1, res2, res3, res4, res5, res6, res7, res8) VALUES ($1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)`
		params = []any{IDUser}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			return
		}
	}
	query = `UPDATE users.daily SET total = total + 1, percent = (success+1)/(total+1) WHERE id_user = $1`
	params = []any{IDUser}
	_, err = db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
}
