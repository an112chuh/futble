package routes

import (
	"futble/game"

	"github.com/gorilla/mux"
)

func GetAllHandlers(r *mux.Router) {
	r.HandleFunc("/api/register", game.RegHandler)
	r.HandleFunc("/api/login", game.LoginHandler)
	r.HandleFunc("/api/test", game.TestHandler)
	r.HandleFunc("/api/find", game.FindPlayerHandler)
	r.HandleFunc("/api/stats/week", game.GetWeekStatsHandler)
	r.HandleFunc("/api/stats/month", game.GetMonthStatsHandler)
	r.HandleFunc("/api/stats/year", game.GetYearStatsHandler)
	GetDailyHandler(r)
}

func GetDailyHandler(r *mux.Router) {
	r.HandleFunc("/api/daily", game.DailyHandler)
	r.HandleFunc("/api/daily/answer", game.DailyAnswerHandler)
	r.HandleFunc("/api/daily/stats", game.GetDailyStatsHandler)
	r.HandleFunc("/api/unlimited", game.UnlimitedHandler)
	r.HandleFunc("/api/unlimited/answer", game.UnlimitedAnswerHandler)
	r.HandleFunc("/api/rating", game.RatingHandler)
	r.HandleFunc("/api/rating/answer", game.RatingAnswerHandler)
}
