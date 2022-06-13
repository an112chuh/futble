package game

import (
	"database/sql"
	"futble/config"
	"futble/report"
	"futble/result"
	"net/http"
)

func UnlimitedHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	IDGame := CheckUnlimitedGameExist(user)
	if IDGame == -1 {
		res = result.SetErrorResult(`Ошибка при поиске текущей игры`)
		result.ReturnJSON(w, &res)
		return
	}
	if IDGame == 0 {
		var err error
		IDGame, err = CreateGame(UNLIMITED, user)
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

func UnlimitedAnswerHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	IDGame := CheckUnlimitedGameExist(user)
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
	}
	if GameResult == NOTHING {
		res.Done = true
		res.Items = `Game continue`
	}
	result.ReturnJSON(w, &res)
}

func UnlimitedNewHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	IDGame := CheckUnlimitedGameExist(user)
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
	ChangeUnlimitedStatusGame(IDGame)
	res.Done = true
	result.ReturnJSON(w, &res)
}

func CheckUnlimitedGameExist(user config.User) int {
	db := config.ConnectDB()
	var ID int
	query := `SELECT id FROM games.list WHERE id_user = $1 AND game_type = 3 AND active = TRUE`
	params := []any{user.ID}
	err := db.QueryRow(query, params...).Scan(&ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0
		}
		report.ErrorServer(nil, err)
		return -1
	}
	return ID
}

func ChangeUnlimitedStatusGame(IDGame int) {
	db := config.ConnectDB()
	query := `UPDATE games.list SET active = false WHERE id = $1`
	params := []any{IDGame}
	_, err := db.Exec(query, params...)
	if err != nil {
		report.ErrorServer(nil, err)
	}
}

func GetIDBySurname(Surname string) (ID int, err error) {
	db := config.ConnectDB()
	query := `SELECT id FROM players.data WHERE surname = $1`
	params := []any{Surname}
	err = db.QueryRow(query, params...).Scan(&ID)
	return ID, err
}
