package game

import (
	"futble/config"
	"futble/constants"
	"futble/report"
	"futble/result"
	"math/rand"
	"net/http"
	"time"
)

var DAILY int = 1
var RATING int = 2
var UNLIMITED int = 3

var GREY int = 0
var YELLOW int = 1
var GREEN int = 2

type PlayerData struct {
	ID       int
	Name     string
	Surname  string
	Club     string
	League   string
	Nation   string
	Position string
	Price    int
	Birth    time.Time
}

type SearchPlayers struct {
	Surname []string `json:"surname"`
}

func CreateGame(Type int, user config.User) (int, error) {
	db := config.ConnectDB()
	var ID int
	switch Type {
	case DAILY:
		var Answer, Day int
		var DayFinish time.Time
		query := `SELECT id_answer, id, day_finish FROM games.daily_answers WHERE day_start < $1 AND day_finish > $1`
		params := []any{time.Now()}
		err := db.QueryRow(query, params...).Scan(&Answer, &Day, &DayFinish)
		if err != nil {
			report.ErrorServer(nil, err)
			return -1, err
		}
		query = `INSERT INTO games.list (game_type, id_user, id_answer) VALUES (1, $1, $2) RETURNING id`
		params = []any{user.ID, Answer}
		err = db.QueryRow(query, params...).Scan(&ID)
		if err != nil {
			report.ErrorServer(nil, err)
			return -1, err
		}
		query = `INSERT INTO games.daily (id_game, id_user, end_time) VALUES ($1, $2, $3)`
		params = []any{ID, user.ID, DayFinish}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorServer(nil, err)
			return -1, err
		}
		return ID, nil
	case RATING:
		var CountPlayers int
		var IsLogged bool
		query := `SELECT not_logged FROM users.accounts WHERE id = $1`
		params := []any{user.ID}
		err := db.QueryRow(query, params...).Scan(&IsLogged)
		if err != nil {
			report.ErrorServer(nil, err)
			return -1, err
		}
		if !IsLogged {
			return -2, err
		}
		query = `SELECT COUNT(*) FROM players.data`
		err = db.QueryRow(query).Scan(&CountPlayers)
		if err != nil {
			report.ErrorServer(nil, err)
			return -1, err
		}
		IDAnswer := rand.Intn(CountPlayers)
		query = `INSERT INTO games.list (game_type, id_user, id_answer) VALUES (2, $1, $2) RETURNING id`
		params = []any{user.ID, IDAnswer}
		err = db.QueryRow(query, params...).Scan(&ID)
		if err != nil {
			report.ErrorServer(nil, err)
			return -1, err
		}
		var DayFinish time.Time
		query = `SELECT day_finish FROM games.daily_answers WHERE day_start < $1 AND day_finish > $1`
		params = []any{time.Now()}
		err = db.QueryRow(query, params...).Scan(&DayFinish)
		if err != nil {
			report.ErrorServer(nil, err)
			return -1, err
		}
		query = `INSERT INTO games.rating (id_game, id_user, end_time) VALUES ($1, $2, $3)`
		params = []any{ID, user.ID, DayFinish}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorServer(nil, err)
			return -1, err
		}
		return ID, nil
	case UNLIMITED:
		var CountPlayers int
		query := `SELECT COUNT(*) FROM players.data`
		err := db.QueryRow(query).Scan(&CountPlayers)
		if err != nil {
			report.ErrorServer(nil, err)
			return -1, err
		}
		IDAnswer := rand.Intn(CountPlayers)
		query = `INSERT INTO games.list (game_type, id_user, id_answer, active) VALUES (3, $1, $2, TRUE) RETURNING id`
		params := []any{user.ID, IDAnswer}
		err = db.QueryRow(query, params...).Scan(&ID)
		if err != nil {
			report.ErrorServer(nil, err)
			return -1, err
		}
		return ID, nil

	}
	return -1, nil
}

