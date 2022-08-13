package daemon

import (
	"fmt"
	"futble/config"
	"futble/report"
	"sync"
	"time"
)

type InviteGames struct {
	User1      int
	UserLogin  string
	Found      bool
	Rejected   bool
	StartTime  time.Time
	ExpiryTime time.Time
}

type SafeInviteGames struct {
	Items map[int][]InviteGames
	Mutex sync.Mutex
}

var Invites SafeInviteGames

func InviteSearch() {
	Invites.Items = make(map[int][]InviteGames)
	db := config.ConnectDB()
	query := `SELECT user1, user2, expiry FROM users.invites WHERE searching = TRUE AND is_invite = TRUE`
	rows, err := db.Query(query)
	if err != nil {
		report.ErrorSQLServer(nil, err, query)
		return
	}
	Invites.Mutex.Lock()
	for rows.Next() {
		var i InviteGames
		i.Found = false
		var User2 int
		err = rows.Scan(&i.User1, &User2, &i.ExpiryTime)
		if err != nil {
			report.ErrorServer(nil, err)
		}
		Invites.Items[User2] = append(Invites.Items[User2], i)
	}
	Invites.Mutex.Unlock()
	for {
		fmt.Printf("Item - %+v\n", Invites.Items)
		Invites.Mutex.Lock()
		for key, value := range Invites.Items {
			for i := 0; i < len(value); i++ {
				if value[i].ExpiryTime.Before(time.Now()) {
					go DiscardSearch(value[i].User1)
					value = RemoveElements(value, i)
					i--
					Invites.Items[key] = value
				}
			}
		}
		Invites.Mutex.Unlock()
		time.Sleep(10 * time.Second)
	}
}

func DiscardSearch(IDUser int) {
	db := config.ConnectDB()
	query := `UPDATE users.invites SET searching = false, finish_search = $1 WHERE user1 = $2 AND finish_search IS NULL`
	params := []any{time.Now(), IDUser}
	_, err := db.Exec(query, params...)
	if err != nil {
		report.ErrorSQLServer(nil, err, query, params...)
	}
}

func RemoveElements(InputSlice []InviteGames, i int) (ReturnSlice []InviteGames) {
	ReturnSlice = append(InputSlice[:i], InputSlice[i+1:]...)
	return
}
