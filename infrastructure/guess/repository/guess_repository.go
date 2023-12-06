package infrastracture

import (
	"database/sql"
	"errors"
	"futble/config"
	"futble/constants"
	repository "futble/domain/guess"
	"futble/entity"
	"futble/report"
	"time"
)

type GuessRepository struct {
}

func NewGuessRepository() repository.GuessRepository {
	return &GuessRepository{}
}

func (this GuessRepository) GetGuessesByGame(IDGame int) ([]entity.Guess, error) {
	var Guesses []entity.Guess
	db := config.ConnectDB()
	query := `SELECT id_guess FROM games.guess WHERE id_game = $1 ORDER BY id ASC`
	params := []any{IDGame}
	rows, err := db.Query(query, params...)
	if err != nil {
		report.ErrorServer(nil, err)
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var Guess entity.Guess
		err := rows.Scan(&Guess.ID)
		if err != nil {
			report.ErrorServer(nil, err)
			return nil, err
		}
		Guesses = append(Guesses, Guess)
	}
	return Guesses, nil
}

func (this GuessRepository) GetAnswer(IDGame int) (*entity.Guess, error) {
	var AnswerID int
	db := config.ConnectDB()
	query := `SELECT id_answer FROM games.list WHERE id = $1`
	params := []any{IDGame}
	err := db.QueryRow(query, params...).Scan(&AnswerID)
	if err != nil {
		report.ErrorServer(nil, err)
		return &entity.Guess{}, err
	}
	return &entity.Guess{
		ID: AnswerID,
	}, nil
}

func (this *GuessRepository) CheckGuess(IDPlayer int, IDAnswer int) (entity.Guess, error) {
	var a entity.Guess
	var Guess, Answer entity.Player
	Guess.ID = IDPlayer
	Answer.ID = IDAnswer
	db := config.ConnectDB()
	query := `SELECT name, surname, birth, players.data.club, league, players.data.nation, position, price, c.short, n.short FROM players.data
		INNER JOIN players.club c ON c.club = players.data.club
		INNER JOIN players.nation n ON n.country = players.data.nation
		WHERE players.data.id = $1`
	params := []any{IDPlayer}
	err := db.QueryRow(query, params...).Scan(&Guess.Name, &Guess.Surname, &Guess.Birth, &Guess.Club, &Guess.League, &Guess.Nation, &Guess.Position, &Guess.Price, &Guess.ClubShort, &Guess.NationShort)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return a, err
	}
	params = []any{IDAnswer}
	err = db.QueryRow(query, params...).Scan(&Answer.Name, &Answer.Surname, &Answer.Birth, &Answer.Club, &Answer.League, &Answer.Nation, &Answer.Position, &Answer.Price, &Answer.ClubShort, &Answer.NationShort)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return a, err
	}
	a.ID = IDPlayer
	a.Name = Guess.Name
	a.Surname = Guess.Surname
	a.Age = funcAge(Guess.Birth, time.Now())
	AnswerAge := funcAge(Answer.Birth, time.Now())
	if a.Age == AnswerAge {
		a.AgeColor = entity.GREEN
	} else if abs(a.Age, AnswerAge) == 1 {
		a.AgeColor = entity.YELLOW
	} else if abs(a.Age, AnswerAge) < 10 {
		a.AgeColor = entity.GREY
	} else {
		a.AgeColor = entity.RED
	}
	a.Club = Guess.ClubShort
	if Guess.Club == Answer.Club {
		a.ClubColor = entity.GREEN
	} else {
		a.ClubColor = entity.GREY
	}
	a.League = Guess.League
	a.LeagueColor, err = getLeagueColor(Guess.League, Answer.League)
	if err != nil {
		return a, err
	}
	a.Nation = Guess.NationShort
	a.NationColor = getNationColor(Guess.Nation, Answer.Nation)
	a.Position = Guess.Position
	a.PositionColor = getPositionColor(Guess.Position, Answer.Position)
	a.Price = Guess.Price
	Res := abs(Answer.Price, Guess.Price)
	if Res < 5000000 {
		a.PriceColor = entity.GREEN
	} else if Res < 10000000 {
		a.PriceColor = entity.YELLOW
	} else if Res < 50000000 {
		a.PriceColor = entity.GREY
	} else {
		a.PriceColor = entity.RED
	}
	return a, nil
}