func GameInfoCollect(ID int) (Game, error) {
	var res Game
	var GuessIDs []int
	db := config.ConnectDB()
	query := `SELECT id_guess FROM games.guess WHERE id_game = $1`
	params := []any{ID}
	rows, err := db.Query(query, params...)
	if err != nil {
		report.ErrorServer(nil, err)
		return res, err
	}
	defer rows.Close()
	for rows.Next() {
		var ID int
		err := rows.Scan(&ID)
		if err != nil {
			report.ErrorServer(nil, err)
			return res, err
		}
		GuessIDs = append(GuessIDs, ID)
	}
	var AnswerID int
	query = `SELECT id_answer FROM games.list WHERE id = $1`
	params = []any{ID}
	err = db.QueryRow(query, params...).Scan(&AnswerID)
	if err != nil {
		report.ErrorServer(nil, err)
		return res, err
	}
	for i := range GuessIDs {
		Answer, err := CheckRecord(GuessIDs[i], AnswerID)
		if err != nil {
			report.ErrorServer(nil, err)
			return res, err
		}
		res.Answers = append(res.Answers, Answer)
	}
	res.ID = ID
	return res, nil
}

func CheckRecord(IDPlayer int, IDAnswer int) (AnswerType, error) {
	var a AnswerType
	var Guess, Answer PlayerData
	Guess.ID = IDPlayer
	Answer.ID = IDAnswer
	db := config.ConnectDB()
	query := `SELECT name, surname, birth, club, league, nation, position, price FROM players.data WHERE id = $1`
	params := []any{IDPlayer}
	err := db.QueryRow(query, params...).Scan(&Guess.Name, &Guess.Surname, &Guess.Birth, &Guess.Club, &Guess.League, &Guess.Nation, &Guess.Position, &Guess.Price)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return a, err
	}
	params = []any{IDAnswer}
	err = db.QueryRow(query, params...).Scan(&Answer.Name, &Answer.Surname, &Answer.Birth, &Answer.Club, &Answer.League, &Answer.Nation, &Answer.Position, &Answer.Price)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return a, err
	}
	a.ID = IDPlayer
	a.Name = Guess.Name
	a.Surname = Guess.Surname
	a.Age = FuncAge(Guess.Birth, time.Now())
	AnswerAge := FuncAge(Answer.Birth, time.Now())
	if a.Age == AnswerAge {
		a.AgeColor = GREEN
	} else if abs(a.Age, AnswerAge) == 1 {
		a.AgeColor = YELLOW
	} else {
		a.AgeColor = GREY
	}
	a.Club = Guess.Club
	if Guess.Club == Answer.Club {
		a.ClubColor = GREEN
	} else if Guess.Club[0] == Answer.Club[0] {
		a.ClubColor = YELLOW
	} else {
		a.ClubColor = GREY
	}
	a.League = Guess.League
	a.LeagueColor, err = GetLeagueColor(Guess.League, Answer.League)
	if err != nil {
		return a, err
	}
	a.Nation = Guess.Nation
	a.NationColor = GetNationColor(Guess.Nation, Answer.Nation)
	a.Position = Guess.Position
	a.PositionColor = GetPositionColor(Guess.Position, Answer.Position)
	a.Price = Guess.Price
	Res := float64(Answer.Price) / float64(Guess.Price)
	if Res > 0.9 && Res < 1.1 {
		a.PriceColor = GREEN
	} else if Res > 0.7 && Res < 1.3 {
		a.PriceColor = YELLOW
	} else {
		a.PriceColor = GREY
	}
	return a, nil
}

func FuncAge(birthdate, today time.Time) int {
	today = today.In(birthdate.Location())
	ty, tm, td := today.Date()
	today = time.Date(ty, tm, td, 0, 0, 0, 0, time.UTC)
	by, bm, bd := birthdate.Date()
	birthdate = time.Date(by, bm, bd, 0, 0, 0, 0, time.UTC)
	if today.Before(birthdate) {
		return 0
	}
	age := ty - by
	anniversary := birthdate.AddDate(age, 0, 0)
	if anniversary.After(today) {
		age--
	}
	return age
}

func abs(a int, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}

func GetLeagueColor(Guess string, Answer string) (int, error) {
	var GuessLeague, AnswerLeague int
	db := config.ConnectDB()
	query := `SELECT place FROM players.league WHERE league = $1`
	params := []any{Guess}
	err := db.QueryRow(query, params...).Scan(&GuessLeague)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return -1, err
	}
	params = []any{Answer}
	err = db.QueryRow(query, params...).Scan(&AnswerLeague)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return -1, err
	}
	if GuessLeague == AnswerLeague {
		return GREEN, nil
	}
	if abs(GuessLeague, AnswerLeague) == 1 {
		return YELLOW, nil
	}
	return GREY, nil
}

