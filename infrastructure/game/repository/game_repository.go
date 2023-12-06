package infrastructure

import (
	"database/sql"
	"errors"
	"futble/aggregate"
	"futble/check"
	"futble/config"
	repository "futble/domain/game"
	"futble/entity"
	infrastracture "futble/infrastructure/guess/repository"
	"futble/report"
	"math/rand"
	"time"
)

type GameRepository struct {
}

func NewGameRepository() repository.GameRepository {
	return &GameRepository{}
}

func (this *GameRepository) New(Type int, user entity.User) (*aggregate.Game, error) {
	var ID int
	var err error
	switch Type {
	case entity.DAILY:
		ID, err = createDailyGame(user)
	case entity.RATING:
		ID, err = createRatingGame(user)
	case entity.UNLIMITED:
		ID, err = createUnlimitedGame(user)
	default:
		ID = -1
		err = errors.New(`ты не должен быть здесь`)
	}
	if err != nil {
		return &aggregate.Game{}, err
	}
	return &aggregate.Game{
		Info: entity.GameBasic{
			ID:       ID,
			GameMode: Type,
		},
	}, nil
}

func (this *GameRepository) Get(ID int, Hintlist ...any) (*aggregate.Game, error) {
	var guesses []entity.Guess
	var gameBasic entity.GameBasic
	guessRep := infrastracture.NewGuessRepository()
	Guesses, err := guessRep.GetGuessesByGame(ID)
	if err != nil {
		return &aggregate.Game{}, err
	}
	Answer, err := guessRep.GetAnswer(ID)
	if err != nil {
		return &aggregate.Game{}, err
	}
	for i := range Guesses {
		Guess, err := guessRep.CheckGuess(Guesses[i].ID, Answer.ID)
		if err != nil {
			report.ErrorServer(nil, err)
			return &aggregate.Game{}, err
		}
		guesses = append(guesses, Guess)
	}
	var Hints []entity.Hint
	for i := 0; i < len(Hintlist); i++ {
		hint, ok := Hintlist[i].(entity.Hint)
		if !ok {
			report.ErrorServer(nil, errors.New(`error in getting hints`))
			break
		}
		Hints = append(Hints, hint)
	}
	if len(Hintlist) > 0 {
		for i := len(Guesses); i < 8; i++ {
			Guess, err := mockGetAnswerWithHints(ID, Answer.ID, Hints)
			if err != nil {
				report.ErrorServer(nil, err)
				return &aggregate.Game{}, err
			}
			guesses = append(guesses, Guess)
		}
	}
	gameBasic.ID = ID
	gameBasic.TimeStart = new(string)
	Mode, err := getGameTypeByID(ID)
	if err != nil {
		return &aggregate.Game{}, err
	}
	if Mode == entity.DAILY || Mode == entity.UNLIMITED {
		*gameBasic.TimeStart, err = getTimeStart(ID)
		if err != nil {
			return &aggregate.Game{}, err
		}
	}
	gameBasic.GameMode = Mode
	return &aggregate.Game{
		Info:    gameBasic,
		Guesses: guesses,
	}, nil
}

func createDailyGame(user entity.User) (int, error) {
	db := config.ConnectDB()
	var Answer, Day, ID int
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
}

func createRatingGame(user entity.User) (int, error) {
	db := config.ConnectDB()
	var IDGame, tries, ID int
	query := `SELECT id_game, tries FROM games.rating WHERE id_user = $1 ORDER BY id desc LIMIT 1`
	params := []any{user.ID}
	err := db.QueryRow(query, params...).Scan(&IDGame, &tries)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return -1, err
	}
	var IDAnswer int
	query = `SELECT id_answer FROM games.rating_answers WHERE id_game = $1 AND ans_in_game = $2`
	params = []any{IDGame, tries + 1}
	err = db.QueryRow(query, params...).Scan(&IDAnswer)
	if err != nil && err != sql.ErrNoRows {
		report.ErrorSQLServer(nil, err, query, params...)
		return -1, err
	}
	if err == sql.ErrNoRows {
		IDAnswer = createNewAnswer(IDGame, tries+1)
	}
	query = `INSERT INTO games.list (game_type, id_user, id_answer, active) VALUES (2, $1, $2, TRUE) RETURNING id`
	params = []any{user.ID, IDAnswer}
	err = db.QueryRow(query, params...).Scan(&ID)
	if err != nil {
		report.ErrorServer(nil, err)
		return -1, err
	}
	return ID, nil
}

func createUnlimitedGame(user entity.User) (int, error) {
	db := config.ConnectDB()
	var ID int
	PlayerPos := rand.Intn(len(check.IDs))
	IDAnswer := check.IDs[PlayerPos]
	query := `INSERT INTO games.list (game_type, id_user, id_answer, active) VALUES (3, $1, $2, TRUE) RETURNING id`
	params := []any{user.ID, IDAnswer}
	err := db.QueryRow(query, params...).Scan(&ID)
	if err != nil {
		report.ErrorServer(nil, err)
		return -1, err
	}
	return ID, nil
}

func createNewAnswer(IDGame int, AnsInGame int) int {
	db := config.ConnectDB()
	PlayerPos := rand.Intn(len(check.IDs))
	IDAnswer := check.IDs[PlayerPos]
	query := `INSERT INTO games.rating_answers (id_game, ans_in_game, id_answer) VALUES ($1, $2, $3)`
	params := []any{IDGame, AnsInGame, IDAnswer}
	_, err := db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return -1
	}
	return IDAnswer
}

func getGameTypeByID(GameID int) (int, error) {
	db := config.ConnectDB()
	var GameType int
	query := `SELECT game_type FROM games.list WHERE id = $1`
	params := []any{GameID}
	err := db.QueryRow(query, params...).Scan(&GameType)
	return GameType, err
}

func getTimeStart(GameID int) (string, error) {
	db := config.ConnectDB()
	var TimeStart *time.Time
	query := `SELECT start_time FROM games.list WHERE id = $1`
	params := []any{GameID}
	err := db.QueryRow(query, params...).Scan(&TimeStart)
	if err != nil {
		return ``, err
	}
	if TimeStart == nil {
		return time.Now().Format("02-01-2006 15:04:05"), nil
	}
	return TimeStart.Format("02-01-2006 15:04:05"), nil
}

func mockGetAnswerWithHints(a int, b int, h []entity.Hint) (entity.Guess, error) {
	return entity.Guess{}, nil
}
