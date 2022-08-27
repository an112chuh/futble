package daemon

import (
	"futble/config"
	"futble/report"
	"sync"
	"time"
)

type RatingGames struct {
	ID        int
	TimeStart *time.Time
}

type SafeRatingGames struct {
	Games []RatingGames
	Mutex sync.Mutex
}

type SafeFinishedGames struct {
	Games []int
	Mutex sync.Mutex
}

var GamesList SafeRatingGames
var FinishedList SafeFinishedGames

func RatingGameFinishing() {
	db := config.ConnectDB()
	query := `SELECT id, created_at FROM games.rating_pairs WHERE active IS TRUE`
	rows, err := db.Query(query)
	if err != nil {
		report.ErrorSQLServer(nil, err, query)
		return
	}
	GamesList.Mutex.Lock()
	for rows.Next() {
		var r RatingGames
		err = rows.Scan(&r.ID, &r.TimeStart)
		if err != nil {
			report.ErrorServer(nil, err)
		}
		GamesList.Games = append(GamesList.Games, r)
	}
	rows.Close()
	GamesList.Mutex.Unlock()
	for {
		GamesList.Mutex.Lock()
		for i := 0; i < len(GamesList.Games); i++ {
			//			fmt.Println(len(GamesList.Games))
			if GamesList.Games[i].TimeStart.Add(10 * time.Minute).Before(time.Now()) {
				FinishedList.Mutex.Lock()
				FinishedList.Games = append(FinishedList.Games, GamesList.Games[i].ID)
				FinishedList.Mutex.Unlock()
				GamesList.RemoveElements(i)
				i--
			}
		}
		GamesList.Mutex.Unlock()
	}
}

func (GamesList *SafeRatingGames) RemoveElements(i int) {
	NewSlice := append(GamesList.Games[:i], GamesList.Games[i+1:]...)
	GamesList.Games = NewSlice
}

func (FinishedList *SafeFinishedGames) RemoveElements(i int) {
	NewSlice := append(FinishedList.Games[:i], FinishedList.Games[i+1:]...)
	FinishedList.Games = NewSlice
}
