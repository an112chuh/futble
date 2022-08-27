package game

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"futble/config"
	"futble/report"
	"futble/result"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	gomail "gopkg.in/mail.v2"
)

type AnswerStruct struct {
	Body string `json:"answer"`
}

type QuestionStruct struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Theme string `json:"theme"`
	Body  string `json:"body"`
}

type RequestStruct struct {
	Answered *bool                  `json:"answered,omitempty"`
	Records  []RequestRecordsStruct `json:"records"`
}

type RequestRecordsStruct struct {
	Theme string  `json:"theme"`
	Time  string  `json:"time"`
	Body  string  `json:"body"`
	IsNew *bool   `json:"is_new,omitempty"`
	Login *string `json:"login,omitempty"`
	Name  *string `json:"name,omitempty"`
}

func MessagesAnswerHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	if user.Rights == config.NotLogged {
		res = result.SetErrorResult(NOT_LOGGED_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	vars := mux.Vars(r)
	IDRequestString := vars["id"]
	IDRequest, err := strconv.Atoi(IDRequestString)
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(CONVERT_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	var data AnswerStruct
	b, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(b))
	err = json.Unmarshal(b, &data)
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(JSON_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	res = MessageAnswer(user.ID, IDRequest, data)
	result.ReturnJSON(w, &res)
}

func MessagesCreateHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	var data QuestionStruct
	b, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(b))
	err := json.Unmarshal(b, &data)
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(JSON_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	res = MessageCreate(user, data)
	result.ReturnJSON(w, &res)
}

func MessagesItemHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	if user.Rights == config.NotLogged {
		res = result.SetErrorResult(NOT_LOGGED_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	vars := mux.Vars(r)
	IDRequestString := vars["id"]
	IDRequest, err := strconv.Atoi(IDRequestString)
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(CONVERT_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	res = MessageRequest(IDRequest, user.ID)
	result.ReturnJSON(w, &res)
}

func MessagesListHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	if user.Rights == config.NotLogged {
		res = result.SetErrorResult(NOT_LOGGED_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	res = MessageList(user.ID)
	result.ReturnJSON(w, &res)
}

func MessagesCloseRequestHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	if user.Rights == config.NotLogged {
		res = result.SetErrorResult(NOT_LOGGED_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	vars := mux.Vars(r)
	IDRequestString := vars["id"]
	IDRequest, err := strconv.Atoi(IDRequestString)
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(CONVERT_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	res = MessageClose(IDRequest, user.ID)
	result.ReturnJSON(w, &res)
}

func AdminMessagesAnswerHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	if user.Rights != config.Admin {
		res = result.SetErrorResult(FORBIDDEN_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	vars := mux.Vars(r)
	IDRequestString := vars["id"]
	IDRequest, err := strconv.Atoi(IDRequestString)
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(CONVERT_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	var data AnswerStruct
	b, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(b))
	err = json.Unmarshal(b, &data)
	if err != nil {
		report.ErrorServer(r, err)
		res = result.SetErrorResult(JSON_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	res = AdminMessageAnswer(IDRequest, user.ID, data)
	result.ReturnJSON(w, &res)
}

func AdminMessagesListHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	user := IsLogin(w, r)
	if user.Rights != config.Admin {
		res = result.SetErrorResult(FORBIDDEN_ERROR)
		result.ReturnJSON(w, &res)
		return
	}
	res = AdminMessageList()
	result.ReturnJSON(w, &res)
}

func MessageAnswer(IDUser int, IDRequest int, data AnswerStruct) (res result.ResultInfo) {
	db := config.ConnectDB()
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM messages.requests WHERE id = $1 AND answered = TRUE AND closed = FALSE)`
	params := []any{IDRequest}
	err := db.QueryRow(query, params...).Scan(&exists)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	if !exists {
		res = result.SetErrorResult(`This request isn't open`)
		return
	}
	query = `SELECT EXISTS(SELECT 1 FROM messages.requests WHERE id = $1 AND id_user = $2)`
	params = []any{IDRequest, IDUser}
	err = db.QueryRow(query, params...).Scan(&exists)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	if !exists {
		res = result.SetErrorResult(`This request doesn't belong to user`)
		return
	}
	var IDMessage int
	query = `INSERT INTO messages.list (id_request, id_user, is_answer, created_at) VALUES ($1, $2, false, $3) RETURNING id`
	params = []any{IDRequest, IDUser, time.Now()}
	err = db.QueryRow(query, params...).Scan(&IDMessage)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	query = `UPDATE messages.requests SET answered = FALSE WHERE id = $1`
	params = []any{IDRequest}
	_, err = db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	file, err := os.Create("messages/" + strconv.Itoa(IDRequest) + "/" + strconv.Itoa(IDMessage) + ".txt")
	if err != nil {
		res = result.SetErrorResult(UNKNOWN_ERROR + "messages")
		report.ErrorServer(nil, err)
		return
	}
	defer file.Close()
	fmt.Fprint(file, data.Body)
	res.Done = true
	return
}

func MessageCreate(user config.User, data QuestionStruct) (res result.ResultInfo) {
	db := config.ConnectDB()
	var IsLogged bool = true
	if user.Rights == config.NotLogged {
		IsLogged = false
	}
	var IDRequest int
	query := `INSERT INTO messages.requests (id_user, name, mail, theme, answered, closed, created_at, is_logged, is_new) VALUES 
		($1, $2, $3, $4, FALSE, FALSE, $5, $6, FALSE) RETURNING id`
	Mail := data.Email
	if IsLogged {
		Mail = user.Mail
	}
	params := []any{user.ID, data.Name, Mail, data.Theme, time.Now(), IsLogged}
	err := db.QueryRow(query, params...).Scan(&IDRequest)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	var IDMessage int
	query = `INSERT INTO messages.list (id_request, id_user, is_answer, created_at) VALUES ($1, $2, FALSE, $3) RETURNING id`
	params = []any{IDRequest, user.ID, time.Now()}
	err = db.QueryRow(query, params...).Scan(&IDMessage)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	err = os.Mkdir("messages/"+strconv.Itoa(IDRequest)+"/", 0777)
	if err != nil {
		res = result.SetErrorResult(UNKNOWN_ERROR + "messages")
		report.ErrorServer(nil, err)
		return
	}
	file, err := os.Create("messages/" + strconv.Itoa(IDRequest) + "/" + strconv.Itoa(IDMessage) + ".txt")
	if err != nil {
		res = result.SetErrorResult(UNKNOWN_ERROR + "messages")
		report.ErrorServer(nil, err)
		return
	}
	defer file.Close()
	fmt.Fprint(file, data.Body)
	res.Done = true
	res.Items = map[string]any{"id": IDRequest}
	return
}

func MessageRequest(IDRequest int, IDUser int) (res result.ResultInfo) {
	db := config.ConnectDB()
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM messages.requests WHERE id = $1 AND id_user = $2)`
	params := []any{IDRequest, IDUser}
	err := db.QueryRow(query, params...).Scan(&exists)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	if !exists {
		res = result.SetErrorResult(`This request doesn't belong to user`)
		return
	}
	type Message struct {
		ID        int
		IsAnswer  bool
		Theme     string
		CreatedAt time.Time
	}
	var r RequestStruct
	var mes []Message
	query = `SELECT messages.list.id, is_answer, theme, messages.list.created_at FROM messages.list
		INNER JOIN messages.requests ON messages.requests.id = id_request
		WHERE id_request = $1 ORDER BY messages.list.id ASC`
	params = []any{IDRequest}
	rows, err := db.Query(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var m Message
		err = rows.Scan(&m.ID, &m.IsAnswer, &m.Theme, &m.CreatedAt)
		if err != nil {
			report.ErrorServer(nil, err)
			res = result.SetErrorResult(DATABASE_ERROR)
			return
		}
		mes = append(mes, m)
	}
	var IsAnswered = false
	for i := 0; i < len(mes); i++ {
		var req RequestRecordsStruct
		req.Time = mes[i].CreatedAt.Format("02.01.2006 15:04")
		if mes[i].IsAnswer {
			req.Theme = "Footble team’s answer:"
		} else {
			req.Theme = mes[i].Theme
		}
		IsAnswered = mes[i].IsAnswer
		file, err := os.Open("messages/" + strconv.Itoa(IDRequest) + "/" + strconv.Itoa(mes[i].ID) + ".txt")
		if err != nil {
			report.ErrorServer(nil, err)
			res = result.SetErrorResult(UNKNOWN_ERROR + "messages")
			return
		}
		defer file.Close()
		b, err := ioutil.ReadAll(file)
		if err != nil {
			report.ErrorServer(nil, err)
			res = result.SetErrorResult(UNKNOWN_ERROR + "messages")
			return
		}
		req.Body = string(b)
		r.Records = append(r.Records, req)
	}
	r.Answered = new(bool)
	*r.Answered = false
	if IsAnswered {
		*r.Answered = true
	}
	res.Done = true
	res.Items = r
	return
}

func MessageList(IDUser int) (res result.ResultInfo) {
	db := config.ConnectDB()
	type Message struct {
		IDReq     int
		Theme     string
		IsNew     bool
		IDMessage int
		CreatedAt time.Time
	}
	var mes []Message
	query := `SELECT m.id AS id_request, m.theme, m.is_new, messages.list.id AS id_message, messages.list.created_at FROM messages.list 
	INNER JOIN(
		SELECT id, theme, is_new FROM messages.requests WHERE id_user = $1
	) m ON m.id = messages.list.id_request 
	ORDER BY m.id DESC`
	params := []any{IDUser}
	rows, err := db.Query(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var m Message
		err = rows.Scan(&m.IDReq, &m.Theme, &m.IsNew, &m.IDMessage, &m.CreatedAt)
		if err != nil {
			report.ErrorServer(nil, err)
			res = result.SetErrorResult(UNKNOWN_ERROR + "message")
			return
		}
		mes = append(mes, m)
	}
	var r RequestStruct
	for i := 0; i < len(mes); i++ {
		var req RequestRecordsStruct
		req.IsNew = new(bool)
		req.IsNew = &mes[i].IsNew
		req.Theme = mes[i].Theme
		req.Time = mes[i].CreatedAt.Format("02.01.2006 15:04")
		file, err := os.Open("messages/" + strconv.Itoa(mes[i].IDReq) + "/" + strconv.Itoa(mes[i].IDMessage) + ".txt")
		if err != nil {
			report.ErrorServer(nil, err)
			res = result.SetErrorResult(UNKNOWN_ERROR + "messages")
			return
		}
		defer file.Close()
		b, err := ioutil.ReadAll(file)
		if err != nil {
			report.ErrorServer(nil, err)
			res = result.SetErrorResult(UNKNOWN_ERROR + "messages")
			return
		}
		req.Body = string(b)
		r.Records = append(r.Records, req)
	}
	res.Done = true
	res.Items = r
	return
}

func MessageClose(IDRequest int, IDUser int) (res result.ResultInfo) {
	db := config.ConnectDB()
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM messages.requests WHERE id = $1 AND id_user = $2)`
	params := []any{IDRequest, IDUser}
	err := db.QueryRow(query, params...).Scan(&exists)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	if !exists {
		res = result.SetErrorResult(`This request doesn't belong to user`)
		return
	}
	query = `SELECT EXISTS(SELECT 1 FROM messages.requests WHERE id = $1 AND answered = FALSE AND closed = FALSE)`
	params = []any{IDRequest}
	err = db.QueryRow(query, params...).Scan(&exists)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)

	}
	if !exists {
		res = result.SetErrorResult(`This request isn't closed`)
		return
	}
	query = `UPDATE messages.requests SET closed = TRUE, closed_at = $1 WHERE id = $2`
	params = []any{time.Now(), IDRequest}
	_, err = db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	res.Done = true
	res.Items = map[string]any{"id": IDRequest}
	return
}

