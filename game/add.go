package game

import (
	"futble/config"
	"futble/report"
	"math/rand"
	"time"
)

func AddDailyGames() {
	db := config.ConnectDB()
	var IDs []int
	query := `SELECT id FROM players.data`
	rows, err := db.Query(query)
	if err != nil {
		report.ErrorSQLServer(nil, err, query)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var ID int
		err = rows.Scan(&ID)
		if err != nil {
			report.ErrorServer(nil, err)
			return
		}
		IDs = append(IDs, ID)
	}
	rand.Shuffle(len(IDs), func(i, j int) {
		IDs[i], IDs[j] = IDs[j], IDs[i]
	})
	t := time.Now()
	for i := range IDs {
		query = `INSERT INTO games.daily_answers (id_answer, day_start, day_finish) VALUES ($1, $2, $3)`
		params := []any{IDs[i], t.Add(time.Hour * 24 * time.Duration(i)), t.Add(time.Hour * 24 * time.Duration(i+1))}
		_, err := db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			return
		}
	}
}
