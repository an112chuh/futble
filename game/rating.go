package game

import (
	"database/sql"
	"errors"
	"futble/config"
	"futble/daemon"
	"futble/report"
	"futble/result"
	"math"
	"net/http"
	"strconv"
	"time"
)

var LEN_SEARCH = 5

type Score struct {
	Home int `json:"home"`
	Away int `json:"away"`
}

type RatingUserStats struct {
	ID       int    `json:"id"`
	Login    string `json:"login"`
	Trophies int    `json:"trophies"`
}

type RatingStats struct {
	Home RatingUserStats `json:"home"`
	Away RatingUserStats `json:"away"`
}

type RatingRecord struct {
	Place    int    `json:"place"`
	ID       int    `json:"id"`
	Login    string `json:"login"`
	Trophies int    `json:"trophies"`
	Online   string `json:"online"`
}

type Rating struct {
	Ratings  []RatingRecord `json:"ratings"`
	MyRating *RatingRecord  `json:"my_rating,omitempty"`
}

type InviteStruct struct {
	ID         int    `json:"id"`
	Login      string `json:"login"`
	SendTime   string `json:"send_time"`
	ExpiryTime string `json:"expiry_time"`
}

type NotificationGlobal struct {
	Invites []InviteStruct `json:"invites"`
}

type UserStruct struct {
	ID    int    `json:"id"`
	Login string `json:"login"`
}

type FindUserStruct struct {
	Users []UserStruct `json:"users"`
}

type ResultGameStruct struct {
	Result     string `json:"result"`
	Score      string `json:"score"`
	Rating     int    `json:"rating"`
	RatingDiff int    `json:"rating_diff"`
	AddMoney   int    `json:"add_money"`
}

type RatingHintPricesGlobal struct {
	Red    RatingHintPricesStruct `json:"red"`
	Yellow RatingHintPricesStruct `json:"yellow"`
	Green  RatingHintPricesStruct `json:"green"`
}

type RatingHintPricesStruct struct {
	Age      int  `json:"age"`
	Club     *int `json:"club,omitempty"`
	League   int  `json:"league"`
	Nation   int  `json:"nation"`
	Position *int `json:"position,omitempty"`
	Price    int  `json:"price"`
}

type Hint struct {
	Color int
	Type  int
}

type HintOpponent struct {
	Exist bool `json:"exist"`
	Color *int `json:"color,omitempty"`
}

func SearchRatingGameHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	if user.Rights == config.NotLogged {
		res = result.SetErrorResult(NOT_LOGGED_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	if daemon.IsMaintenaunce {
		res = result.SetErrorResult(MAINTENAUNCE_ERROR + daemon.MaintenaunceReason)
		result.ReturnJSON(w, &res)
		return
	}
	res = SearchRatingGame(r, user)
	result.ReturnJSON(w, &res)
}

func RatingSendInviteHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	if user.Rights == config.NotLogged {
		res = result.SetErrorResult(NOT_LOGGED_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	if daemon.IsMaintenaunce {
		res = result.SetErrorResult(MAINTENAUNCE_ERROR + daemon.MaintenaunceReason)
		result.ReturnJSON(w, &res)
		return
	}
	keys := r.URL.Query()
	if len(keys[`id`]) != 1 {
		res = result.SetErrorResult(`Need id parameter`)
		result.ReturnJSON(w, &res)
		return
	}
	IDInvited, err := strconv.Atoi(keys[`id`][0])
	if err != nil {
		report.ErrorServer(r, nil)
		res = result.SetErrorResult(`Error in parsing id parameter`)
		return
	}
	res = SendInviteGame(r, user, IDInvited)
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
			query := `UPDATE users.invites SET searching = false, finish_search = $1 WHERE user1 = $2 AND finish_search IS NULL`
			params := []any{time.Now(), user.ID}
			_, err := db.Exec(query, params...)
			if err != nil {
				report.ErrorSQLServer(nil, err, query, params...)
			}
			res.Done = true
			res.Items = map[string]any{"found": true}
			break
		}
		time.Sleep(250 * time.Microsecond)
	}
	return
}