func AdminMessageList() (res result.ResultInfo) {
	db := config.ConnectDB()
	type Message struct {
		IDReq     int
		Theme     string
		IsNew     bool
		Name      string
		IDMessage int
		CreatedAt time.Time
		Login     string
	}
	var mes []Message
	query := `SELECT m.id AS id_request, m.theme, m.is_new, m.name, messages.list.id AS id_message, messages.list.created_at, users.accounts.login FROM messages.list 
	INNER JOIN(
		SELECT id, theme, is_new, name FROM messages.requests WHERE answered = false
	) m ON m.id = messages.list.id_request 
	INNER JOIN users.accounts ON users.accounts.id = messages.list.id_user  
	ORDER BY m.id DESC`
	rows, err := db.Query(query)
	if err != nil {
		report.ErrorSQLServer(nil, err, query)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var m Message
		err = rows.Scan(&m.IDReq, &m.Theme, &m.IsNew, &m.Name, &m.IDMessage, &m.CreatedAt, &m.Login)
		if err != nil {
			report.ErrorServer(nil, err)
			res = result.SetErrorResult(UNKNOWN_ERROR + "message")
			return
		}
		mes = append(mes, m)
	}
	var r RequestStruct
	for i := 0; i < len(mes); i++ {
		var req RequestRecordsStruct
		req.Theme = mes[i].Theme
		req.Time = mes[i].CreatedAt.Format("02.01.2006 15:04")
		req.Login = new(string)
		req.Login = &mes[i].Login
		req.Name = new(string)
		req.Name = &mes[i].Name
		file, err := os.Open("messages/" + strconv.Itoa(mes[i].IDReq) + "/" + strconv.Itoa(mes[i].IDMessage) + ".txt")
		if err != nil {
			report.ErrorServer(nil, err)
			res = result.SetErrorResult(UNKNOWN_ERROR + "messages")
			return
		}
		defer file.Close()
		b, err := ioutil.ReadAll(file)
		if err != nil {
			report.ErrorServer(nil, err)
			res = result.SetErrorResult(UNKNOWN_ERROR + "messages")
			return
		}
		req.Body = string(b)
		r.Records = append(r.Records, req)
	}
	res.Done = true
	res.Items = r
	return
}

