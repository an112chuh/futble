package game

import (
	"database/sql"
	"futble/config"
	"futble/report"
	"futble/result"
	"net/http"
	"time"
)

func RatingAnswerHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	IDGame := CheckRatingGameExist(user)
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
	}
	if GameResult == WIN {
		res.Done = true
		res.Items = `Game won`
		AddRatingWin(IDGame, user.ID)
	}
	if GameResult == NOTHING {
		res.Done = true
		res.Items = `Game continue`
	}
	result.ReturnJSON(w, &res)
}

func CheckRatingGameExist(user config.User) int {
	db := config.ConnectDB()
	var ID int
	query := `SELECT id_game FROM games.rating WHERE id_user = $1 AND end_time > $2`
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

func AddRatingWin(IDGame int, IDUser int) {
	db := config.ConnectDB()
	var NumOfGuesses int
	query := `SELECT count(*) FROM games.guess WHERE id_game = $1`
	params := []any{IDGame}
	err := db.QueryRow(query, params...).Scan(&NumOfGuesses)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	var IDWeek, IDMonth, IDYear int
	t := time.Now()
	query = `SELECT id FROM dates.week WHERE start_time < $1 AND end_time > $2`
	params = []any{t, t}
	err = db.QueryRow(query, params...).Scan(&IDWeek)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	query = `SELECT id FROM dates.month WHERE start_time < $1 AND end_time > $2`
	params = []any{t, t}
	err = db.QueryRow(query, params...).Scan(&IDMonth)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	query = `SELECT id FROM dates.year WHERE start_time < $1 AND end_time > $2`
	params = []any{t, t}
	err = db.QueryRow(query, params...).Scan(&IDYear)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	var exists bool
	query = `SELECT EXISTS(SELECT 1 FROM leaderboards.weekly WHERE id_week = $1 AND id_player = $2)`
	params = []any{IDWeek, IDUser}
	err = db.QueryRow(query, params...).Scan(&exists)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	if exists {
		query = `UPDATE leaderboards.weekly SET res = res + 1, tries = tries + $1 WHERE id_player = $2 AND id_week = $3`
		params = []any{NumOfGuesses, IDUser, IDWeek}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			return
		}
	} else {
		query = `INSERT INTO leaderboards.weekly (res, tries, id_player, id_week) VALUES (1, $1, $2, $3)`
		params = []any{NumOfGuesses, IDUser, IDWeek}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			return
		}
	}
	query = `SELECT EXISTS(SELECT 1 FROM leaderboards.monthly WHERE id_month = $1 AND id_player = $2)`
	params = []any{IDMonth, IDUser}
	err = db.QueryRow(query, params...).Scan(&exists)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	if exists {
		query = `UPDATE leaderboards.monthly SET res = res + 1, tries = tries + $1 WHERE id_player = $2 AND id_month = $3`
		params = []any{NumOfGuesses, IDUser, IDMonth}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			return
		}
	} else {
		query = `INSERT INTO leaderboards.monthly (res, tries, id_player, id_month) VALUES (1, $1, $2, $3)`
		params = []any{NumOfGuesses, IDUser, IDMonth}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			return
		}
	}
	query = `SELECT EXISTS(SELECT 1 FROM leaderboards.yearly WHERE id_year = $1 AND id_player = $2)`
	params = []any{IDYear, IDUser}
	err = db.QueryRow(query, params...).Scan(&exists)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	if exists {
		query = `UPDATE leaderboards.yearly SET res = res + 1, tries = tries + $1 WHERE id_player = $2 AND id_year = $3`
		params = []any{NumOfGuesses, IDUser, IDYear}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			return
		}
	} else {
		query = `INSERT INTO leaderboards.yearly (res, tries, id_player, id_year) VALUES (1, $1, $2, $3)`
		params = []any{NumOfGuesses, IDUser, IDYear}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			return
		}
	}
}
