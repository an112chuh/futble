package game

import (
	"database/sql"
	"futble/config"
	"futble/report"
	"futble/result"
	"net/http"
	"time"
)

var LEN_SEARCH = 5

func SearchRatingGameHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	if user.Rights == config.NotLogged {
		res = result.SetErrorResult(`Please, login to find opponent`)
		result.ReturnJSON(w, &res)
		return
	}
	res = SearchRatingGame(r, user)
	result.ReturnJSON(w, &res)
}

func SearchRatingGame(r *http.Request, user config.User) (res result.ResultInfo) {
	db := config.ConnectDB()
	ctx := r.Context()
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users.invites WHERE user1 = $1 AND searching = TRUE)`
	params := []any{user.ID}
	err := db.QueryRow(query, params...).Scan(&exists)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(`Error in database`)
		return
	}
	if exists {
		res = result.SetErrorResult(`Game is already searching`)
		return
	}
	query = `INSERT INTO users.invites (user1, searching, start_search, expiry) VALUES ($1, true, $2, $3)`
	params = []any{user.ID, time.Now(), time.Now().Add(time.Duration(LEN_SEARCH) * time.Second)}
	_, err = db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(`Error in database`)
		return
	}
	var IDGame int
	for {
		select {
		case <-ctx.Done():
			DiscardSearch(user.ID)
			return
		default:
		}
		IDGame, err = CheckGameFound(user.ID)
		if err != nil {
			res = result.SetErrorResult(`Error in database`)
			return
		}
		if IDGame != 0 {
			res.Done = true
			res.Items = map[string]any{"id_game": IDGame}
			break
		}
		time.Sleep(250 * time.Microsecond)
	}
	return
}

func DiscardSearch(IDUser int) error {
	db := config.ConnectDB()
	query := `UPDATE users.invites SET searching = false, finish_search = $1 WHERE user1 = $2 AND finish_search IS NULL`
	params := []any{time.Now(), IDUser}
	_, err := db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
	}
	return err
}

func CheckGameFound(IDUser int) (IDGame int, err error) {
	db := config.ConnectDB()
	query := `SELECT id FROM games.rating WHERE (user1 = $1 OR user2 = $1) AND active IS TRUE`
	params := []any{IDUser}
	err = db.QueryRow(query, params...).Scan(&IDGame)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return
}

func RatingGame(user config.User) (res result.ResultInfo) {
	return
}
