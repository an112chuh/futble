package infrastructure

import (
	"errors"
	"futble/config"
	repository "futble/domain/user"
	"futble/entity"
	"futble/report"
	"hash/fnv"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/sessions"
)

type UserRepository struct {
}

func NewUserRepository() repository.UserRepository {
	return &UserRepository{}
}

func (r *UserRepository) Get(s *sessions.Session) *entity.User {
	val := s.Values["user"]
	var user = entity.User{}
	user, ok := val.(entity.User)
	if !ok {
		return &entity.User{Authenticated: false}
	}
	return &user
}

func (r *UserRepository) SetOnline(user entity.User) error {
	db := config.ConnectDB()
	t := time.Now()
	query := `UPDATE users.accounts SET online = $1 WHERE id = $2`
	params := []interface{}{t, user.ID}
	_, err := db.Exec(query, params)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
	}
	return err
}

func HashCreation(password string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(password))
	return h.Sum32()
}

func (r *UserRepository) CheckPassword(data entity.User) (ID int, err error) {
	db := config.ConnectDB()
	Hash := HashCreation(data.Password)
	var HashFromDB uint32
	query := "SELECT password, id FROM users.accounts WHERE login = $1"
	err = db.QueryRow(query, data.Login).Scan(&HashFromDB, &ID)
	if err != nil {
		report.ErrorServer(nil, err)
		return -1, err
	}
	if Hash != HashFromDB {
		return -1, errors.New(`Неверный логин или пароль`)
	}
	return
}

func (r *UserRepository) FindLogin(data entity.User) (bool, error) {
	db := config.ConnectDB()
	var LoginExist bool
	query := `SELECT EXISTS (SELECT 1 FROM users.accounts WHERE login = $1)`
	err := db.QueryRow(query, data.Login).Scan(&LoginExist)
	if err != nil {
		report.ErrorServer(nil, err)
		return false, err
	}
	if !LoginExist {
		return false, errors.New(`Неверный логин или пароль`)
	}
	return true, nil
}

func (r *UserRepository) Login(data entity.User) (User *entity.User, err error) {
	db := config.ConnectDB()
	isFound, err := r.FindLogin(data)
	if err != nil {
		return &entity.User{}, err
	}
	if isFound {
		IDUser, err := r.CheckPassword(data)
		if err != nil {
			return &entity.User{}, err
		}
		var Rights int
		query := `SELECT rights from users.accounts where id = $1`
		err = db.QueryRow(query, IDUser).Scan(&Rights)
		if err != nil {
			report.ErrorServer(nil, err)
			return &entity.User{}, err
		}
		return &entity.User{
			ID:            IDUser,
			Rights:        entity.Rights(Rights),
			Authenticated: true,
		}, nil
	}
	return &entity.User{}, errors.New(`пользователь не найден`)
}

func (r *UserRepository) Reg(data entity.User, IDUser int) (user *entity.User, err error) {
	db := config.ConnectDB()
	var Rights, ID int
	if data.Login == `newman` {
		Count := 0
		query := `SELECT COUNT(*) FROM users.accounts`
		err := db.QueryRow(query).Scan(&Count)
		if err != nil {
			report.ErrorServer(nil, err)
			return &entity.User{}, err
		}
		data.Login += strconv.Itoa(Count + 1)
		Rights = 0
	} else {
		Rights = 1
	}

	LoginExist, err := r.FindLogin(data)
	if LoginExist {
		return &entity.User{}, errors.New(`Пользователь уже существует`)
	}

	Hash := HashCreation(data.Password)
	if data.Authenticated {
		IDUser, err := logLoggedAccount(data, Hash, user.Rights)
		if err != nil {
			return &entity.User{}, err
		}
		ID = IDUser
	} else {
		err := logNotLoggedAccount(data, IDUser, Hash, user.Rights)
		if err != nil {
			return &entity.User{}, err
		}
		ID = IDUser
	}
	user = &entity.User{
		ID:            ID,
		Rights:        entity.Rights(Rights),
		Authenticated: true,
		Login:         data.Login,
	}
	err = r.SetOnline(*user)
	if err != nil {
		return &entity.User{}, err
	}
	return user, nil
}

func logLoggedAccount(data entity.User, Hash uint32, Rights entity.Rights) (int, error) {
	db := config.ConnectDB()
	var ID int
	query := `INSERT INTO users.accounts (login, not_logged, password, rights) VALUES ($1, $2, $3, $4) RETURNING id`
	params := []any{data.Login, data.Authenticated, Hash, Rights}
	err := db.QueryRow(query, params...).Scan(&ID)
	if err != nil {
		report.ErrorServer(nil, err)
		return -1, err
	}
	return ID, nil
}

func logNotLoggedAccount(data entity.User, IDUser int, Hash uint32, Rights entity.Rights) error {
	db := config.ConnectDB()
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users.accounts WHERE id = $1 AND login not like 'newman%')`
	params := []any{IDUser}
	err := db.QueryRow(query, params...).Scan(&exists)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return err
	}
	if exists {
		return errors.New(`Вы уже в аккаунте`)
	}
	query = `UPDATE users.accounts SET login = $1, not_logged = false, password = $2, rights = $3, mail = $4, promocode = $5 WHERE id = $6`
	params = []any{data.Login, Hash, Rights, data.Mail, data.Promocode, IDUser}
	_, err = db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return err
	}
	query = `INSERT INTO users.trophies (id_user, trophies, updated_at) VALUES ($1, 0, $2)`
	params = []any{IDUser, time.Now()}
	_, err = db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
		return err
	}
	return nil
}

func (r *UserRepository) IsLogin(w http.ResponseWriter, req *http.Request) (user *entity.User) {
	session, err := config.Store.Get(req, "cookie-name")
	//	fmt.Printf("%+v\n", session)
	//	fmt.Printf("%+v\n", session.Options)
	if err != nil {
		report.ErrorServer(req, err)
		return
	}
	user = r.Get(session)
	if auth := user.Authenticated; !auth {
		var data entity.User
		data.Authenticated = true
		data.Login = `newman`
		user, err = r.Reg(data, 0)
		if err != nil {
			session.Values["user"] = user
			err = session.Save(req, w)
			if err != nil {
				report.ErrorServer(req, err)
				return &entity.User{}
			}
		} else {
			//	result.ReturnJSON(w, &res)
			return &entity.User{}
		}
		//result.ReturnJSON(w, &res)
	}
	r.SetOnline(*user)
	return user
}
