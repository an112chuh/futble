package game

import (
	"bytes"
	"encoding/json"
	"futble/config"
	"futble/report"
	"futble/result"
	"hash/fnv"
	"io"
	"net/http"

	"github.com/gorilla/sessions"
)

type AccountData struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	IsLogged bool
}

func TestHandler(w http.ResponseWriter, r *http.Request) {
	IsLogin(w, r)

}

func RegHandler(w http.ResponseWriter, r *http.Request) {
	session, err := config.Store.Get(r, "cookie-name")
	if err != nil {
		report.ErrorServer(r, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var data AccountData
	b, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(b))
	err = json.Unmarshal(b, &data)
	if err != nil {
		report.ErrorServer(r, err)
	}
	res, user := Reg(r, data)
	if res.Done {
		session.Values["user"] = user
		err = session.Save(r, w)
		if err != nil {
			report.ErrorServer(r, err)
			res = result.SetErrorResult(`Внутренняя ошибка`)
			return
		}
	} else {
		result.ReturnJSON(w, &res)
		return
	}
	result.ReturnJSON(w, &res)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	session, err := config.Store.Get(r, "cookie-name")
	if err != nil {
		report.ErrorServer(r, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var data AccountData
	b, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(b))
	err = json.Unmarshal(b, &data)
	if err != nil {
		report.ErrorServer(r, err)
	}
	res, ID, Rights := Login(r, data)
	if res.Done {
		user := &config.User{
			Username:      data.Login,
			ID:            ID,
			Rights:        Rights,
			Authenticated: true,
		}
		session.Values["user"] = user
		err = session.Save(r, w)
		if err != nil {
			report.ErrorServer(r, err)
			res = result.SetErrorResult(`Внутренняя ошибка`)
		}
	}
	result.ReturnJSON(w, &res)
}

func Reg(r *http.Request, data AccountData) (res result.ResultInfo, user config.User) {
	db := config.ConnectDB()
	var LoginExist bool
	query := `SELECT EXISTS(SELECT 1 FROM users.accounts WHERE login = $1)`
	err := db.QueryRow(query, data.Login).Scan(&LoginExist)
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(`Внутренняя ошибка`)
		return
	}
	if LoginExist {
		res = result.SetErrorResult(`Пользователь с таким логином уже существует`)
		return
	}
	Hash := HashCreation(data.Password)
	var ID int
	query = `INSERT INTO users.accounts (login, not_logged, password) VALUES ($1, $2, $3) RETURNING id`
	params := []any{data.Login, data.IsLogged, Hash}
	err = db.QueryRow(query, params...).Scan(&ID)
	if err != nil {
		report.ErrorServer(r, err)
		return
	}
	user.ID = ID
	user.Authenticated = true
	user.Rights = 1
	user.Username = data.Login
	res.Done = true
	return
}

func getUser(s *sessions.Session) config.User {
	val := s.Values["user"]
	var user = config.User{}
	user, ok := val.(config.User)
	if !ok {
		return config.User{Authenticated: false}
	}
	return user
}

func IsLogin(w http.ResponseWriter, r *http.Request) (user config.User) {
	session, err := config.Store.Get(r, "cookie-name")
	if err != nil {
		report.ErrorServer(r, err)
		return
	}
	user = getUser(session)
	if auth := user.Authenticated; !auth {
		var data AccountData
		data.IsLogged = true
		data.Login = `newman`
		var res result.ResultInfo
		res, user = Reg(r, data)
		if res.Done {
			session.Values["user"] = user
			err = session.Save(r, w)
			if err != nil {
				report.ErrorServer(r, err)
				res = result.SetErrorResult(`Внутренняя ошибка`)
				return
			}
		} else {
			result.ReturnJSON(w, &res)
			return
		}
		result.ReturnJSON(w, &res)
	}
	return user
}

func HashCreation(password string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(password))
	return h.Sum32()
}

func Login(r *http.Request, data AccountData) (res result.ResultInfo, ID int, Rights config.Rights) {
	db := config.ConnectDB()
	res = FindLogin(r, data)
	if res.Done {
		res, ID = CheckPassword(r, data)
		if res.Done {
			query := `SELECT rights from users.accounts where id = $1`
			err := db.QueryRow(query, ID).Scan(&Rights)
			if err != nil {
				report.ErrorServer(r, err)
				res = result.SetErrorResult(`Внутренняя ошибка`)
				return
			}
		}
	}
	return res, ID, Rights
}

func FindLogin(r *http.Request, data AccountData) (res result.ResultInfo) {
	db := config.ConnectDB()
	var LoginExist bool
	query := `SELECT EXISTS (SELECT 1 FROM users.accounts WHERE login = $1)`
	err := db.QueryRow(query, data.Login).Scan(&LoginExist)
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(`Внутренняя ошибка`)
		return
	}
	if !LoginExist {
		res = result.SetErrorResult(`Неверные логин или пароль`)
		return
	}
	res.Done = true
	return res
}

func CheckPassword(r *http.Request, data AccountData) (res result.ResultInfo, ID int) {
	db := config.ConnectDB()
	Hash := HashCreation(data.Password)
	var HashFromDB uint32
	query := "SELECT password, id FROM users.accounts WHERE login = $1"
	err := db.QueryRow(query, data.Login).Scan(&HashFromDB, &ID)
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(`Внутренняя ошибка`)
		return
	}
	if Hash == HashFromDB {
		res.Done = true
		res.Items = ID
	} else {
		res = result.SetErrorResult(`Неверные логин или пароль`)
		return
	}
	return res, ID
}
