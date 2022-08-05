package daemon

import (
	"fmt"
	"futble/report"
	"os"
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

type GameResult struct {
	Home          int
	IsDeletedHome bool
	Away          int
	IsDeletedAway bool
}

type SafeGameResult struct {
	Items []GameResult
	Mutex sync.Mutex
}

var RATE_DIFF_BEFORE_10 int = 50
var RATE_DIFF_BEFORE_20 int = 150
var RATE_DIFF_BEFORE_30 int = 300

var SearchList SafeGameFinder
var ResultList SafeGameResult

var SearchingActive bool = true

func (SearchList *SafeGameFinder) SearchingOpponent(SearchingActive bool) {
	file, err := os.Create("logs/balancer.txt")
	if err != nil {
		report.ErrorServer(nil, err)
		return
	}
	defer file.Close()
	for {
		SearchList.Mutex.Lock()
		sort.SliceStable(SearchList.Items, func(i int, j int) bool {
			return SearchList.Items[i].Rating > SearchList.Items[j].Rating
		})
		for i := 0; i < len(SearchList.Items); i++ {
			if i == 1 {
				fmt.Println(len(SearchList.Items))
			}
			var RateDiff int
			if time.Since(SearchList.Items[i].TimeStart) < 10*time.Second {
				RateDiff = RATE_DIFF_BEFORE_10
			} else if time.Since(SearchList.Items[i].TimeStart) < 20*time.Second {
				RateDiff = RATE_DIFF_BEFORE_20
			} else if time.Since(SearchList.Items[i].TimeStart) < 30*time.Second {
				RateDiff = RATE_DIFF_BEFORE_30
			} else {
				RateDiff = 999999
			}
			for j := i + 1; j < len(SearchList.Items); j++ {
				if abs(SearchList.Items[i].Rating-SearchList.Items[j].Rating) < RateDiff {
					ResultList.Mutex.Lock()
					var g GameResult
					g.Home = SearchList.Items[i].ID
					g.Away = SearchList.Items[j].ID
					g.IsDeletedHome = false
					g.IsDeletedAway = false
					ResultList.Items = append(ResultList.Items, g)
					ResultList.Mutex.Unlock()
					SearchList.RemoveElements(i, j)
					i--
					break
				}
			}
		}
		SearchList.Mutex.Unlock()
	}
}

func (SearchList *SafeGameFinder) RemoveElements(i int, j int) {
	NewSlice := append(SearchList.Items[:i], SearchList.Items[i+1:]...)
	j--
	NewSlice2 := append(NewSlice[:j], NewSlice[j+1:]...)
	SearchList.Items = NewSlice2
}

func abs(i int) int {
	if i > 0 {
		return i
	}
	return -i
}

func (ResultList *SafeGameResult) RemoveElements(i int) {
	NewSlice := append(ResultList.Items[:i], ResultList.Items[i+1:]...)
	ResultList.Items = NewSlice
}
