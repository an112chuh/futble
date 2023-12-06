package repository

import (
	"futble/aggregate"
	"futble/entity"
)

type GameRepository interface {
	New(Type int, user entity.User) (*aggregate.Game, error)
}
