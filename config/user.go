package config

import (
	"log"

	"gopkg.in/natefinch/lumberjack.v2"
)

var ErrorLog *log.Logger
var AccessLog *log.Logger

func InitLoggers() {
	ErrorFile := &lumberjack.Logger{
		Filename:   "./logs/errors.log",
		MaxSize:    250,
		MaxBackups: 5,
		MaxAge:     10,
	}
	ErrorLog = log.New(ErrorFile, "ERROR ", log.Ldate|log.Ltime|log.Lshortfile)
	AccessLog = log.New(ErrorFile, "SERVER ", log.Ldate|log.Ltime)
}