func GetNationColor(Guess string, Answer string) int {
	if Guess == Answer {
		return GREEN
	}
	for i := range constants.NationMatches[Guess] {
		if constants.NationMatches[Guess][i] == Answer {
			return YELLOW
		}
	}
	return GREY
}

func GetPositionColor(Guess string, Answer string) int {
	if Guess == Answer {
		return GREEN
	}
	if (Guess == "CF" && Answer == "ST") || (Guess == "ST" && Answer == "CF") {
		return GREEN
	}
	for i := range Matches[Guess] {
		if Matches[Guess][i] == Answer {
			return YELLOW
		}
	}
	return GREY
}

var Matches = map[string][]string{
	"GK":  {},
	"LD":  {"LWB", "CD"},
	"CD":  {"LB", "RD", "CDM"},
	"RD":  {"RWB", "CD"},
	"LWB": {"LD", "LM", "CDM"},
	"CDM": {"LWB", "RWB", "CD", "CM"},
	"RWB": {"RD", "RM", "CDM"},
	"LM":  {"LWB", "LW", "CM"},
	"CM":  {"LM", "RM", "CDM", "CAM"},
	"RM":  {"RWB", "RW", "CM"},
	"LW":  {"LM", "LF", "CAM"},
	"CAM": {"CM", "CF", "ST", "LW", "RW"},
	"RW":  {"RM", "RF", "CAM"},
	"LF":  {"LW", "CF", "ST"},
	"CF":  {"CAM", "LF", "RF"},
	"ST":  {"CAM", "LF", "RF"},
	"RF":  {"RW", "CF", "ST"},
}

func PutGuess(Guess int, IDGame int) (res result.ResultInfo, GameResult int, err error) {
	db := config.ConnectDB()
	var NumOfGuesses int
	GameResult = -10
	query := `SELECT count(*) FROM games.guess WHERE id_game = $1`
	params := []any{IDGame}
	err = db.QueryRow(query, params...).Scan(&NumOfGuesses)
	if err != nil {
		res = result.SetErrorResult(`Ошибка при запросе к БД`)
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	if NumOfGuesses >= 8 {
		res = result.SetErrorResult(`Достигнуто максимальное число попыток(8)`)
		return
	}
	var LastGuess int
	query = `SELECT id_guess FROM games.guess WHERE id_game = $1 ORDER BY id DESC LIMIT 1`
	err = db.QueryRow(query, params...).Scan(&LastGuess)
	if err != nil {
		res = result.SetErrorResult(`Ошибка при запросе к БД`)
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	var Answer int
	query = `SELECT id_answer FROM games.list WHERE id = $1`
	err = db.QueryRow(query, params...).Scan(&Answer)
	if err != nil {
		res = result.SetErrorResult(`Ошибка при запросе к БД`)
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	if LastGuess == Answer {
		res = result.SetErrorResult(`Игрок уже угадан`)
		GameResult = -10
		return
	}
	if Guess == Answer {
		GameResult = WIN
	} else if Guess != Answer && NumOfGuesses == 7 {
		GameResult = LOSE
	} else {
		GameResult = NOTHING
	}
	query = `INSERT INTO games.guess (id_game, id_guess) VALUES ($1, $2)`
	params = []any{IDGame, Guess}
	_, err = db.Exec(query, params...)
	if err != nil {
		res = result.SetErrorResult(`Ошибка при запросе к БД`)
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
	return
}

func FindPlayerHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	var s SearchPlayers
	db := config.ConnectDB()
	keys := r.URL.Query()
	PlayerFind := keys[`name`][0]
	query := `SELECT surname FROM players.data WHERE surname like $1`
	params := []any{PlayerFind + "%"}
	rows, err := db.Query(query, params...)
	if err != nil {
		res = result.SetErrorResult(`Ошибка при поиске игроков`)
		report.ErrorServer(r, nil)
		result.ReturnJSON(w, &res)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var surname string
		err = rows.Scan(&surname)
		if err != nil {
			report.ErrorServer(r, err)
			return
		}
		s.Surname = append(s.Surname, surname)
	}
	res.Done = true
	res.Items = s
	result.ReturnJSON(w, &res)
}
