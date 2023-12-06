package repository

import "futble/entity"

type GuessRepository interface {
	GetGuessesByGame(IDGame int) ([]entity.Guess, error)
	GetAnswer(IDGame int) (*entity.Guess, error)
	CheckGuess(IDPlayer int, IDAnswer int) (entity.Guess, error)
}
