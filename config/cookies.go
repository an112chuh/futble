package config

import (
	"futble/constants"

	"github.com/gorilla/sessions"
)

var Store *sessions.CookieStore

func InitCookies() {
	constants.SetCookies()
	Store = sessions.NewCookieStore(
		constants.Cookies.AuthKeyOne,
		constants.Cookies.EncryptionOne,
	)
	Store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   60 * 300000000,
		HttpOnly: true,
	}
	Store.MaxAge(60 * 300000000)
}
