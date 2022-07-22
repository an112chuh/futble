package daemon

import (
	"futble/config"
	"sort"
	"sync"
	"time"
)

type GameFinder struct {
	ID        int
	Rating    int
	TimeStart time.Time
}

type SafeGameFinder struct {
	Items []GameFinder
	Mutex sync.Mutex
}

var SearchList SafeGameFinder

func SearchingOpponent(In chan config.SendType, Out chan config.ReceiveType) {
	for {
		SearchList.Mutex.Lock()
		sort.SliceStable(SearchList.Items, func(i int, j int) bool {
			return SearchList.Items[i].Rating < SearchList.Items[j].Rating
		})
		for i := 0; i < len(SearchList.Items); i++ {

		}
		SearchList.Mutex.Unlock()
	}
}
