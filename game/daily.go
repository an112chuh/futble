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

func DailyGame(user config.User) (res result.ResultInfo) {
	IDGame := CheckTodayDailyGameExist(user)
	if IDGame == -1 {
		res = result.SetErrorResult(`Ошибка при поиске текущей игры`)
		return
	}
	if IDGame == 0 {
		var err error
		IDGame, err = CreateGame(DAILY, user)
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
	GameInfo.GameMode = DAILY
	res.Done = true
	res.Items = GameInfo
	return res
}

func DailyGameAnswer(user config.User, IDGuess int) (res result.ResultInfo) {
	IDGame := CheckTodayDailyGameExist(user)
	if IDGame == -1 {
		res = result.SetErrorResult(`Ошибка при поиске текущей игры`)
		return
	}
	if IDGame == 0 {
		res = result.SetErrorResult(`Данной игры не существует`)
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
		AddDailyStatLose(user.ID)
		UpdateSuccess(IDGame, false)
	}
	if GameResult == WIN {
		*GameInfo.GameResult = `GAME WIN`
		AddDailyStatWin(user.ID, IDGame)
		UpdateSuccess(IDGame, true)
	}
	if GameResult == NOTHING {
		*GameInfo.GameResult = `GAME CONTINUE`
	}
	res.Done = true
	res.Items = GameInfo
	return res
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
	var StartTime, FinishTime time.Time
	query = `select start_time, finish_time from games.list where id = $1`
	params = []any{IDGame}
	err = db.QueryRow(query, params...).Scan(&StartTime, &FinishTime)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	GameTime := time.Time{}.Add(FinishTime.Sub(StartTime))
	if !exists {
		query = `INSERT INTO users.daily (id_user, total, success, percent, res1, res2, res3, 
			res4, res5, res6, res7, res8, cur_streak, max_streak) 
			VALUES ($1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)`
		params = []any{IDUser}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			return
		}
	}
	var CurrentStreak, MaxStreak int
	var BestTime *time.Time
	query = `SELECT cur_streak, max_streak, best_time FROM users.daily WHERE id_user = $1`
	params = []any{IDUser}
	err = db.QueryRow(query, params...).Scan(&CurrentStreak, &MaxStreak, &BestTime)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	if MaxStreak == CurrentStreak {
		query = `UPDATE users.daily SET total = total + 1, success = success + 1, percent = (success+1)/(total+1),
		res` + strconv.Itoa(NumOfGuesses) + ` = res` + strconv.Itoa(NumOfGuesses) + ` + 1, 
		cur_streak = cur_streak + 1, max_streak = max_streak + 1 
		WHERE id_user = $1`
	} else {
		query = `UPDATE users.daily SET total = total + 1, success = success + 1, percent = (success+1)/(total+1),
		res` + strconv.Itoa(NumOfGuesses) + ` = res` + strconv.Itoa(NumOfGuesses) + ` + 1, 
		cur_streak = cur_streak + 1 
		WHERE id_user = $1`
	}
	params = []any{IDUser}
	_, err = db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	if BestTime == nil || GameTime.Before(*BestTime) {
		query = `UPDATE users.daily SET best_time = $1 WHERE id_user = $2`
		params = []any{GameTime, IDUser}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			return
		}
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
		query = `INSERT INTO users.daily (id_user, total, success, percent, res1, res2, res3, 
			res4, res5, res6, res7, res8, best_time) 
			VALUES ($1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, $2)`
		params = []any{IDUser, time.Second * 0}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			return
		}
	}
	query = `UPDATE users.daily SET total = total + 1, percent = (success+1)/(total+1), cur_streak = 0 WHERE id_user = $1`
	params = []any{IDUser}
	_, err = db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
}

func UpdateSuccess(IDGame int, IsWin bool) {
	db := config.ConnectDB()
	var query string
	if IsWin {
		query = `UPDATE games.daily SET success = TRUE, win = TRUE WHERE id_game = $1`
	} else {
		query = `UPDATE games.daily SET success = TRUE, win = FALSE WHERE id_game = $1`
	}
	params := []any{IDGame}
	_, err := db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
}

func GetDailyStatsHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	res = GetDailyStats(user.ID)
	result.ReturnJSON(w, &res)
}

func GetDailyStats(IDUser int) (res result.ResultInfo) {
	db := config.ConnectDB()
	var d DailyStats
	query := `SELECT total, success, percent, res1, res2, res3, res4, res5, res6, res7, res8, 
	cur_streak, max_streak, best_time FROM users.daily WHERE id_user = $1`
	params := []any{IDUser}
	var Time *time.Time
	err := db.QueryRow(query, params...).Scan(&d.Total, &d.Success, &d.Percent, &d.Res1,
		&d.Res2, &d.Res3, &d.Res4, &d.Res5, &d.Res6, &d.Res7, &d.Res8, &d.CurrentStreak, &d.MaxStreak, &Time)
	if err != nil && err != sql.ErrNoRows {
		res = result.SetErrorResult(`Ошибка в базе данных`)
		report.ErrorSQLServer(nil, err, query, params...)
		return res
	}
	if Time == nil {
		d.BestTime = ``
	} else {
		d.BestTime = Time.Format("15:04:05")
	}
	var exists bool
	t := time.Now()
	query = `SELECT EXISTS(SELECT 1 FROM games.daily WHERE id_user=$1 AND success = TRUE AND end_time > $2)`
	params = []any{IDUser, t}
	err = db.QueryRow(query, params...).Scan(&exists)
	if err != nil {
		res = result.SetErrorResult(`Ошибка в базе данных`)
		report.ErrorSQLServer(nil, err, query, params...)
		return res
	}
	if exists {
		var Name, Surname string
		var StartTime, FinishTime time.Time
		var Win bool
		query = `SELECT name, surname, start_time, finish_time, win FROM players.data 
			INNER JOIN games.list on games.list.id_answer = players.data.id
			INNER JOIN games.daily on games.daily.id_game = games.list.id
			WHERE games.list.id_user = $1 AND game_type = 1 ORDER BY games.list.id DESC LIMIT 1`
		params = []any{IDUser}
		err = db.QueryRow(query, params...).Scan(&Name, &Surname, &StartTime, &FinishTime, &Win)
		if err != nil {
			res = result.SetErrorResult(`Ошибка в базе данных`)
			report.ErrorSQLServer(nil, err, query, params...)
			return res
		}
		if Name == `` {
			d.LastAnswer = Surname
		} else {
			d.LastAnswer = Name + ` ` + Surname
		}
		d.LastTime = time.Time{}.Add(FinishTime.Sub(StartTime)).Format("15:04:05")
		if Win {
			d.Result = "GAME WIN"
		} else {
			d.Result = "GAME LOSE"
		}
	}
	var EndTime time.Time
	query = `SELECT end_time FROM games.daily WHERE id_user=$1 AND success = TRUE AND end_time > $2`
	params = []any{IDUser, t}
	err = db.QueryRow(query, params...).Scan(&EndTime)
	if err != nil {
		res = result.SetErrorResult(`Ошибка в базе данных`)
		report.ErrorSQLServer(nil, err, query, params...)
		return res
	}
	d.NextGame = time.Time{}.Add(EndTime.Sub(t)).Format("15:04:05")
	res.Done = true
	res.Items = d
	return
}