func Add(Guess int, IDGame int) (GameResult int, err error) {
	db := config.ConnectDB()
	var NumOfGuesses int
	GameResult = -10
	query := `SELECT count(*) FROM games.guess WHERE id_game = $1`
	params := []any{IDGame}
	err = db.QueryRow(query, params...).Scan(&NumOfGuesses)
	if err != nil {
		return GameResult, errors.New(`Ошибка при запросе к БД`)
	}
	if NumOfGuesses >= 8 {
		return GameResult, errors.New(`Достигнуто максимальное число попыток(8)`)
	}
	if NumOfGuesses == 0 {
		var Mode int
		Mode, err = getGameTypeByID(IDGame)
		if err != nil {
			report.ErrorServer(nil, err)
			return GameResult, errors.New(`Error in scan game mode`)
		}
		if Mode == entity.DAILY || Mode == entity.UNLIMITED {
			query = `UPDATE games.list SET start_time = $1 WHERE id = $2`
			params = []any{time.Now(), IDGame}
			_, err = db.Exec(query, params...)
			if err != nil {
				return GameResult, errors.New(`Ошибка при запросе к БД`)
			}
		}
	}
	var LastGuess int
	query = `SELECT id_guess FROM games.guess WHERE id_game = $1 ORDER BY id DESC LIMIT 1`
	params = []any{IDGame}
	err = db.QueryRow(query, params...).Scan(&LastGuess)
	if err != nil && err != sql.ErrNoRows {
		return GameResult, errors.New(`Ошибка при запросе к БД`)
	}
	var Answer int
	query = `SELECT id_answer FROM games.list WHERE id = $1`
	err = db.QueryRow(query, params...).Scan(&Answer)
	if err != nil {
		return GameResult, errors.New(`Ошибка при запросе к БД`)
	}
	if LastGuess == Answer {
		return -10, errors.New(`Игрок уже угадан`)
	}
	if Guess == Answer {
		GameResult = entity.WIN
	} else if Guess != Answer && NumOfGuesses == 7 {
		GameResult = entity.LOSE
	} else {
		GameResult = entity.NOTHING
	}
	if GameResult == entity.WIN || GameResult == entity.LOSE {
		var Mode int
		Mode, err = getGameTypeByID(IDGame)
		if err != nil {
			report.ErrorServer(nil, err)
			return GameResult, errors.New(`Error in scan game mode`)
		}
		if Mode == entity.DAILY || Mode == entity.UNLIMITED {
			query = `UPDATE games.list SET finish_time = $1 WHERE id = $2`
			params = []any{time.Now(), IDGame}
			_, err = db.Exec(query, params...)
			if err != nil {
				return GameResult, errors.New(`Ошибка при запросе к БД`)
			}
		}
	}
	query = `INSERT INTO games.guess (id_game, id_guess) VALUES ($1, $2)`
	params = []any{IDGame, Guess}
	_, err = db.Exec(query, params...)
	if err != nil {
		return GameResult, errors.New(`Ошибка при запросе к БД`)
	}
	return
}

func funcAge(birthdate, today time.Time) int {
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

func getLeagueColor(Guess string, Answer string) (int, error) {
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
		return entity.GREEN, nil
	}
	if abs(GuessLeague, AnswerLeague) == 1 {
		return entity.YELLOW, nil
	}
	if GuessLeague/100 != AnswerLeague/100 {
		return entity.RED, nil
	}
	return entity.GREY, nil
}

func getNationColor(Guess string, Answer string) int {
	if Guess == Answer {
		return entity.GREEN
	}
	for i := range constants.NationMatches[Guess] {
		if constants.NationMatches[Guess][i] == Answer {
			return entity.YELLOW
		}
	}
	var GuessContinent, AnswerContinent int
	db := config.ConnectDB()
	query := `SELECT continent FROM players.nation WHERE country = $1`
	params := []any{Guess}
	err := db.QueryRow(query, params...).Scan(&GuessContinent)
	if err != nil {
		report.ErrorServer(nil, err)
	}
	params = []any{Answer}
	err = db.QueryRow(query, params...).Scan(&AnswerContinent)
	if err != nil {
		report.ErrorServer(nil, err)
	}
	if GuessContinent != AnswerContinent {
		return entity.RED
	}
	return entity.GREY
}

func getPositionColor(Guess string, Answer string) int {
	if Guess == Answer {
		return entity.GREEN
	}
	if (Guess == "CF" && Answer == "ST") || (Guess == "ST" && Answer == "CF") {
		return entity.GREEN
	}
	for i := range Matches[Guess] {
		if Matches[Guess][i] == Answer {
			return entity.YELLOW
		}
	}
	return entity.GREY
}

var Matches = map[string][]string{
	"GK":  {"LB", "CB", "RB"},
	"LB":  {"LWB", "CB", "GK"},
	"CB":  {"LB", "RB", "CDM", "GK"},
	"RB":  {"RWB", "CB", "GK"},
	"LWB": {"LB", "LM", "CDM"},
	"CDM": {"LWB", "RWB", "CB", "CM"},
	"RWB": {"RB", "RM", "CDM"},
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

func abs(a int, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}

func getGameTypeByID(GameID int) (int, error) {
	db := config.ConnectDB()
	var GameType int
	query := `SELECT game_type FROM games.list WHERE id = $1`
	params := []any{GameID}
	err := db.QueryRow(query, params...).Scan(&GameType)
	return GameType, err
}
