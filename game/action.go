package game

import (
	"futble/config"
	"futble/report"
	"futble/result"
	"net/http"
	"strconv"
)

type Game struct {
	ID         int          `json:"id"`
	GameMode   int          `json:"game_mode"`
	Started    *bool        `json:"started,omitempty"`
	TimeStart  *string      `json:"start_time,omitempty"`
	GameResult *string      `json:"game_result,omitempty"`
	Answers    []AnswerType `json:"answers"`
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

func GameHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	GameMode, err := GetGameModeByID(user)
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(`Error in getting current game mode`)
		result.ReturnJSON(w, &res)
		return
	}
	switch GameMode {
	case DAILY:
		res = DailyGame(user)
	case UNLIMITED:
		res = UnlimitedGame(user)
	case RATING:
		if user.Rights == config.NotLogged {
			res = result.SetErrorResult(NOT_LOGGED_ERROR)
			result.ReturnJSON(w, &res)
			return
		}
		res = RatingGame(user)

	}
	result.ReturnJSON(w, &res)
}

func GameAnswerHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	GameMode, err := GetGameModeByID(user)
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(`Error in getting current game mode`)
		result.ReturnJSON(w, &res)
		return
	}
	keys := r.URL.Query()
	if len(keys[`id`]) < 1 {
		res = result.SetErrorResult(`Need 'id'`)
		result.ReturnJSON(w, &res)
		return
	}
	IDAnswer, err := strconv.Atoi(keys[`id`][0])
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(`Answer ID must be int`)
		result.ReturnJSON(w, &res)
		return
	}
	switch GameMode {
	case DAILY:
		res = DailyGameAnswer(user, IDAnswer)
	case RATING:
		if user.Rights == config.NotLogged {
			res = result.SetErrorResult(NOT_LOGGED_ERROR)
			result.ReturnJSON(w, &res)
			return
		}
		res = RatingGameAnswer(user, IDAnswer)
	case UNLIMITED:
		res = UnlimitedGameAnswer(user, IDAnswer)
	}
	result.ReturnJSON(w, &res)
}

func SwitchModeHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	keys := r.URL.Query()
	if len(keys[`id`]) < 1 {
		res = result.SetErrorResult(`Need 'id'`)
		result.ReturnJSON(w, &res)
		return
	}
	GameMode, err := strconv.Atoi(keys[`id`][0])
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(`GameMode must be int`)
		result.ReturnJSON(w, &res)
		return
	}
	if GameMode > 3 || GameMode < 1 {
		res = result.SetErrorResult(`GameMode must be from 1 to 3`)
		result.ReturnJSON(w, &res)
		return
	}
	db := config.ConnectDB()
	query := `UPDATE users.accounts SET game_mode = $1 WHERE id = $2`
	params := []any{GameMode, user.ID}
	_, err = db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(r, err, query, params...)
		res = result.SetErrorResult(`Error in database`)
		result.ReturnJSON(w, &res)
		return
	}
	res.Done = true
	result.ReturnJSON(w, &res)
}

func GetGameModeByID(user config.User) (int, error) {
	var GameMode int
	db := config.ConnectDB()
	query := `SELECT game_mode FROM users.accounts WHERE id = $1`
	params := []any{user.ID}
	err := db.QueryRow(query, params...).Scan(&GameMode)
	return GameMode, err
}

func CheckPlayerIDExist(ID int) (exists bool) {
	db := config.ConnectDB()
	query := `SELECT EXISTS(SELECT 1 FROM players.data WHERE id = $1)`
	params := []any{ID}
	err := db.QueryRow(query, params...).Scan(&exists)
	if err != nil {
		report.ErrorServer(nil, err)
	}
	return exists
}

func CheckCurrentGameFinished(IDGame int) (result int) {
	db := config.ConnectDB()
	var Finished bool
	query := `SELECT EXISTS(SELECT 1 FROM games.list WHERE id = $1 AND finish_time IS NOT NULL)`
	params := []any{IDGame}
	err := db.QueryRow(query, params...).Scan(&Finished)
	if err != nil {
		report.ErrorServer(nil, err)
		return -10
	}
	if !Finished {
		return NOTHING
	}
	var AnswerID, LastGuessID int
	query = `SELECT id_answer FROM games.list WHERE id = $1`
	err = db.QueryRow(query, params...).Scan(&AnswerID)
	if err != nil {
		report.ErrorServer(nil, err)
		return -10
	}
	query = `SELECT id_guess FROM games.guess WHERE id_game = $1 ORDER BY id DESC LIMIT 1`
	err = db.QueryRow(query, params...).Scan(&LastGuessID)
	if err != nil {
		report.ErrorServer(nil, err)
		return -10
	}
	if AnswerID == LastGuessID {
		return WIN
	}
	return LOSE
}
