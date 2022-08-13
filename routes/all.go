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

}

func GetRatingHandler(r *mux.Router) {
	r.HandleFunc("/api/rating/start_search", game.SearchRatingGameHandler)
	r.HandleFunc("/api/rating/score", game.RatingScoreHandler)
	r.HandleFunc("/api/rating/stats", game.RatingGameStatsHandler)
	r.HandleFunc("/api/rating/coins", game.RatingCoinsHandler)
	r.HandleFunc("/api/rating/standings", game.RatingStandingsHandler)
	r.HandleFunc("/api/rating/send_invite", game.RatingSendInviteHandler)
	r.HandleFunc("/api/rating/notifications", game.RatingNotificationsHandler)
	r.HandleFunc("/api/rating/confirm_invite", game.RatingConfirmInviteHandler)
	r.HandleFunc("/api/rating/reject_invite", game.RatingRejectInviteHandler)
	r.HandleFunc("/api/rating/user_search", game.RatingUserSearchHandler)
	r.HandleFunc("/api/rating/result", game.RatingResultHandler)
	r.HandleFunc("/api/rating/hint_prices", game.RatingHintPricesHandler)
	r.HandleFunc("/api/rating/hint", game.RatingHintHandler)

}
