package game

import (
	"database/sql"
	"futble/config"
	"futble/report"
	"futble/result"
	"net/http"
	"time"
)

type AvgTime struct {
	LastAnswer string `json:"last_answer"`
	AvgTime    string `json:"avg_time"`
}

func UnlimitedGame(user config.User) (res result.ResultInfo) {
	IDGame := CheckUnlimitedGameExist(user)
	if IDGame == -1 {
		res = result.SetErrorResult(`Ошибка при поиске текущей игры`)
		return
	}
	if IDGame == 0 {
		var err error
		IDGame, err = CreateGame(UNLIMITED, user)
		if err != nil {
			res = result.SetErrorResult(`Ошибка при создании новой игры`)
			return
		}
	}
	GameInfo, err := GameInfoCollect(IDGame)
	if err != nil {
		res = result.SetErrorResult(`Ошибка при получении данных об игре`)
		return
	}
	Finished := CheckCurrentGameFinished(IDGame)
	GameInfo.GameResult = new(string)
	if Finished == LOSE {
		*GameInfo.GameResult = `GAME LOSE`
	}
	if Finished == WIN {
		*GameInfo.GameResult = `GAME WIN`
	}
	if Finished == NOTHING {
		*GameInfo.GameResult = `GAME CONTINUE`
	}
	GameInfo.GameMode = UNLIMITED
	res.Done = true
	res.Items = GameInfo
	return res
}

func UnlimitedGameAnswer(user config.User, IDGuess int) (res result.ResultInfo) {
	IDGame := CheckUnlimitedGameExist(user)
	if IDGame == -1 {
		res = result.SetErrorResult(`Ошибка при поиске текущей игры`)
		return
	}
	if IDGame == 0 {
		res = result.SetErrorResult(`Данной игры не существует`)
		return
	}
	exists := CheckPlayerIDExist(IDGuess)
	if !exists {
		res = result.SetErrorResult(`This player doesn't exist`)
		return
	}
	res, GameResult, err := PutGuess(IDGuess, IDGame)
	if err != nil {
		res = result.SetErrorResult(`Ошибка при вставлении результата`)
		return
	}
	if GameResult == -10 {
		return
	}
	GameInfo, err := GameInfoCollect(IDGame)
	if err != nil {
		res = result.SetErrorResult(`Ошибка при получении данных об игре`)
		return
	}
	GameInfo.GameResult = new(string)
	if GameResult == LOSE {
		*GameInfo.GameResult = `GAME LOSE`
		AddResultUnlimited(user.ID)
	}
	if GameResult == WIN {
		*GameInfo.GameResult = `GAME WIN`
		AddResultUnlimited(user.ID)
	}
	if GameResult == NOTHING {
		*GameInfo.GameResult = `GAME CONTINUE`
	}
	res.Done = true
	res.Items = GameInfo
	return res
}

func UnlimitedNewHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	GameType, err := GetGameModeByID(user)
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(`Ошибка при поиске текущей игры`)
		result.ReturnJSON(w, &res)
		return
	}
	if GameType != UNLIMITED {
		res = result.SetErrorResult(`Wrong game type(must be unlimited)`)
		result.ReturnJSON(w, &res)
		return
	}
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

func AddResultUnlimited(IDUser int) {
	db := config.ConnectDB()
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users.unlimited WHERE id_user = $1)`
	params := []any{IDUser}
	err := db.QueryRow(query, params...).Scan(&exists)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	if !exists {
		t := time.Now()
		query = `INSERT INTO users.unlimited (id_user, total_time, tries) VALUES ($1, $2, 0)`
		params = []any{IDUser, time.Time{}.Add(t.Sub(t))}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			return
		}
	}
	var StartTime, FinishTime time.Time
	query = `SELECT start_time, finish_time FROM games.list WHERE game_type = 3 AND id_user = $1 ORDER BY id DESC LIMIT 1`
	params = []any{IDUser}
	err = db.QueryRow(query, params...).Scan(&StartTime, &FinishTime)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	DiffTime := time.Time{}.Add(FinishTime.Sub(StartTime)).Format("15:04:05")
	query = `UPDATE users.unlimited SET tries = tries + 1, total_time = total_time + $1 WHERE id_user = $2`
	params = []any{DiffTime, IDUser}
	_, err = db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
}

func UnlimitedAvgHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	var a AvgTime
	var TotalTime time.Time
	var tries int
	db := config.ConnectDB()
	query := `SELECT total_time, tries FROM users.unlimited WHERE id_user = $1`
	params := []any{user.ID}
	err := db.QueryRow(query, params...).Scan(&TotalTime, &tries)
	if err != nil {
		report.ErrorSQLServer(r, err, query, params...)
		res = result.SetErrorResult(`Error in database`)
		result.ReturnJSON(w, &res)
		return
	}
	FirstTime, err := time.Parse("02.01.2006 15:04:05 -0700", "01.01.0000 00:00:00 +0000")
	if err != nil {
		report.ErrorServer(nil, err)
		return
	}
	a.AvgTime = time.Time{}.Add(TotalTime.Sub(FirstTime) / time.Duration(tries)).Format("15:04:05")
	t := time.Now()
	query = `SELECT name, surname FROM players.data 
		INNER JOIN games.list ON games.list.id_answer = players.data.id
		WHERE id_user = $1 AND game_type = 3 AND finish_time < $2
		ORDER BY games.list.id DESC LIMIT 1`
	params = []any{user.ID, t}
	var Name, Surname string
	err = db.QueryRow(query, params...).Scan(&Name, &Surname)
	if err != nil {
		report.ErrorSQLServer(r, err, query, params...)
		res = result.SetErrorResult(`Error in database`)
		result.ReturnJSON(w, &res)
		return
	}
	if Name == `` {
		a.LastAnswer = Surname
	} else {
		a.LastAnswer = Name + ` ` + Surname
	}
	res.Done = true
	res.Items = a
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