func AdminMessageAnswer(IDRequest int, IDUser int, data AnswerStruct) (res result.ResultInfo) {
	db := config.ConnectDB()
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM messages.requests WHERE id = $1 AND answered = FALSE and closed = FALSE)`
	params := []any{IDRequest}
	err := db.QueryRow(query, params...).Scan(&exists)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	if !exists {
		res = result.SetErrorResult("request doesn't exist")
		return
	}
	var MailRequest, MailAccount *string
	var IsLogged bool
	query = `SELECT messages.requests.mail AS mail1, users.accounts.mail AS mail2, is_logged FROM users.accounts 
		INNER JOIN messages.requests ON id_user = users.accounts.id
		WHERE messages.requests.id = $1`
	err = db.QueryRow(query, params...).Scan(&MailRequest, &MailAccount, &IsLogged)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	var MailToSend string
	if IsLogged {
		if MailAccount != nil {
			MailToSend = *MailAccount
		}
		query = `UPDATE messages.requests SET answered = TRUE WHERE id = $1`
		params = []any{IDRequest}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			res = result.SetErrorResult(DATABASE_ERROR)
			return
		}
	} else {
		if MailRequest != nil {
			MailToSend = *MailRequest
		}
		query = `UPDATE messages.requests SET answered = TRUE AND closed = TRUE WHERE id = $1`
		params = []any{IDRequest}
		_, err = db.Exec(query, params...)
		if err != nil {
			report.ErrorSQLServer(nil, err, query, params...)
			res = result.SetErrorResult(DATABASE_ERROR)
			return
		}
	}
	var IDMessage int
	query = `INSERT INTO messages.list (id_request, id_user, is_answer, created_at) VALUES ($1, $2, TRUE, $3) RETURNING id`
	params = []any{IDRequest, IDUser, time.Now()}
	err = db.QueryRow(query, params...).Scan(&IDMessage)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		res = result.SetErrorResult(DATABASE_ERROR)
		return
	}
	file, err := os.Create("messages/" + strconv.Itoa(IDRequest) + "/" + strconv.Itoa(IDMessage) + ".txt")
	if err != nil {
		res = result.SetErrorResult(UNKNOWN_ERROR + "messages")
		report.ErrorServer(nil, err)
		return
	}
	defer file.Close()
	fmt.Fprint(file, data.Body)
	m := gomail.NewMessage()
	fmt.Printf("MailToSend - %s", MailToSend)
	m.SetHeader("From", "a.chuhnov@yandex.ru")
	m.SetHeader("To", MailToSend)
	m.SetHeader("Subject", "Answer to your request in support")
	m.SetBody("text/plain", data.Body)
	d := gomail.NewDialer("smtp.yandex.ru", 587, "a.chuhnov@yandex.ru", "password")
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	if err = d.DialAndSend(m); err != nil {
		report.ErrorServer(nil, err)
		res = result.SetErrorResult(`Error in sending message`)
	}
	res.Done = true
	return
}
