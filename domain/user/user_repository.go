package repository

import (
	"futble/entity"
	"net/http"

	"github.com/gorilla/sessions"
)

type UserRepository interface {
	Get(*sessions.Session) *entity.User
	SetOnline(user entity.User) error
	CheckPassword(data entity.User) (ID int, err error)
	FindLogin(data entity.User) (bool, error)
	Login(data entity.User) (User *entity.User, err error)
	Reg(data entity.User, IDUser int) (user *entity.User, err error)
	IsLogin(w http.ResponseWriter, req *http.Request) (user *entity.User)
}
