package check

import (
	"fmt"
	"futble/config"
	"futble/report"
)

type PairIDNation struct {
	ID     int
	Nation string
}

var IDs []int

func CheckNationsExist(Nations map[string][]string) {
	db := config.ConnectDB()
	query := `SELECT id, nation FROM players.data`
	rows, err := db.Query(query)
	if err != nil {
		report.ErrorSQLServer(nil, err, query)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var p PairIDNation
		err = rows.Scan(&p.ID, &p.Nation)
		if err != nil {
			report.ErrorServer(nil, err)
			return
		}
		_, ok := Nations[p.Nation]
		if !ok {
			fmt.Printf("ID - %d, COUNRTY BORDER FAIL\n", p.ID)
		}
	}
}

func CheckNationCorrect() {
	db := config.ConnectDB()
	var CountPlayers int
	query := `SELECT id FROM players.data ORDER BY id DESC LIMIT 1`
	err := db.QueryRow(query).Scan(&CountPlayers)
	if err != nil {
		report.ErrorServer(nil, err)
		return
	}
	for i := 1; i <= CountPlayers; i++ {
		var exists bool
		query := `SELECT exists (SELECT short from players.nation inner join players.data on players.data.nation = players.nation.country where players.data.id = $1)`
		params := []any{i}
		err := db.QueryRow(query, params...).Scan(&exists)
		if err != nil {
			report.ErrorServer(nil, err)
			return
		}
		if !exists {
			var exist bool
			query := `SELECT EXISTS(SELECT 1 FROM players.data WHERE id = $1)`
			err = db.QueryRow(query, params...).Scan(&exist)
			if err != nil {
				report.ErrorServer(nil, err)
				return
			}
			if exist {
				fmt.Printf("ID - %d, COUNTRY SHORT FAIL\n", i)
			}
		}
	}
}

func CheckClubCorrect() {
	db := config.ConnectDB()
	var CountPlayers int
	query := `SELECT id FROM players.data ORDER BY id DESC LIMIT 1`
	err := db.QueryRow(query).Scan(&CountPlayers)
	if err != nil {
		report.ErrorServer(nil, err)
		return
	}
	for i := 1; i <= CountPlayers; i++ {
		var exists bool
		query := `SELECT exists (SELECT short from players.club inner join players.data on players.data.club = players.club.club where players.data.id = $1)`
		params := []any{i}
		err := db.QueryRow(query, params...).Scan(&exists)
		if err != nil {
			report.ErrorServer(nil, err)
			return
		}
		if !exists {
			var exist bool
			query := `SELECT EXISTS(SELECT 1 FROM players.data WHERE id = $1)`
			err = db.QueryRow(query, params...).Scan(&exist)
			if err != nil {
				report.ErrorServer(nil, err)
				return
			}
			if exist {
				fmt.Printf("ID - %d, CLUB SHORT FAIL\n", i)
			}
		}
	}
}

func CheckLeagueCorrect() {
	db := config.ConnectDB()
	var CountPlayers int
	query := `SELECT id FROM players.data ORDER BY id DESC LIMIT 1`
	err := db.QueryRow(query).Scan(&CountPlayers)
	if err != nil {
		report.ErrorServer(nil, err)
		return
	}
	for i := 1; i <= CountPlayers; i++ {
		var exists bool
		query := `SELECT exists (SELECT place from players.league inner join players.data on players.data.league = players.league.league where players.data.id = $1)`
		params := []any{i}
		err := db.QueryRow(query, params...).Scan(&exists)
		if err != nil {
			report.ErrorServer(nil, err)
			return
		}
		if !exists {
			var exist bool
			query := `SELECT EXISTS(SELECT 1 FROM players.data WHERE id = $1)`
			err = db.QueryRow(query, params...).Scan(&exist)
			if err != nil {
				report.ErrorServer(nil, err)
				return
			}
			if exist {
				fmt.Printf("ID - %d, LEAGUE FAIL\n", i)
			}
		}
	}
}

func DownloadIDs() {
	db := config.ConnectDB()
	query := `SELECT id FROM players.data`
	rows, err := db.Query(query)
	if err != nil {
		report.ErrorSQLServer(nil, err, query)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		err = rows.Scan(&id)
		if err != nil {
			report.ErrorServer(nil, err)
			return
		}
		IDs = append(IDs, id)
	}
	//	fmt.Println(IDs)
}
