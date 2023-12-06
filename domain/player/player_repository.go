package repository

import "futble/entity"

type PlayerRepository interface {
	GetAll() ([]entity.Player, error)
}
