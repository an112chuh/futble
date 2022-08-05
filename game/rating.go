package game

import (
	"database/sql"
	"futble/config"
	"futble/daemon"
	"futble/report"
	"futble/result"
	"math"
	"net/http"
	"time"
)

var LEN_SEARCH = 5

type Score struct {
	Home int `json:"home"`
	Away int `json:"away"`
}

func SearchRatingGameHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	if user.Rights == config.NotLogged {
		res = result.SetErrorResult(NOT_LOGGED_ERROR)
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
	query = `SELECT EXISTS(SELECT 1 FROM games.rating_pairs WHERE (user1 = $1 OR user2 = $1) AND active = TRUE)`
	err = db.QueryRow(query, params...).Scan(&exists)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(`Error in database`)
		return
	}
	if exists {
		res = result.SetErrorResult(`Can't search game because you are playing`)
		return
	}
	var s daemon.GameFinder
	query = `SELECT trophies FROM users.trophies WHERE id_user = $1`
	params = []any{user.ID}
	err = db.QueryRow(query, params...).Scan(&s.Rating)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(`Error in database`)
		return
	}
	query = `INSERT INTO users.invites (user1, searching, start_search, expiry) VALUES ($1, true, $2, $3)`
	params = []any{user.ID, time.Now(), time.Now().Add(time.Second * 60)}
	_, err = db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(`Error in database`)
		return
	}
	t := time.Now()
	s.ID = user.ID
	s.TimeStart = t
	daemon.SearchList.Mutex.Lock()
	daemon.SearchList.Items = append(daemon.SearchList.Items, s)
	daemon.SearchList.Mutex.Unlock()
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
			res.Items = map[string]any{"found": true}
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
	daemon.SearchList.Mutex.Lock()
	var position int
	for i := 0; i < len(daemon.SearchList.Items); i++ {
		if daemon.SearchList.Items[i].ID == IDUser {
			position = i
			break
		}
	}
	NewSlice := append(daemon.SearchList.Items[:position], daemon.SearchList.Items[position+1:]...)
	daemon.SearchList.Items = NewSlice
	daemon.SearchList.Mutex.Unlock()
	return err
}

func CheckGameFound(IDUser int) (IDGame int, err error) {
	daemon.ResultList.Mutex.Lock()
	for i := 0; i < len(daemon.ResultList.Items); i++ {
		found := false
		CurrentItem := daemon.ResultList.Items[i]
		if CurrentItem.Home == IDUser && !CurrentItem.IsDeletedHome {
			daemon.ResultList.Items[i].IsDeletedHome = true
			CurrentItem.IsDeletedHome = true
			found = true
		}
		if CurrentItem.Away == IDUser && !CurrentItem.IsDeletedAway {
			daemon.ResultList.Items[i].IsDeletedAway = true
			CurrentItem.IsDeletedAway = true
			found = true
		}
		if found && CurrentItem.IsDeletedHome && CurrentItem.IsDeletedAway {
			IDGame, err = AddNewRatingGame(CurrentItem.Home, CurrentItem.Away)
			daemon.ResultList.RemoveElements(i)
		} else {
			IDGame = 1
		}
	}
	daemon.ResultList.Mutex.Unlock()
	return
}

func AddNewRatingGame(User1 int, User2 int) (int, error) {
	db := config.ConnectDB()
	var IDGame int
	query := `INSERT INTO games.rating_pairs (user1, user2, created_at, active) VALUES ($1, $2, $3, TRUE) RETURNING id`
	params := []any{User1, User2, time.Now()}
	err := db.QueryRow(query, params...).Scan(&IDGame)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return -1, err
	}
	query = `INSERT INTO games.rating (id_game, is_home, id_user, score, tries) VALUES ($1, $2, $3, 0, 0)`
	params = []any{IDGame, true, User1}
	_, err = db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return -1, err
	}
	params = []any{IDGame, false, User2}
	_, err = db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return -1, err
	}
	query = `UPDATE users.invites SET searching = FALSE WHERE user1 = $1 OR user1 = $2`
	params = []any{User1, User2}
	_, err = db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return -1, err
	}
	return IDGame, err
}

