package apiplayer

import (
	infrastructure "futble/infrastructure/player/repository"
	"futble/report"
	"futble/result"
	"net/http"
)

func PlayerListHandler(w http.ResponseWriter, r *http.Request) {
	var res result.ResultInfo
	playerRep := infrastructure.NewPlayerRepository()
	PlayersList, err := playerRep.GetAll()
	if err != nil {
		report.ErrorServer(r, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	res.Done = true
	res.Items = PlayersList
	result.ReturnJSON(w, &res)
}
