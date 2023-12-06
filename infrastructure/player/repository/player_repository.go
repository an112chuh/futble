package infrastructure

import (
	"futble/config"
	repository "futble/domain/player"
	"futble/entity"
	"futble/report"
)

type PlayerRepository struct {
}

func NewPlayerRepository() repository.PlayerRepository {
	return &PlayerRepository{}
}

func (this *PlayerRepository) GetAll() ([]entity.Player, error) {
	var res []entity.Player
	db := config.ConnectDB()
	query := `SELECT id, name, surname FROM players.data ORDER BY id ASC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p entity.Player
		var name, surname string
		err = rows.Scan(&p.ID, &name, &surname)
		if err != nil {
			report.ErrorServer(nil, err)
			return nil, err
		}
		if name == `` {
			p.Name = surname
		} else {
			p.Name = name + ` ` + surname
		}
		res = append(res, p)
	}
	return res, nil
}
