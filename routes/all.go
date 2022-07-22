package routes

import (
	"futble/game"

	"github.com/gorilla/mux"
)

func GetAllHandlers(r *mux.Router) {
	r.HandleFunc("/api/register", game.RegHandler)
	r.HandleFunc("/api/login", game.LoginHandler)
	r.HandleFunc("/api/test", game.TestHandler)
	r.HandleFunc("/api/all_names", game.PlayerListHandler)
	GetDailyHandler(r)
	GetRatingHandler(r)
}

func GetDailyHandler(r *mux.Router) {
	r.HandleFunc("/api/game", game.GameHandler)
	r.HandleFunc("/api/answer", game.GameAnswerHandler)
	r.HandleFunc("/api/daily/stats", game.GetDailyStatsHandler)
	r.HandleFunc("/api/switch_mode", game.SwitchModeHandler)
	r.HandleFunc("/api/unlimited/new", game.UnlimitedNewHandler)
	r.HandleFunc("/api/unlimited/avg_time", game.UnlimitedAvgHandler)

	//	r.HandleFunc("/api/rating/answer", game.RatingAnswerHandler)

}

func GetRatingHandler(r *mux.Router) {
	r.HandleFunc("/api/start_random", game.SearchRatingGameHandler)
	//	r.HandleFunc("/api/cancel_random", game.CancelSearchRatingGameHandler)
}
