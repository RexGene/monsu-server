package handler

import (
	"../../cache/usermanager"
	"crypto/md5"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var synChan chan int

func isStringValid(str string) bool {
	str = strings.TrimSpace(str)
	str = strings.ToLower(str)

	strlen := len(str)

	if strlen == 0 || strlen > 32 {
		return false
	}

	for _, c := range str {
		if (c < '0' || c > '9') && (c < 'a' || c > 'z') && c != '_' && c != '-' && c < 128 {
			return false
		}
	}

	return true
}

func handleRegist(w http.ResponseWriter, r *http.Request) {
	synChan <- 1
	defer func() { <-synChan }()

	log.Println(r.URL)

	result := 0
	responseStr := ""
	certHexStr := ""
	var err error

	defer func() {
		responseStr = fmt.Sprintf("{result:'%d', cert:'%s'}", result, certHexStr)
		w.Write([]byte(responseStr))
	}()

	userName := r.FormValue("userName")
	if !isStringValid(userName) {
		log.Println("userName invalid")
		return
	}

	macAddr := r.FormValue("macAddr")
	if !isStringValid(macAddr) {
		log.Println("macAddr invalid")
		return
	}

	timeStamp := r.FormValue("timeStamp")
	if !isStringValid(timeStamp) {
		log.Println("timeStamp invalid")
		return
	}

	certStr := userName + macAddr + timeStamp
	cert := md5.Sum([]byte(certStr))

	for _, b := range cert {
		certHexStr += strconv.FormatInt(int64(b), 16)
	}

	userManager := usermanager.GetInstance()
	err = userManager.AddUser(userName, certHexStr, macAddr)
	if err == nil {
		result = 1
	}

}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	synChan <- 1
	defer func() { <-synChan }()

}

func handleGetReward(w http.ResponseWriter, r *http.Request) {
	synChan <- 1
	defer func() { <-synChan }()

}

func handleGetResult(w http.ResponseWriter, r *http.Request) {
	synChan <- 1
	defer func() { <-synChan }()

}

func handleStartBattle(w http.ResponseWriter, r *http.Request) {
	synChan <- 1
	defer func() { <-synChan }()

}

func handleUploadRecord(w http.ResponseWriter, r *http.Request) {
	synChan <- 1
	defer func() { <-synChan }()

}

func handleFindEnemy(w http.ResponseWriter, r *http.Request) {
	synChan <- 1
	defer func() { <-synChan }()

}

func Init(synChannel chan int) {
	synChan = synChannel

	http.HandleFunc("/regist", handleRegist)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/getReward", handleRegist)
	http.HandleFunc("/getResult", handleGetResult)
	http.HandleFunc("/startBattle", handleStartBattle)
	http.HandleFunc("/uploadRecord", handleUploadRecord)
	http.HandleFunc("/findEnemy", handleFindEnemy)

	log.Fatal(http.ListenAndServe(":14000", nil))
}
