package main

import (
	"./cache/recordmanager"
	"./cache/resultmanager"
	"./cache/usermanager"
	"./interface/handler"
	"log"
	"time"
)

var synChan chan int

func initData() {
	recordmanager.GetInstance().LoadData()
	resultmanager.GetInstance().LoadData()
	usermanager.GetInstance().LoadUser()

	synChan = make(chan int, 1)
}

func handleCmd() {
	for {
		select {
		case <-time.After(time.Minute * 10):
			updateDB()
		}
	}
}

func updateDB() {
	synChan <- 1
	defer func() { <-synChan }()

	recordmanager.GetInstance().UpdateToDB()
	resultmanager.GetInstance().UpdateToDB()
	usermanager.GetInstance().UpdateUserToDB()
}

func main() {
	initData()
	log.Println("init data finish")

	go handleCmd()
	handler.Init(synChan)
}
