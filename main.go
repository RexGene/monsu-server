package main

import (
	"./cache/recordmanager"
	"./cache/usermanager"
	"./interface/handler"
	"log"
	"time"
)

var synChan chan int

func initData() {
	err := recordmanager.GetInstance().LoadData()
	if err != nil {
		log.Fatal(err)
	}

	err = usermanager.GetInstance().LoadUser()
	if err != nil {
		log.Fatal(err)
	}

	synChan = make(chan int, 1)
}

func handleCmd() {
	for {
		select {
		case <-time.After(time.Minute * 1):
			updateDB()
		}
	}
}

func updateDB() {
	synChan <- 1
	defer func() { <-synChan }()

	recordmanager.GetInstance().UpdateToDB()
	usermanager.GetInstance().UpdateUserToDB()
}

func main() {
	initData()
	log.Println("init data finish")

	go handleCmd()
	handler.Init(synChan)
}
