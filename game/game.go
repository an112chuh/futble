package game

/*func GetAnswerWithHints(IDGame int, IDAnswer int, Hints []Hint) (AnswerType, error) {
	var a AnswerType
	var err error
	var Answer PlayerData
	db := config.ConnectDB()
	query := `SELECT name, surname, birth, players.data.club, league, players.data.nation, position, price, c.short, n.short FROM players.data
		INNER JOIN players.club c ON c.club = players.data.club
		INNER JOIN players.nation n ON n.country = players.data.nation
		WHERE players.data.id = $1`
	params := []any{IDAnswer}
	err = db.QueryRow(query, params...).Scan(&Answer.Name, &Answer.Surname, &Answer.Birth, &Answer.Club, &Answer.League, &Answer.Nation, &Answer.Position, &Answer.Price, &Answer.ClubShort, &Answer.NationShort)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return a, err
	}
	for i := 0; i < len(Hints); i++ {
		Hint := Hints[i]
		switch Hint.Type {
		case AGE:
			switch Hint.Color {
			case RED:
				AnswerAge := FuncAge(Answer.Birth, time.Now())
				Diff := IDGame % 3
				if AnswerAge >= 30 {
					a.Age = AnswerAge - 10 - Diff
				} else if AnswerAge <= 27 {
					a.Age = AnswerAge + 10 + Diff
				} else {
					Change := IDGame % 2
					if Change == 0 {
						a.Age = AnswerAge - 10 - Diff
					} else {
						a.Age = AnswerAge + 10 + Diff
					}
				}
				a.AgeColor = RED
			case YELLOW:
				AnswerAge := FuncAge(Answer.Birth, time.Now())
				side := IDGame % 2
				if side == 0 {
					a.Age = AnswerAge + 1
				} else {
					a.Age = AnswerAge - 1
				}
				a.AgeColor = YELLOW
			case GREEN:
				AnswerAge := FuncAge(Answer.Birth, time.Now())
				a.Age = AnswerAge
				a.AgeColor = GREEN
			default:
				err = errors.New(`error in getting hint color`)
				return a, err
			}
		case CLUB:
			switch Hint.Color {
			case GREEN:
				a.Club = Answer.ClubShort
				a.ClubColor = GREEN
			default:
				err = errors.New(`error in getting hint color`)
				return a, err
			}
		case LEAGUE:
			switch Hint.Color {
			case RED:
				ArrayNations := []int{0, 2, 4, 5}
				var LeagueID int
				query = `SELECT place FROM players.league WHERE league = $1`
				params = []any{Answer.League}
				err = db.QueryRow(query, params...).Scan(&LeagueID)
				if err != nil {
					report.ErrorSQLServer(nil, err, query, params...)
					return a, err
				}
				LeagueID /= 100
				rand := IDGame % 3
				LeagueBorder := ArrayNations[rand]
				if LeagueBorder == LeagueID {
					LeagueBorder = ArrayNations[rand+1]
				}
				var count int
				query = `SELECT COUNT(*) FROM players.league WHERE place > $1 AND place < $2`
				params = []any{LeagueBorder * 100, (LeagueBorder + 1) * 100}
				err = db.QueryRow(query, params...).Scan(&count)
				if err != nil {
					report.ErrorSQLServer(nil, err, query, params...)
					return a, err
				}
				rand = LeagueBorder*100 + IDGame%count + 1
				var League string
				query = `SELECT league FROM players.league WHERE place = $1`
				params = []any{rand}
				err = db.QueryRow(query, params...).Scan(&League)
				if err != nil {
					report.ErrorSQLServer(nil, err, query, params...)
					return a, err
				}
				a.League = League
				a.LeagueColor = RED
			case YELLOW:
				var LeagueID int
				query = `SELECT place FROM players.league WHERE league = $1`
				params = []any{Answer.League}
				err = db.QueryRow(query, params...).Scan(&LeagueID)
				if err != nil {
					report.ErrorSQLServer(nil, err, query, params...)
					return a, err
				}
				rand := IDGame & 2
				NewLeague := LeagueID
				if rand == 0 {
					NewLeague--
				} else {
					NewLeague++
				}
				var exists bool
				query = `SELECT EXISTS(SELECT 1 FROM players.league WHERE place = $1)`
				params = []any{NewLeague}
				err = db.QueryRow(query, params...).Scan(&exists)
				if err != nil {
					report.ErrorSQLServer(nil, err, query, params...)
					return a, err
				}
				if !exists {
					if rand == 0 {
						NewLeague += 2
					} else {
						NewLeague -= 2
					}
				}
				var NewLeagueString string
				query = `SELECT league FROM players.league WHERE place = $1`
				params = []any{NewLeague}
				err = db.QueryRow(query, params...).Scan(&NewLeagueString)
				if err != nil {
					report.ErrorSQLServer(nil, err, query, params...)
					return a, err
				}
				a.League = NewLeagueString
				a.LeagueColor = YELLOW
			case GREEN:
				a.League = Answer.League
				a.LeagueColor = GREEN
			default:
				err = errors.New(`error in getting hint color`)
				return a, err
			}
		case NATION:
			switch Hint.Color {
			case RED:
				var Continent int
				query = `SELECT continent FROM players.nation WHERE short = $1`
				params = []any{Answer.NationShort}
				err = db.QueryRow(query, params...).Scan(&Continent)
				if err != nil {
					report.ErrorSQLServer(nil, err, query, params...)
					return a, err
				}
				rand := IDGame % 6
				if rand == Continent {
					if rand > 1 {
						rand--
					} else {
						rand++
					}
				}
				var CountNations int
				query = `SELECT COUNT(*) FROM players.nation WHERE continent = $1`
				params = []any{rand}
				err = db.QueryRow(query, params...).Scan(&CountNations)
				if err != nil {
					report.ErrorSQLServer(nil, err, query, params...)
					return a, err
				}
				rand = IDGame % CountNations
				var NationShort string
				query = `SELECT short FROM players.nation where continent = $1 ORDER BY id ASC OFFSET $2 limit 1`
				params = []any{Continent, rand}
				err = db.QueryRow(query, params...).Scan(&NationShort)
				if err != nil {
					report.ErrorSQLServer(nil, err, query, params...)
					return a, err
				}
				a.Nation = NationShort
				a.NationColor = RED
			case YELLOW:
				len := len(constants.NationMatches[a.Nation])
				if len == 0 {
					a.Nation = Answer.NationShort
					a.NationColor = GREEN
				} else {
					rand := IDGame % len
					Nation := constants.NationMatches[a.Nation][rand]
					var NationShort string
					query = `SELECT short FROM players.nation WHERE country = $1`
					params = []any{Nation}
					err = db.QueryRow(query, params...).Scan(&NationShort)
					if err != nil {
						report.ErrorSQLServer(nil, err, query, params...)
						return a, err
					}
					a.Nation = NationShort
					a.NationColor = YELLOW
				}
			case GREEN:
				a.Nation = Answer.NationShort
				a.NationColor = GREEN
			default:
				err = errors.New(`error in getting hint color`)
				return a, err
			}
		case POSITION:
			switch Hint.Color {
			case YELLOW:
				len := len(Matches[Answer.Position])
				rand := IDGame % len
				NewPosition := Matches[Answer.Position][rand]
				a.Position = NewPosition
				a.PositionColor = YELLOW
			case GREEN:
				a.Position = Answer.Position
				a.PositionColor = GREEN
			default:
				err = errors.New(`error in getting hint color`)
				return a, err
			}
		case PRICE:
			switch Hint.Color {
			case RED:
				rand := IDGame % 20
				rand2 := (IDGame + (IDGame % 45)) % 2
				var NewPrice int
				if rand2 == 0 {
					NewPrice = Answer.Price - 50000000 - rand*1000000
					if NewPrice < 0 {
						NewPrice = Answer.Price + 50000000 + rand*1000000
					}
				} else {
					NewPrice = Answer.Price + 50000000 + rand*1000000
					if NewPrice > 150000000 {
						NewPrice = Answer.Price - 50000000 - rand*1000000
					}
				}
				a.Price = NewPrice
				a.PriceColor = RED
			case YELLOW:
				rand := IDGame%5 + 5
				rand2 := IDGame % 2
				var NewPrice int
				if rand2 == 0 {
					NewPrice = Answer.Price - rand*1000000
					if NewPrice < 0 {
						NewPrice = Answer.Price + rand*1000000
					}
				} else {
					NewPrice = Answer.Price + rand*1000000
				}
				a.Price = NewPrice
				a.PriceColor = YELLOW
			case GREEN:
				a.Price = Answer.Price
				a.PriceColor = GREEN
			default:
				err = errors.New(`error in getting hint color`)
				return a, err
			}
		default:
			err = errors.New(`error in getting hint type`)
			return a, err
		}
	}
	return a, nil
}
*/
