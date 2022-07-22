package config

import "time"

type SendType struct {
	IDUser int
	Start  time.Time
}

type ReceiveType struct {
	IDUser1 int
	IDUser2 int
}

var In chan SendType
var Out chan ReceiveType

func InitChannels() {
	In = make(chan SendType)
	Out = make(chan ReceiveType)
}
