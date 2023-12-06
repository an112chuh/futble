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
	"strconv"
	"time"

	"github.com/gorilla/sessions"
)

type AccountData struct {
	Login     string  `json:"login"`
	Password  string  `json:"password"`
	Mail      *string `json:"mail,omitempty"`
	Promocode *string `json:"promocode,omitempty"`
	IsLogged  bool
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
	var res result.ResultInfo
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(`Внутренняя ошибка`)
		result.ReturnJSON(w, &res)
		return
	}
	user := IsLogin(w, r)
	res, user = Reg(r, data, user.ID)
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

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, err := config.Store.Get(r, "cookie-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	user := getUser(session)
	if user.Rights != config.NotLogged {
		session.Values["user"] = config.User{}
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

func Reg(r *http.Request, data AccountData, IDUser int) (res result.ResultInfo, user config.User) {
	db := config.ConnectDB()
	if data.Login == `newman` {
		Count := 0
		query := `SELECT COUNT(*) FROM users.accounts`
		err := db.QueryRow(query).Scan(&Count)
		if err != nil {
			report.ErrorServer(r, err)
			res = result.SetErrorResult(`Внутренняя ошибка`)
			return
		}
		data.Login += strconv.Itoa(Count + 1)
		user.Rights = 0
	} else {
		user.Rights = 1
	}
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
	if data.IsLogged {
		query = `INSERT INTO users.accounts (login, not_logged, password, rights) VALUES ($1, $2, $3, $4) RETURNING id`
		params := []any{data.Login, data.IsLogged, Hash, user.Rights}
		err = db.QueryRow(query, params...).Scan(&ID)
		if err != nil {
			report.ErrorServer(r, err)
			return
		}
		user.ID = ID
	} else {
		var exists bool
		query = `SELECT EXISTS(SELECT 1 FROM users.accounts WHERE id = $1 AND login not like 'newman%')`
		params := []any{IDUser}
		err = db.QueryRow(query, params...).Scan(&exists)
		if err != nil {
			report.ErrorSQLServer(r, err, query, params...)
			res = result.SetErrorResult(`Server Error`)
			return
		}
		if exists {
			res = result.SetErrorResult(`You are already in account`)
			return
		}
		query = `UPDATE users.accounts SET login = $1, not_logged = false, password = $2, rights = $3, mail = $4, promocode = $5 WHERE id = $6`
		params = []any{data.Login, Hash, user.Rights, data.Mail, data.Promocode, IDUser}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(r, err, query, params...)
			return
		}
		query = `INSERT INTO users.trophies (id_user, trophies, updated_at) VALUES ($1, 0, $2)`
		params = []any{IDUser, time.Now()}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(r, err, query, params...)
			return
		}
		user.ID = IDUser
	}
	user.Authenticated = true
	user.Username = data.Login
	res.Done = true
	SetOnline(user)
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
	//	fmt.Printf("%+v\n", session)
	//	fmt.Printf("%+v\n", session.Options)
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
		res, user = Reg(r, data, 0)
		if res.Done {
			session.Values["user"] = user
			err = session.Save(r, w)
			if err != nil {
				report.ErrorServer(r, err)
				res = result.SetErrorResult(`Внутренняя ошибка`)
				return
			}
		} else {
			//	result.ReturnJSON(w, &res)
			return
		}
		//result.ReturnJSON(w, &res)
	}
	SetOnline(user)
	return user
}

func SetOnline(user config.User) {
	db := config.ConnectDB()
	t := time.Now()
	query := `UPDATE users.accounts SET online = $1 WHERE id = $2`
	params := []interface{}{t, user.ID}
	_, err := db.Exec(query, params)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return
	}
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