func RatingGame(user config.User) (res result.ResultInfo) {
	IDGlobalGame := CheckRatingGameExist(user)
	if IDGlobalGame == -1 {
		res = result.SetErrorResult(`Error in searching current game`)
		return
	}
	if IDGlobalGame == 0 {
		res = result.SetErrorResult(`Please, create new game`)
		return
	}
	IDGame := CheckRatingGamePartExist(user)
	if IDGame == -1 {
		res = result.SetErrorResult(`Error in searching local game`)
		return
	}
	if IDGame == 0 {
		var err error
		IDGame, err = CreateGame(RATING, user)
		if err != nil {
			res = result.SetErrorResult(`Ошибка при создании новой игры`)
			return
		}
	}
	GameInfo, err := GameInfoCollect(IDGame)
	if err != nil {
		res = result.SetErrorResult(`Error in collecting game data`)
		return
	}
	GameInfo.GameMode = RATING
	res.Done = true
	res.Items = GameInfo
	return res
}

func RatingGameAnswer(user config.User, IDGuess int) (res result.ResultInfo) {
	IDGlobalGame := CheckRatingGameExist(user)
	if IDGlobalGame == -1 {
		res = result.SetErrorResult(`Error in searching current game`)
		return
	}
	if IDGlobalGame == 0 {
		res = result.SetErrorResult(`Please, create new game`)
		return
	}
	IDGame := CheckRatingGamePartExist(user)
	if IDGame == -1 {
		res = result.SetErrorResult(`Error in searching local game`)
		return
	}
	if IDGame == 0 {
		res = result.SetErrorResult(`This game doesn't exist`)
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
	if GameResult == LOSE {
		RatingAnswerFail(user.ID, IDGlobalGame)
		ChangeStatusRatingGame(IDGame)
		IDGame, err = CreateGame(RATING, user)
		if err != nil {
			res = result.SetErrorResult(`Error in creating new game`)
			return
		}
		GameInfo, err = GameInfoCollect(IDGame)
		if err != nil {
			res = result.SetErrorResult(`Error in collecting game data`)
			return
		}
	}
	if GameResult == WIN {
		Finished := RatingAnswerSuccess(user.ID, IDGlobalGame)
		ChangeStatusRatingGame(IDGame)
		if !Finished {
			IDGame, err = CreateGame(RATING, user)
			if err != nil {
				res = result.SetErrorResult(`Error in creating new game`)
				return
			}
			GameInfo, err = GameInfoCollect(IDGame)
			if err != nil {
				res = result.SetErrorResult(`Error in collecting game data`)
				return
			}
		}
	}
	res.Done = true
	res.Items = GameInfo
	return res
}

func RatingScoreHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	GameType, err := GetGameModeByID(user)
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(`Error in searching current game`)
		result.ReturnJSON(w, &res)
		return
	}
	if GameType != RATING {
		res = result.SetErrorResult(`Wrong game type(must be rating)`)
		result.ReturnJSON(w, &res)
		return
	}
	IDGlobalGame := CheckRatingGameExist(user)
	res = RatingScore(IDGlobalGame, user.ID)
	result.ReturnJSON(w, &res)
}

func RatingScore(IDGame int, IDUser int) (res result.ResultInfo) {
	db := config.ConnectDB()
	type UserScore struct {
		Score int
		ID    int
	}
	var UserGame []UserScore
	query := `SELECT score, id_user FROM games.rating WHERE id_game = $1`
	params := []any{IDGame}
	rows, err := db.Query(query, params...)
	if err != nil {
		res = result.SetErrorResult(`Error in getting score`)
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var u UserScore
		err = rows.Scan(&u.Score, &u.ID)
		if err != nil {
			res = result.SetErrorResult(`Error in getting score`)
			report.ErrorServer(nil, err)
			return
		}
		UserGame = append(UserGame, u)
	}
	var s Score
	if len(UserGame) < 2 {
		res = result.SetErrorResult(`Error in getting current game(probably doesn't exist)`)
		return
	}
	if UserGame[0].ID == IDUser {
		s.Home = UserGame[0].Score
		s.Away = UserGame[1].Score
	} else {
		s.Home = UserGame[1].Score
		s.Away = UserGame[0].Score
	}
	res.Items = s
	res.Done = true
	return
}

