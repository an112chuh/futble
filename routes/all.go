package routes

import (
	"futble/game"

	"github.com/gorilla/mux"
)

func GetAllHandlers(r *mux.Router) {
	r.HandleFunc("/api/register", game.RegHandler)
	r.HandleFunc("/api/login", game.LoginHandler)
	r.HandleFunc("/api/logout", game.LogoutHandler)
	r.HandleFunc("/api/test", game.TestHandler)
	r.HandleFunc("/api/all_names", game.PlayerListHandler)
	GetGameHandler(r)
	GetRatingHandler(r)
	GetMessagesHandler(r)
	GetAdminHandler(r)
}

func GetGameHandler(r *mux.Router) {
	r.HandleFunc("/api/game", game.GameHandler)
	r.HandleFunc("/api/answer", game.GameAnswerHandler)
	r.HandleFunc("/api/daily/stats", game.GetDailyStatsHandler)
	r.HandleFunc("/api/switch_mode", game.SwitchModeHandler)
	r.HandleFunc("/api/unlimited/new", game.UnlimitedNewHandler)
	r.HandleFunc("/api/unlimited/avg_time", game.UnlimitedAvgHandler)
	//	r.HandleFunc("/api/notification", game.NotificationsHandler)
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
	r.HandleFunc("/api/rating/hint_opponent", game.RatingHintOpponentHandler)
}

func GetMessagesHandler(r *mux.Router) {
	r.HandleFunc("/api/messages/list", game.MessagesListHandler)
	r.HandleFunc("/api/messages/create", game.MessagesCreateHandler)
	r.HandleFunc("/api/messages/close_request/{id:[0-9]+}", game.MessagesCloseRequestHandler)
	r.HandleFunc("/api/messages/item/{id:[0-9]+}", game.MessagesItemHandler)
	r.HandleFunc("/api/messages/answer/{id:[0-9]+}", game.MessagesAnswerHandler)
}

func GetAdminHandler(r *mux.Router) {
	r.HandleFunc("/api/admin/messages/list", game.AdminMessagesListHandler)
	r.HandleFunc("/api/admin/messages/answer/{id:[0-9]+}", game.AdminMessagesAnswerHandler)
}
