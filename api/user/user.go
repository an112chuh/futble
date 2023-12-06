package api

import (
	"bytes"
	"encoding/json"
	"futble/config"
	"futble/entity"
	infrastructure "futble/infrastructure/user/repository"
	"futble/report"
	"futble/result"
	"io"
	"net/http"
)

func TestHandler(w http.ResponseWriter, r *http.Request) {
	userRep := infrastructure.NewUserRepository()
	userRep.IsLogin(w, r)
}

func RegHandler(w http.ResponseWriter, r *http.Request) {
	userRep := infrastructure.NewUserRepository()
	session, err := config.Store.Get(r, "cookie-name")
	if err != nil {
		report.ErrorServer(r, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var data entity.User
	b, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(b))
	err = json.Unmarshal(b, &data)
	var res result.ResultInfo
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(`Внутренняя ошибка`)
		result.ReturnJSON(w, &res)
		return
	}
	user := userRep.IsLogin(w, r)
	user, err = userRep.Reg(data, user.ID)
	if err != nil {
		res = result.SetErrorResult(err.Error())
		result.ReturnJSON(w, &res)
		return
	}
	session.Values["user"] = user
	err = session.Save(r, w)
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(`Внутренняя ошибка`)
		return
	}
	res.Done = true
	result.ReturnJSON(w, &res)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	userRep := infrastructure.NewUserRepository()
	session, err := config.Store.Get(r, "cookie-name")
	if err != nil {
		report.ErrorServer(r, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var data entity.User
	b, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(b))
	err = json.Unmarshal(b, &data)
	if err != nil {
		report.ErrorServer(r, err)
	}
	var res result.ResultInfo
	user, err := userRep.Login(data)
	if err != nil {
		res = result.SetErrorResult(err.Error())
		result.ReturnJSON(w, &res)
		return
	}
	session.Values["user"] = user
	err = session.Save(r, w)
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(`Внутренняя ошибка`)
	}
	result.ReturnJSON(w, &res)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	userRep := infrastructure.NewUserRepository()
	session, err := config.Store.Get(r, "cookie-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	user := userRep.Get(session)
	if user.Rights != entity.NotLogged {
		session.Values["user"] = entity.User{}
		session.Options.MaxAge = -1
		err = session.Save(r, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var res result.ResultInfo
		res.Done = true
		result.ReturnJSON(w, &res)
	} else {
		result.SetErrorResult("Can't logout unlogged user")
		return
	}
}
