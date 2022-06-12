package game

import (
	"database/sql"
	"futble/config"
	"futble/report"
	"futble/result"
	"net/http"
	"time"
)

type DailyStats struct {
	Total   int     `json:"total"`
	Success int     `json:"success"`
	Percent float64 `json:"percent"`
	Res1    int     `json:"res1"`
	Res2    int     `json:"res2"`
	Res3    int     `json:"res3"`
	Res4    int     `json:"res4"`
	Res5    int     `json:"res5"`
	Res6    int     `json:"res6"`
	Res7    int     `json:"res7"`
	Res8    int     `json:"res8"`
}

type StatsStruct struct {
	Stats []Leaderboards `json:"stats"`
}

type Leaderboards struct {
	Login string `json:"login"`
	Res   int    `json:"res"`
	Tries int    `json:"tries"`
}

func GetDailyStatsHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	res = GetDailyStats(user.ID)
	result.ReturnJSON(w, &res)
}

func GetWeekStatsHandler(w http.ResponseWriter, r *http.Request) {
	res := GetWeekStats()
	result.ReturnJSON(w, &res)
}

func GetMonthStatsHandler(w http.ResponseWriter, r *http.Request) {
	res := GetMonthStats()
	result.ReturnJSON(w, &res)
}

func GetYearStatsHandler(w http.ResponseWriter, r *http.Request) {
	res := GetYearStats()
	result.ReturnJSON(w, &res)
}

func GetDailyStats(IDUser int) (res result.ResultInfo) {
	db := config.ConnectDB()
	var d DailyStats
	query := `SELECT total, success, percent, res1, res2, res3, res4, res5, res6, res7, res8 FROM users.daily WHERE id_user = $1`
	params := []any{IDUser}
	err := db.QueryRow(query, params...).Scan(&d.Total, &d.Success, &d.Percent, &d.Res1, &d.Res2, &d.Res3, &d.Res4, &d.Res5, &d.Res6, &d.Res7, &d.Res8)
	if err != nil && err != sql.ErrNoRows {
		res = result.SetErrorResult(`Ошибка в базе данных`)
		report.ErrorSQLServer(nil, err, query, params...)
		return res
	}
	res.Done = true
	res.Items = d
	return
}

func GetWeekStats() (res result.ResultInfo) {
	db := config.ConnectDB()
	var ID int
	var s StatsStruct
	t := time.Now()
	query := `SELECT id FROM dates.week WHERE start_time < $1 AND end_time > $2`
	params := []any{t, t}
	err := db.QueryRow(query, params...).Scan(&ID)
	if err != nil {
		res = result.SetErrorResult(`Ошибка в запросе`)
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	query = `SELECT leaderboards.weekly.res, leaderboards.weekly.tries, users.accounts.login FROM leaderboards.weekly  
		INNER JOIN users.accounts on users.accounts.id = leaderboards.weekly.id_player  
		WHERE leaderboards.weekly.id_week = $1 
		ORDER BY leaderboards.weekly.res DESC, leaderboards.weekly.tries DESC LIMIT 50`
	params = []any{ID}
	rows, err := db.Query(query, params...)
	if err != nil {
		res = result.SetErrorResult(`Ошибка в запросе`)
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var l Leaderboards
		err = rows.Scan(&l.Res, &l.Res, &l.Login)
		if err != nil {
			res = result.SetErrorResult(`Ошибка`)
			report.ErrorServer(nil, err)
			return
		}
		s.Stats = append(s.Stats, l)
	}
	res.Done = true
	res.Items = s
	return
}

func GetMonthStats() (res result.ResultInfo) {
	db := config.ConnectDB()
	var ID int
	var s StatsStruct
	t := time.Now()
	query := `SELECT id FROM dates.month WHERE start_time < $1 AND end_time > $2`
	params := []any{t, t}
	err := db.QueryRow(query, params...).Scan(&ID)
	if err != nil {
		res = result.SetErrorResult(`Ошибка в запросе`)
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	query = `SELECT leaderboards.monthly.res, leaderboards.monthly.tries, users.accounts.login FROM leaderboards.monthly  
		INNER JOIN users.accounts on users.accounts.id = leaderboards.monthly.id_player  
		WHERE leaderboards.monthly.id_month = $1 
		ORDER BY leaderboards.monthly.res DESC, leaderboards.monthly.tries DESC LIMIT 50`
	params = []any{ID}
	rows, err := db.Query(query, params...)
	if err != nil {
		res = result.SetErrorResult(`Ошибка в запросе`)
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var l Leaderboards
		err = rows.Scan(&l.Res, &l.Res, &l.Login)
		if err != nil {
			res = result.SetErrorResult(`Ошибка`)
			report.ErrorServer(nil, err)
			return
		}
		s.Stats = append(s.Stats, l)
	}
	res.Done = true
	res.Items = s
	return
}
func GetYearStats() (res result.ResultInfo) {
	db := config.ConnectDB()
	var ID int
	var s StatsStruct
	t := time.Now()
	query := `SELECT id FROM dates.year WHERE start_time < $1 AND end_time > $2`
	params := []any{t, t}
	err := db.QueryRow(query, params...).Scan(&ID)
	if err != nil {
		res = result.SetErrorResult(`Ошибка в запросе`)
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	query = `SELECT leaderboards.yearly.res, leaderboards.yearly.tries, users.accounts.login FROM leaderboards.yearly  
		INNER JOIN users.accounts on users.accounts.id = leaderboards.yearly.id_player  
		WHERE leaderboards.yearly.id_year = $1 
		ORDER BY leaderboards.yearly.res DESC, leaderboards.yearly.tries DESC LIMIT 50`
	params = []any{ID}
	rows, err := db.Query(query, params...)
	if err != nil {
		res = result.SetErrorResult(`Ошибка в запросе`)
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var l Leaderboards
		err = rows.Scan(&l.Res, &l.Res, &l.Login)
		if err != nil {
			res = result.SetErrorResult(`Ошибка`)
			report.ErrorServer(nil, err)
			return
		}
		s.Stats = append(s.Stats, l)
	}
	res.Done = true
	res.Items = s
	return
}
