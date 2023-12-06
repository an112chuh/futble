package entity

type User struct {
	ID            int     `json:"id"`
	Login         string  `json:"login"`
	Password      string  `json:"password"`
	Mail          *string `json:"mail,omitempty"`
	Rights        Rights
	Promocode     *string `json:"promocode,omitempty"`
	Authenticated bool
}

type Rights int

const (
	NotLogged Rights = 0
	Default   Rights = 1
	Admin     Rights = 2
)