func CheckRatingGameExist(user config.User) int {
	db := config.ConnectDB()
	var ID int
	query := `SELECT id FROM games.rating_pairs WHERE (user1 = $1 OR user2 = $1) AND active IS TRUE`
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

func SearchRatingGameExtra(user config.User) int {
	db := config.ConnectDB()
	var ID int
	query := `SELECT id FROM games.rating_pairs WHERE (user1 = $1 OR user2 = $1) ORDER BY id DESC LIMIT 1`
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

func CheckRatingGamePartExist(user config.User) int {
	db := config.ConnectDB()
	var ID int
	query := `SELECT id FROM games.list WHERE id_user = $1 AND game_type = 2 AND active = TRUE`
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

func RatingAnswerSuccess(IDUser int, IDGame int) (Finished bool) {
	db := config.ConnectDB()
	Finished = false
	var Score, User1, User2 int
	query := `UPDATE games.rating SET score = score + 1, tries = tries + 1 WHERE id_user = $1 AND id_game = $2 RETURNING score`
	params := []any{IDUser, IDGame}
	err := db.QueryRow(query, params...).Scan(&Score)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	if Score >= 3 {
		query = `UPDATE games.rating_pairs SET active = FALSE WHERE id = $1 RETURNING user1, user2`
		params = []any{IDGame}
		err = db.QueryRow(query, params...).Scan(&User1, &User2)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
		}
		ChangeWin, ChangeLose := CountRateDiff(IDGame)
		var Change1, Change2 int
		query = `UPDATE games.rating SET ratediff = $1 WHERE id_game = $2 AND id_user = $3`
		if IDUser == User1 {
			params = []any{ChangeWin, IDGame, User1}
			Change1 = ChangeWin
		} else {
			params = []any{ChangeWin, IDGame, User2}
			Change2 = ChangeLose
		}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			return
		}
		if IDUser == User1 {
			params = []any{ChangeLose, IDGame, User2}
			Change1 = ChangeLose
		} else {
			params = []any{ChangeLose, IDGame, User1}
			Change2 = ChangeWin
		}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			return
		}
		query = `UPDATE users.trophies SET trophies = trophies + $1 WHERE id_user = $2`
		params = []any{Change1, User1}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			return
		}
		params = []any{Change2, User2}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			return
		}
		Finished = true
	}
	return
}

func RatingAnswerFail(IDUser int, IDGame int) {
	db := config.ConnectDB()
	var Score int
	query := `UPDATE games.rating SET score = score, tries = tries + 1 WHERE id_user = $1 AND id_game = $2 RETURNING score`
	params := []any{IDUser, IDGame}
	err := db.QueryRow(query, params...).Scan(&Score)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
}

func CountRateDiff(IDGame int) (ChangeWin int, ChangeLose int) {
	db := config.ConnectDB()
	type Data struct {
		Trophs int
		Score  int
	}
	var User []Data
	query := `SELECT trophies, score FROM users.trophies 
		INNER JOIN games.rating ON games.rating.id_user = users.trophies.id_user
		WHERE id_game = $1`
	params := []any{IDGame}
	rows, err := db.Query(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var t Data
		err = rows.Scan(&t.Trophs, &t.Score)
		if err != nil {
			report.ErrorServer(nil, err)
			return
		}
		User = append(User, t)
	}
	if User[1].Trophs < User[0].Trophs {
		tmp := User[0]
		User[0] = User[1]
		User[1] = tmp
	}
	Diff := User[1].Trophs - User[0].Trophs
	k := (math.Log(float64(Diff+200)) * 1.5) - 6.94947
	var Sum float64 = 60
	Sum /= (k + 1)
	var k2 float64
	if User[1].Score > User[0].Score {
		ChangeWin = int(Sum)
		if User[0].Score < 1000 {
			k2 = float64((User[0].Score)/10) / 100
		}
		if User[0].Score-int(Sum*k*k2) < 0 {
			ChangeLose = -User[0].Score
		} else {
			ChangeLose = -int(Sum * k * k2)
		}
	} else {
		ChangeWin = int(Sum * k)
		if User[1].Score < 1000 {
			k2 = float64((User[1].Score)/10) / 100
		}
		if User[1].Score-int(Sum) < 0 {
			ChangeLose = -User[1].Score
		} else {
			ChangeLose = -int(Sum)
		}
	}
	return
}

func ChangeStatusRatingGame(IDGame int) {
	db := config.ConnectDB()
	query := `UPDATE games.list SET active = false WHERE id = $1`
	params := []any{IDGame}
	_, err := db.Exec(query, params...)
	if err != nil {
		report.ErrorServer(nil, err)
	}
}
