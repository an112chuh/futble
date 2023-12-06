package main

import (
	"encoding/gob"
	"fmt"
	"futble/check"
	"futble/config"
	"futble/constants"
	"futble/daemon"
	"futble/entity"
	"futble/game"
	"futble/routes"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

var IsOpeningLocal bool

func main() {
	IsOpeningLocal = false
	var AdminName string
	if len(os.Args) == 2 {
		IsOpeningLocal = true
		AdminName = os.Args[1]
	}
	config.InitCookies()
	config.InitRandom()
	config.InitDB(IsOpeningLocal, AdminName)
	config.InitLoggers()
	config.InitChannels()
	err := constants.InitNations()
	if err != nil {
		fmt.Println(err)
		return
	}
	check.CheckNationsExist(constants.NationMatches)
	check.CheckNationCorrect()
	check.CheckClubCorrect()
	check.CheckLeagueCorrect()
	check.DownloadIDs()
	// game.AddDailyGames()
	//	go daemon.TestCountRateDiff()
	go daemon.SearchList.SearchingOpponent()
	go daemon.RatingGameFinishing()
	go daemon.ClearDatabase()
	go daemon.InviteSearch()
	go daemon.CommandLine()
	go game.FinishingGames()
	gob.Register(entity.User{})
	routeAll := mux.NewRouter()
	routes.GetAllHandlers(routeAll)
	routeAll.Use(mw)
	http.Handle("/", routeAll)
	var APP_IP, APP_PORT string
	if IsOpeningLocal {
		APP_IP = "127.0.0.1"
		APP_PORT = "8080"
	} else {
		APP_IP = "127.0.0.1"
		APP_PORT = "8080"
	}
	fmt.Println("[SERVER] Server address is " + APP_IP + ":" + APP_PORT)
	http.ListenAndServe(APP_IP+":"+APP_PORT, nil)
	fmt.Println("[SERVER] Server is started")
	defer config.Db.Close()
}

func mw(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