func SendInviteGame(r *http.Request, user config.User, IDInvited int) (res result.ResultInfo) {
	db := config.ConnectDB()
	ctx := r.Context()
	t := time.Now()
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
	query = `INSERT INTO users.invites (user1, user2, searching, start_search, is_invite, expiry, accepted) VALUES ($1, $2, TRUE, $3, TRUE, $4, FALSE)`
	params = []any{user.ID, IDInvited, t, t.Add(2 * time.Minute)}
	_, err = db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(r, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	var inv daemon.InviteGames
	query = `SELECT login FROM users.accounts WHERE id = $1`
	params = []any{user.ID}
	err = db.QueryRow(query, params...).Scan(&inv.UserLogin)
	if err != nil {
		report.ErrorSQLServer(r, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	daemon.Invites.Mutex.Lock()
	inv.Found = false
	inv.Rejected = false
	inv.User1 = user.ID
	inv.StartTime = t
	inv.ExpiryTime = t.Add(20 * time.Second)
	daemon.Invites.Items[IDInvited] = append(daemon.Invites.Items[IDInvited], inv)
	daemon.Invites.Mutex.Unlock()
	for {
		select {
		case <-ctx.Done():
			go daemon.DiscardSearch(user.ID)
			daemon.Invites.Mutex.Lock()
			delete(daemon.Invites.Items, IDInvited)
			daemon.Invites.Mutex.Unlock()
			return
		default:
		}
		daemon.Invites.Mutex.Lock()
		value, ok := daemon.Invites.Items[IDInvited]
		if !ok {
			res = result.SetErrorResult(`Element not found`)
			daemon.Invites.Mutex.Unlock()
			return
		}
		var exists bool = false
		var item daemon.InviteGames
		var myitem int
		for i := 0; i < len(value); i++ {
			if value[i].User1 == user.ID {
				exists = true
				item = value[i]
				myitem = i
				break
			}
		}
		if !exists {
			res = result.SetErrorResult(`Timeout reached`)
			daemon.Invites.Mutex.Unlock()
			return
		}
		if item.Found {
			_, err = AddNewRatingGame(user.ID, IDInvited)
			if err != nil {
				res = result.SetErrorResult(DATABASE_ERROR)
				daemon.Invites.Mutex.Unlock()
				return
			}
			value = daemon.RemoveElements(value, myitem)
			daemon.Invites.Items[IDInvited] = value
			res.Done = true
			res.Items = map[string]any{"found": true}
			daemon.Invites.Mutex.Unlock()
			return
		}
		if item.Rejected {
			go daemon.DiscardSearch(user.ID)
			daemon.Invites.Mutex.Lock()
			value = daemon.RemoveElements(value, myitem)
			daemon.Invites.Items[IDInvited] = value
			daemon.Invites.Mutex.Unlock()
			res = result.SetErrorResult(`Invite declined`)
			return
		}
		daemon.Invites.Mutex.Unlock()
		time.Sleep(250 * time.Microsecond)
	}
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
	t := time.Now()
	query := `INSERT INTO games.rating_pairs (user1, user2, created_at, active) VALUES ($1, $2, $3, TRUE) RETURNING id`
	params := []any{User1, User2, t}
	err := db.QueryRow(query, params...).Scan(&IDGame)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return -1, err
	}
	daemon.GamesList.Mutex.Lock()
	var r daemon.RatingGames
	r.ID = IDGame
	r.TimeStart = &t
	daemon.GamesList.Games = append(daemon.GamesList.Games, r)
	daemon.GamesList.Mutex.Unlock()
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
	Hints, err := GetCurrentGameHints(IDGame)
	if err != nil {
		res = result.SetErrorResult(`Error in getting hints`)
		return
	}
	var GameInfo Game
	if len(Hints) > 0 {
		GameInfo, err = GameInfoCollect(IDGame, Hints)
	} else {
		GameInfo, err = GameInfoCollect(IDGame)
	}
	if err != nil {
		res = result.SetErrorResult(`Error in collecting game data`)
		return
	}
	GameInfo.GameMode = RATING
	GameInfo.TimeFinish = new(string)
	*GameInfo.TimeFinish = RatingGameTimeFinish(IDGlobalGame)
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
	Hints, err := GetCurrentGameHints(IDGame)
	if err != nil {
		res = result.SetErrorResult(`Error in getting hints`)
		return
	}
	var GameInfo Game
	if len(Hints) > 0 {
		GameInfo, err = GameInfoCollect(IDGame, Hints)
	} else {
		GameInfo, err = GameInfoCollect(IDGame)
	}
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
		Hints, err := GetCurrentGameHints(IDGame)
		if err != nil {
			res = result.SetErrorResult(`Error in getting hints`)
			return
		}
		if len(Hints) > 0 {
			GameInfo, err = GameInfoCollect(IDGame, Hints)
		} else {
			GameInfo, err = GameInfoCollect(IDGame)
		}
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
			Hints, err := GetCurrentGameHints(IDGame)
			if err != nil {
				res = result.SetErrorResult(`Error in getting hints`)
				return
			}
			if len(Hints) > 0 {
				GameInfo, err = GameInfoCollect(IDGame, Hints)
			} else {
				GameInfo, err = GameInfoCollect(IDGame)
			}
			if err != nil {
				res = result.SetErrorResult(`Error in collecting game data`)
				return
			}
		}
	}
	GameInfo.TimeFinish = new(string)
	*GameInfo.TimeFinish = RatingGameTimeFinish(IDGlobalGame)
	res.Done = true
	res.Items = GameInfo
	return res
}

func RatingScoreHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	if user.Rights == config.NotLogged {
		res = result.SetErrorResult(NOT_LOGGED_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	GameType, err := GetGameModeByID(user)
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(GAME_SEARCH_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	if GameType != RATING {
		res = result.SetErrorResult(WRONG_GAME_TYPE)
		result.ReturnJSON(w, &res)
		return
	}
	IDGlobalGame := CheckRatingGameExist(user)
	res = RatingScore(IDGlobalGame, user.ID)
	result.ReturnJSON(w, &res)
}

func RatingGameStatsHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	if user.Rights == config.NotLogged {
		res = result.SetErrorResult(NOT_LOGGED_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	GameType, err := GetGameModeByID(user)
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(GAME_SEARCH_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	if GameType != RATING {
		res = result.SetErrorResult(WRONG_GAME_TYPE)
		result.ReturnJSON(w, &res)
		return
	}
	IDGlobalGame := CheckRatingGameExist(user)
	res = GetGameData(IDGlobalGame, user.ID)
	result.ReturnJSON(w, &res)
}

func RatingCoinsHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	if user.Rights == config.NotLogged {
		res = result.SetErrorResult(NOT_LOGGED_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	Coins := GetUserCoins(user.ID)
	res.Done = true
	res.Items = map[string]interface{}{"coins": Coins}
	result.ReturnJSON(w, &res)
}

func RatingStandingsHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	keys := r.URL.Query()
	if len(keys[`page`]) != 1 {
		res = result.SetErrorResult(`Need field 'page'`)
		result.ReturnJSON(w, &res)
		return
	}
	IDPage, err := strconv.Atoi(keys[`page`][0])
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(`Wrong field 'page'`)
		return
	}
	var IsLogged bool = true
	if user.Rights == config.NotLogged {
		IsLogged = false
	}
	res = GetRatingStandings(user.ID, IsLogged, IDPage)
	result.ReturnJSON(w, &res)
}

func RatingNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	if user.Rights == config.NotLogged {
		res = result.SetErrorResult(NOT_LOGGED_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	var Notifications NotificationGlobal
	var Invite InviteStruct
	daemon.Invites.Mutex.Lock()
	value, ok := daemon.Invites.Items[user.ID]
	if !ok {
		res.Done = true
		res.Items = Notifications
		result.ReturnJSON(w, &res)
		return
	}
	for i := 0; i < len(value); i++ {
		Invite.ID = value[i].User1
		Invite.Login = value[i].UserLogin
		Invite.SendTime = value[i].StartTime.Format("15:04:05")
		Invite.ExpiryTime = value[i].ExpiryTime.Format("15:04:05")
		Notifications.Invites = append(Notifications.Invites, Invite)
	}
	daemon.Invites.Mutex.Unlock()
	res.Done = true
	res.Items = Notifications
	result.ReturnJSON(w, &res)
}

func RatingConfirmInviteHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	if user.Rights == config.NotLogged {
		res = result.SetErrorResult(NOT_LOGGED_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	keys := r.URL.Query()
	if len(keys[`id`]) != 1 {
		res = result.SetErrorResult(`Need field 'id'`)
		result.ReturnJSON(w, &res)
		return
	}
	IDInviter, err := strconv.Atoi(keys[`id`][0])
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(`Wrong field 'id'`)
		return
	}
	daemon.Invites.Mutex.Lock()
	value, ok := daemon.Invites.Items[user.ID]
	if !ok {
		res = result.SetErrorResult(`Don't have any invites`)
		daemon.Invites.Mutex.Unlock()
		result.ReturnJSON(w, &res)
		return
	}
	for i := 0; i < len(value); i++ {
		if value[i].User1 == IDInviter {
			value[i].Found = true
			daemon.Invites.Items[user.ID] = value
			daemon.Invites.Mutex.Unlock()
			res.Done = true
			result.ReturnJSON(w, &res)
			return
		}
	}
	res = result.SetErrorResult(`Current invite doesn't exist`)
	daemon.Invites.Mutex.Unlock()
	result.ReturnJSON(w, &res)
}

func RatingRejectInviteHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	if user.Rights == config.NotLogged {
		res = result.SetErrorResult(NOT_LOGGED_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	keys := r.URL.Query()
	if len(keys[`id`]) != 1 {
		res = result.SetErrorResult(`Need field 'id'`)
		result.ReturnJSON(w, &res)
		return
	}
	IDInviter, err := strconv.Atoi(keys[`id`][0])
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(`Wrong field 'id'`)
		return
	}
	daemon.Invites.Mutex.Lock()
	value, ok := daemon.Invites.Items[user.ID]
	if !ok {
		res = result.SetErrorResult(`Don't have any invites`)
		daemon.Invites.Mutex.Unlock()
		result.ReturnJSON(w, &res)
		return
	}
	for i := 0; i < len(value); i++ {
		if value[i].User1 == IDInviter {
			value[i].Rejected = true
			daemon.Invites.Items[user.ID] = value
			daemon.Invites.Mutex.Unlock()
			res.Done = true
			result.ReturnJSON(w, &res)
			return
		}
	}
	res = result.SetErrorResult(`Current invite doesn't exist`)
	daemon.Invites.Mutex.Unlock()
	result.ReturnJSON(w, &res)
}

func RatingUserSearchHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	if user.Rights == config.NotLogged {
		res = result.SetErrorResult(NOT_LOGGED_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	keys := r.URL.Query()
	if len(keys[`login`]) != 1 {
		res = result.SetErrorResult(`Need field 'login'`)
		result.ReturnJSON(w, &res)
		return
	}
	res = FindUserByName(keys[`login`][0])
	result.ReturnJSON(w, &res)
}

func RatingResultHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	if user.Rights == config.NotLogged {
		res = result.SetErrorResult(NOT_LOGGED_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	res = GetLastRatingGame(user.ID)
	result.ReturnJSON(w, &res)
}

func RatingHintPricesHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	if user.Rights == config.NotLogged {
		res = result.SetErrorResult(NOT_LOGGED_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	res = GetHintsPrices()
	result.ReturnJSON(w, &res)
}

func RatingHintHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	if user.Rights == config.NotLogged {
		res = result.SetErrorResult(NOT_LOGGED_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	keys := r.URL.Query()
	if len(keys[`color`]) != 1 {
		res = result.SetErrorResult(`Need field 'color'`)
		result.ReturnJSON(w, &res)
		return
	}
	IDColor, err := strconv.Atoi(keys[`color`][0])
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(`Wrong field 'color'`)
		return
	}
	if len(keys[`type`]) != 1 {
		res = result.SetErrorResult(`Need field 'type'`)
		result.ReturnJSON(w, &res)
		return
	}
	IDType, err := strconv.Atoi(keys[`type`][0])
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(`Wrong field 'type'`)
		return
	}
	res = PutHint(user, IDColor, IDType)
	result.ReturnJSON(w, &res)
}

func RatingHintOpponentHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	if user.Rights == config.NotLogged {
		res = result.SetErrorResult(NOT_LOGGED_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	res = GetHintOpponent(user)
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

func GetGameData(IDGlobalGame int, IDUser int) (res result.ResultInfo) {
	db := config.ConnectDB()
	query := `SELECT games.rating.id_user, users.accounts.login, trophies FROM users.trophies
		INNER JOIN games.rating ON games.rating.id_user = users.trophies.id_user
		INNER JOIN users.accounts ON users.accounts.id = users.trophies.id_user
		WHERE games.rating.id_game = $1`
	params := []any{IDGlobalGame}
	var Stats []RatingUserStats
	rows, err := db.Query(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var p RatingUserStats
		err = rows.Scan(&p.ID, &p.Login, &p.Trophies)
		if err != nil {
			report.ErrorServer(nil, err)
			res = result.SetErrorResult(DATABASE_ERROR)
			return
		}
		Stats = append(Stats, p)
	}
	if len(Stats) < 2 {
		res = result.SetErrorResult(`Error in getting current game(probably doesn't exist)`)
		return
	}
	var FinalStats RatingStats
	if Stats[0].ID == IDUser {
		FinalStats.Home = Stats[0]
		FinalStats.Away = Stats[1]
	} else {
		FinalStats.Home = Stats[1]
		FinalStats.Away = Stats[0]
	}
	res.Done = true
	res.Items = FinalStats
	return
}

func GetUserCoins(IDUser int) int {
	var Coins int
	db := config.ConnectDB()
	query := `SELECT coins FROM users.trophies WHERE id_user = $1`
	params := []any{IDUser}
	err := db.QueryRow(query, params...).Scan(&Coins)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return -1
	}
	return Coins
}

func GetRatingStandings(IDUser int, IsLogged bool, IDPage int) (res result.ResultInfo) {
	var p result.Paginator
	var Rating Rating
	var Count int
	db := config.ConnectDB()
	query := `SELECT COUNT(*) FROM users.trophies`
	err := db.QueryRow(query).Scan(&Count)
	if err != nil {
		report.ErrorSQLServer(nil, err, query)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	p.CountPage = Count / 15
	p.Limit = 15
	p.Page = IDPage
	p.Total = Count
	p.Offset = (IDPage - 1) * 15
	query = `SELECT * from (SELECT trophies, users.accounts.id, login, online, RANK() OVER 
		(ORDER BY trophies DESC, updated_at ASC) AS number FROM users.trophies
		INNER JOIN users.accounts ON users.accounts.id = users.trophies.id_user) as abc limit 15 offset $1`
	params := []any{p.Offset}
	rows, err := db.Query(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var r RatingRecord
		err = rows.Scan(&r.Trophies, &r.ID, &r.Login, &r.Online, &r.Place)
		if err != nil {
			report.ErrorServer(nil, err)
			res = result.SetErrorResult(DATABASE_ERROR)
			return
		}
		Rating.Ratings = append(Rating.Ratings, r)
	}
	if IsLogged {
		var r RatingRecord
		Rating.MyRating = new(RatingRecord)
		query = `SELECT * FROM (SELECT trophies, users.accounts.id, login, online, RANK()
			OVER (ORDER BY trophies DESC, updated_at ASC) AS number FROM users.trophies
			INNER JOIN users.accounts ON users.accounts.id = users.trophies.id_user) AS abc where abc.id = $1`
		params = []any{IDUser}
		err = db.QueryRow(query, params...).Scan(&r.Trophies, &r.ID, &r.Login, &r.Online, &r.Place)
		if err != nil {
			report.ErrorServer(nil, err)
			res = result.SetErrorResult(DATABASE_ERROR)
			return
		}
		*Rating.MyRating = r
	}
	res.Paginator = new(result.Paginator)
	res.Done = true
	res.Items = Rating
	*res.Paginator = p
	return
}

func GetCurrentGameHints(IDGame int) (hints []Hint, err error) {
	db := config.ConnectDB()
	query := `SELECT color, type FROM hints.prices
		INNER JOIN hints.game ON hints.game.id_hint = hints.prices.id
		WHERE hints.game.id_local_game = $1`
	params := []any{IDGame}
	rows, err := db.Query(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var h Hint
		err = rows.Scan(&h.Color, &h.Type)
		if err != nil {
			report.ErrorServer(nil, err)
			return
		}
		hints = append(hints, h)
	}
	return
}

func FindUserByName(Login string) (res result.ResultInfo) {
	db := config.ConnectDB()
	query := `SELECT id, login FROM users.accounts WHERE login like '` + Login + `%' AND not_logged = FALSE`
	rows, err := db.Query(query)
	if err != nil {
		report.ErrorSQLServer(nil, err, query)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	var Finds FindUserStruct
	defer rows.Close()
	for rows.Next() {
		var u UserStruct
		err = rows.Scan(&u.ID, &u.Login)
		if err != nil {
			report.ErrorServer(nil, err)
		}
		Finds.Users = append(Finds.Users, u)
	}
	res.Done = true
	res.Items = Finds
	return
}

func GetLastRatingGame(IDUser int) (res result.ResultInfo) {
	var ResultGame ResultGameStruct
	var IDGame int
	var exists bool
	db := config.ConnectDB()
	type Result struct {
		IDUser    int
		Score     int
		Rating    int
		RateDiff  int
		CoinsDiff int
	}
	query := `SELECT EXISTS(SELECT 1 FROM games.rating_pairs WHERE (user1 = $1 OR user2 = $1) AND active IS TRUE)`
	params := []any{IDUser}
	err := db.QueryRow(query, params...).Scan(&exists)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	if exists {
		res = result.SetErrorResult(`You have an active game`)
		return
	}
	query = `SELECT id FROM games.rating_pairs WHERE (user1 = $1 OR user2 = $1) AND active IS FALSE ORDER BY id DESC LIMIT 1`
	params = []any{IDUser}
	err = db.QueryRow(query, params...).Scan(&IDGame)
	if err != nil && err != sql.ErrNoRows {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	if err == sql.ErrNoRows {
		res = result.SetErrorResult(`You don't have completed games`)
		return
	}
	var ResultScan []Result
	query = `SELECT games.rating.id_user, score, ratediff, coinsdiff, trophies FROM games.rating
		INNER JOIN users.trophies ON users.trophies.id_user = games.rating.id_user
		WHERE id_game = $1`
	params = []any{IDGame}
	rows, err := db.Query(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var r Result
		err = rows.Scan(&r.IDUser, &r.Score, &r.RateDiff, &r.CoinsDiff, &r.Rating)
		if err != nil {
			report.ErrorServer(nil, err)
		}
		ResultScan = append(ResultScan, r)
	}
	if len(ResultScan) != 2 {
		res = result.SetErrorResult(`Error in getting game`)
		report.ErrorServer(nil, errors.New(`error in getting resultscan`))
		return
	}
	if ResultScan[0].IDUser == IDUser {
		ResultGame.AddMoney = ResultScan[0].CoinsDiff
		ResultGame.Rating = ResultScan[0].Rating
		ResultGame.RatingDiff = ResultScan[0].RateDiff
		ResultGame.Score = strconv.Itoa(ResultScan[0].Score) + ":" + strconv.Itoa(ResultScan[1].Score)
		if ResultScan[0].Score > ResultScan[1].Score {
			ResultGame.Result = `YOU WIN`
		} else if ResultScan[0].Score == ResultScan[1].Score {
			ResultGame.Result = `TIE`
		} else {
			ResultGame.Result = `YOU LOSE`
		}
	} else if ResultScan[1].IDUser == IDUser {
		ResultGame.AddMoney = ResultScan[1].CoinsDiff
		ResultGame.Rating = ResultScan[1].Rating
		ResultGame.RatingDiff = ResultScan[1].RateDiff
		ResultGame.Score = strconv.Itoa(ResultScan[1].Score) + ":" + strconv.Itoa(ResultScan[0].Score)
		if ResultScan[1].Score > ResultScan[0].Score {
			ResultGame.Result = `YOU WIN`
		} else if ResultScan[1].Score == ResultScan[0].Score {
			ResultGame.Result = `TIE`
		} else {
			ResultGame.Result = `YOU LOSE`
		}
	} else {
		res = result.SetErrorResult(`Unknown error`)
	}
	res.Done = true
	res.Items = ResultGame
	return
}

func GetHintsPrices() (res result.ResultInfo) {
	var Prices RatingHintPricesGlobal
	db := config.ConnectDB()
	query := `SELECT id, price FROM hints.prices ORDER BY id ASC`
	rows, err := db.Query(query)
	if err != nil {
		report.ErrorSQLServer(nil, err, query)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, price int
		err = rows.Scan(&id, &price)
		if err != nil {
			report.ErrorServer(nil, err)
		}
		switch id {
		case 1:
			Prices.Red.Age = price
		case 2:
			Prices.Red.League = price
		case 3:
			Prices.Red.Nation = price
		case 4:
			Prices.Red.Price = price
		case 5:
			Prices.Yellow.Age = price
		case 6:
			Prices.Yellow.League = price
		case 7:
			Prices.Yellow.Nation = price
		case 8:
			Prices.Yellow.Position = new(int)
			*Prices.Yellow.Position = price
		case 9:
			Prices.Yellow.Price = price
		case 10:
			Prices.Green.Age = price
		case 11:
			Prices.Green.Club = new(int)
			*Prices.Green.Club = price
		case 12:
			Prices.Green.League = price
		case 13:
			Prices.Green.Nation = price
		case 14:
			Prices.Green.Position = new(int)
			*Prices.Green.Position = price
		case 15:
			Prices.Green.Price = price
		}
	}
	res.Done = true
	res.Items = Prices
	return
}

func PutHint(user config.User, IDColor int, IDType int) (res result.ResultInfo) {
	db := config.ConnectDB()
	IDGlobalGame := CheckRatingGameExist(user)
	if IDGlobalGame == -1 {
		res = result.SetErrorResult(`Error in searching current game`)
		return
	}
	if IDGlobalGame == 0 {
		res = result.SetErrorResult(`Please, create new game`)
		return
	}
	IDLocalGame := CheckRatingGamePartExist(user)
	if IDLocalGame == -1 {
		res = result.SetErrorResult(`Error in searching local game`)
		return
	}
	if IDLocalGame == 0 {
		res = result.SetErrorResult(`This game doesn't exist`)
		return
	}
	var IDHint, PriceHint int
	query := `SELECT id, price FROM hints.prices WHERE color = $1 AND type = $2`
	params := []any{IDColor, IDType}
	err := db.QueryRow(query, params...).Scan(&IDHint, &PriceHint)
	if err != nil && err != sql.ErrNoRows {
		res = result.SetErrorResult(DATABASE_ERROR)
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	if err == sql.ErrNoRows {
		res = result.SetErrorResult(`No hints with this params`)
		return
	}
	var LastColor int
	query = `SELECT color FROM hints.prices
		INNER JOIN hints.game ON id_hint = hints.prices.id
		WHERE type = $1 and id_local_game = $2 ORDER BY hints.game.id DESC LIMIT 1`
	params = []any{IDType, IDLocalGame}
	err = db.QueryRow(query, params...).Scan(&LastColor)
	if err != nil && err != sql.ErrNoRows {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	switch LastColor {
	case GREY:
	case YELLOW:
		if IDColor == YELLOW || IDColor == RED {
			res = result.SetErrorResult(`Can't use this hint(yellow)`)
		}
	case GREEN:
		res = result.SetErrorResult(`Can't use this hint(green)`)
	case RED:
		if IDColor == RED {
			res = result.SetErrorResult(`Can't use this hint(red)`)
		}
	}
	Coins := GetUserCoins(user.ID)
	if Coins < PriceHint {
		res = result.SetErrorResult(`You don't have enough coins for this hint`)
		return
	}
	query = `INSERT INTO hints.game (id_global_game, id_local_game, id_user, id_hint) VALUES ($1, $2, $3, $4)`
	params = []any{IDGlobalGame, IDLocalGame, user.ID, IDHint}
	_, err = db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	query = `UPDATE users.trophies SET coins = coins - $1 WHERE id_user = $2`
	params = []any{PriceHint, user.ID}
	_, err = db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	res.Done = true
	return
}

func GetHintOpponent(user config.User) (res result.ResultInfo) {
	db := config.ConnectDB()
	var h HintOpponent
	IDGlobalGame := CheckRatingGameExist(user)
	if IDGlobalGame == -1 {
		res = result.SetErrorResult(`Error in searching current game`)
		return
	}
	if IDGlobalGame == 0 {
		res = result.SetErrorResult(`Please, create new game`)
		return
	}
	var IDColor, IDHint int
	query := `SELECT color, hints.game.id FROM hints.prices 
		INNER JOIN hints.game ON id_hint = hints.prices.id
		WHERE id_global_game = $1 AND shared = FALSE ORDER BY hints.game.id ASC LIMIT 1`
	params := []any{IDGlobalGame}
	err := db.QueryRow(query, params...).Scan(&IDColor, &IDHint)
	if err != nil && err != sql.ErrNoRows {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	if err == sql.ErrNoRows {
		h.Exist = false
		res.Done = true
		res.Items = h
		return
	}
	query = `UPDATE hints.game SET shared = TRUE WHERE id = $1`
	params = []any{IDHint}
	_, err = db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	h.Color = new(int)
	h.Color = &IDColor
	h.Exist = true
	res.Done = true
	res.Items = h
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
		daemon.GamesList.Mutex.Lock()
		for i := 0; i < len(daemon.GamesList.Games); i++ {
			if daemon.GamesList.Games[i].ID == IDGame {
				daemon.GamesList.RemoveElements(i)
				break
			}
		}
		daemon.GamesList.Mutex.Unlock()
		query = `UPDATE games.rating_pairs SET active = FALSE WHERE id = $1 RETURNING user1, user2`
		params = []any{IDGame}
		err = db.QueryRow(query, params...).Scan(&User1, &User2)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
		}
		ChangeWin, ChangeLose := CountRateDiff(IDGame)
		UpdateGameCoins(IDGame)
		var Change1, Change2 int
		query = `UPDATE games.rating SET ratediff = $1 WHERE id_game = $2 AND id_user = $3`
		if IDUser == User1 {
			params = []any{ChangeWin, IDGame, User1}
			Change1 = ChangeWin
		} else {
			params = []any{ChangeWin, IDGame, User2}
			Change2 = ChangeWin
		}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			return
		}
		if IDUser == User1 {
			params = []any{ChangeLose, IDGame, User2}
			Change2 = ChangeLose
		} else {
			params = []any{ChangeLose, IDGame, User1}
			Change1 = ChangeLose
		}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			return
		}
		query = `UPDATE users.trophies SET trophies = trophies + $1, updated_at = $2 WHERE id_user = $3`
		params = []any{Change1, time.Now(), User1}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			return
		}
		params = []any{Change2, time.Now(), User2}
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

func FinishingGames() {
	db := config.ConnectDB()
	type ScorePair struct {
		ID    int
		Score int
	}
	for {
		daemon.FinishedList.Mutex.Lock()
		for i := 0; i < len(daemon.FinishedList.Games); i++ {
			IDGame := daemon.FinishedList.Games[i]
			query := `UPDATE games.rating_pairs SET active = FALSE WHERE id = $1`
			params := []any{IDGame}
			_, err := db.Exec(query, params...)
			if err != nil {
				report.ErrorSQLServer(nil, err, query, params...)
			}
			ChangeWin, ChangeLose := CountRateDiff(IDGame)
			UpdateGameCoins(IDGame)
			var Change1, Change2 int
			var Pairs []ScorePair
			query = `SELECT id_user, score FROM games.rating WHERE id_game = $1`
			rows, err := db.Query(query, params...)
			if err != nil {
				report.ErrorSQLServer(nil, err, query, params...)
			}
			defer rows.Close()
			for rows.Next() {
				var s ScorePair
				err = rows.Scan(&s.ID, &s.Score)
				if err != nil {
					report.ErrorServer(nil, err)
				}
				Pairs = append(Pairs, s)
			}
			if len(Pairs) != 2 {
				report.ErrorServer(nil, errors.New(`len(Pairs) != 2`))
				continue
			}
			query = `UPDATE games.rating SET ratediff = $1 WHERE id_game = $2 AND id_user = $3`
			if Pairs[0].Score > Pairs[1].Score {
				params = []any{ChangeWin, IDGame, Pairs[0].ID}
				Change1 = ChangeWin
			} else {
				params = []any{ChangeWin, IDGame, Pairs[1].ID}
				Change2 = ChangeWin
			}
			_, err = db.Exec(query, params...)
			if err != nil {
				report.ErrorSQLServer(nil, err, query, params...)
				return
			}
			if Pairs[0].Score > Pairs[1].Score {
				params = []any{ChangeLose, IDGame, Pairs[1].ID}
				Change2 = ChangeLose
			} else {
				params = []any{ChangeLose, IDGame, Pairs[0].ID}
				Change1 = ChangeLose
			}
			_, err = db.Exec(query, params...)
			if err != nil {
				report.ErrorSQLServer(nil, err, query, params...)
				return
			}
			if Pairs[0].Score == Pairs[1].Score {
				var Trophies1, Trophies2 int
				query1 := `SELECT trophies FROM users.trophies WHERE id_user = $1`
				params = []any{Pairs[0].ID}
				err = db.QueryRow(query1, params...).Scan(&Trophies1)
				if err != nil {
					report.ErrorSQLServer(nil, err, query, params...)
				}
				params = []any{Pairs[1].ID}
				err = db.QueryRow(query1, params...).Scan(&Trophies2)
				if err != nil {
					report.ErrorSQLServer(nil, err, query, params...)
				}
				if Trophies1 < Trophies2 {
					params = []any{ChangeWin, IDGame, Pairs[0].ID}
					Change1 = ChangeWin
				} else {
					params = []any{ChangeWin, IDGame, Pairs[1].ID}
					Change2 = ChangeWin
				}
				_, err = db.Exec(query, params...)
				if err != nil {
					report.ErrorSQLServer(nil, err, query, params...)
					return
				}
				if Trophies1 < Trophies2 {
					params = []any{ChangeLose, IDGame, Pairs[1].ID}
					Change2 = ChangeLose
				} else {
					params = []any{ChangeLose, IDGame, Pairs[0].ID}
					Change1 = ChangeLose
				}
				_, err = db.Exec(query, params...)
				if err != nil {
					report.ErrorSQLServer(nil, err, query, params...)
					return
				}
			}
			query = `UPDATE users.trophies SET trophies = trophies + $1, updated_at = $2 WHERE id_user = $3`
			params = []any{Change1, time.Now(), Pairs[0].ID}
			_, err = db.Exec(query, params...)
			if err != nil {
				report.ErrorSQLServer(nil, err, query, params...)
				return
			}
			params = []any{Change2, time.Now(), Pairs[1].ID}
			_, err = db.Exec(query, params...)
			if err != nil {
				report.ErrorSQLServer(nil, err, query, params...)
				return
			}
			daemon.FinishedList.RemoveElements(i)
			i--
		}
		daemon.FinishedList.Mutex.Unlock()
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
	k := (math.Log(float64(Diff+200)) * 1.5) - 6.94747605
	var Sum float64 = 60
	Sum /= (k + 1)
	var k2 float64 = 1.
	if User[1].Score > User[0].Score {
		ChangeWin = int(math.Round(Sum)) + 1
		if User[0].Trophs < 1000 {
			k2 = float64((User[0].Trophs)/10) / 100
		}
		if User[0].Trophs-int(Sum*k2) < 0 {
			ChangeLose = -User[0].Trophs
		} else {
			ChangeLose = -int(math.Round(Sum * k2))
		}
	} else if User[1].Score < User[0].Score {
		ChangeWin = int(math.Round(Sum*k)) + 1
		if User[1].Trophs < 1000 {
			k2 = float64((User[1].Trophs)/10) / 100
		}
		if User[1].Trophs-int(Sum*k2*k) < 0 {
			ChangeLose = -User[1].Trophs
		} else {
			ChangeLose = -int(math.Round(Sum * k2 * k))
		}
	} else if User[1].Score == User[0].Score {
		ChangeWin = int(math.Round(Sum*k)) - 30
		if User[1].Trophs < 1000 {
			k2 = float64((User[0].Trophs)/10) / 100
		}
		if User[1].Trophs-int(Sum*k*k2)+int(math.Round(30*k2)) < 0 {
			ChangeLose = -User[0].Trophs + int(math.Round(30*k2))
		} else {
			ChangeLose = -int(math.Round(Sum*k2*k)) + int(math.Round(30*k2))
		}

	}
	return
}

func UpdateGameCoins(IDGame int) {
	db := config.ConnectDB()
	type UserScore struct {
		ID    int
		Score int
	}
	var Player []UserScore
	query := `SELECT id_user, score FROM games.rating WHERE id = $1`
	params := []any{IDGame}
	rows, err := db.Query(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
	}
	defer rows.Close()
	for rows.Next() {
		var u UserScore
		err = rows.Scan(&u.ID, &u.Score)
		if err != nil {
			report.ErrorServer(nil, err)
		}
		params = append(params, u)
	}
	if len(Player) != 2 {
		report.ErrorServer(nil, errors.New(`must be 2 players in game`))
		return
	}
	p0 := Player[0]
	p1 := Player[1]
	query = `UPDATE games.rating SET coinsdiff = $1 WHERE id_user = $2`
	query1 := `UPDATE users.trophies SET coins = coins + $1 WHERE id_user = $2`
	if p0.Score > p1.Score {
		params = []any{p0.Score + 1, p0.ID}
	} else {
		params = []any{p0.Score, p0.ID}
	}
	_, err = db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
	}
	_, err = db.Exec(query1, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query1, params...)
	}
	if p1.Score > p0.Score {
		params = []any{p1.Score + 1, p1.ID}
	} else {
		params = []any{p1.Score, p1.ID}
	}
	_, err = db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
	}
	_, err = db.Exec(query1, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query1, params...)
	}
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

func RatingGameTimeFinish(IDGame int) string {
	db := config.ConnectDB()
	var t time.Time
	query := `SELECT created_at FROM games.rating_pairs WHERE id = $1`
	params := []any{IDGame}
	err := db.QueryRow(query, params...).Scan(&t)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return ``
	}
	return t.Add(10 * time.Minute).Format("15:04:05")
}
