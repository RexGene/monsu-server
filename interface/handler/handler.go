package handler

import (
	"../../cache/usermanager"
	"crypto/md5"
	"fmt"
	"log"
	"net/http"
	"strings"
)

var synChan chan int
var tokenMap map[string]string

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
	log.Println(r.URL)

	synChan <- 1
	defer func() { <-synChan }()

	result := 0
	responseStr := ""
	certHexStr := ""
	msg := ""
	var err error

	defer func() {
		responseStr = fmt.Sprintf("{\"result\":%d, \"cert\":\"%s\", \"msg\":\"%s\"}", result, certHexStr, msg)
		w.Write([]byte(responseStr))
	}()

	userName := r.FormValue("userName")
	if !isStringValid(userName) {
		msg = "userName invalid"
		log.Println(msg)
		return
	}

	macAddr := r.FormValue("macAddr")
	if !isStringValid(macAddr) {
		msg = "macAddr invalid"
		log.Println(msg)
		return
	}

	timeStamp := r.FormValue("timeStamp")
	if !isStringValid(timeStamp) {
		msg = "timeStamp invalid"
		log.Println(msg)
		return
	}

	certStr := userName + macAddr + timeStamp
	cert := md5.Sum([]byte(certStr))

	for _, b := range cert {
		certHexStr += fmt.Sprintf("%.2x", b)
	}

	userManager := usermanager.GetInstance()
	err = userManager.AddUser(userName, certHexStr, macAddr)
	if err == nil {
		result = 1
	} else {
		msg = err.Error()
		certHexStr = ""
	}
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)

	synChan <- 1
	defer func() { <-synChan }()

	result := 0
	token := ""
	msg := ""
	var surplusAmount [2]int

	defer func() {
		responseStr := fmt.Sprintf("{\"result\":%d, \"surplusAmount\":[%d, %d], \"token\":\"%s\", \"msg\":\"%s\"}",
			result, surplusAmount[0], surplusAmount[1], token, msg)
		w.Write([]byte(responseStr))
	}()

	userName := r.FormValue("userName")
	if !isStringValid(userName) {
		msg = "userName invalid"
		log.Println(msg)
		return
	}

	timeStamp := r.FormValue("timeStamp")
	if !isStringValid(timeStamp) {
		msg = "timeStamp invalid"
		log.Println(msg)
		return
	}

	token = r.FormValue("token")
	if !isStringValid(token) {
		msg = "token invalid"
		log.Println(msg)
		return
	}

	userManager := usermanager.GetInstance()
	user, err := userManager.GetUser(userName)
	if err != nil {
		msg = err.Error()
		log.Println(msg)
		return
	}

	sumStr := userName + timeStamp + user.PasswordSum
	log.Println(sumStr)

	sumValue := md5.Sum([]byte(sumStr))
	sumHexStr := ""

	for _, b := range sumValue {
		sumHexStr += fmt.Sprintf("%.2x", b)
	}

	fmt.Printf("sumHexStr:%v", []byte(sumHexStr))
	fmt.Printf("token:%v", []byte(token))
	if sumHexStr != token {
		msg = "token not same"
		log.Println("sumHexStr:" + sumHexStr)
		log.Println(msg)
		return
	}

	if user.Token != "" {
		delete(tokenMap, user.Token)
	}

	user.Token = sumHexStr
	tokenMap[sumHexStr] = userName

	result = 1
	surplusAmount[0] = int(user.GoldCount)
	surplusAmount[1] = int(user.DiamondCount)
}

func handleChangeUserName() {

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
	tokenMap = make(map[string]string)

	http.HandleFunc("/regist", handleRegist)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/startBattle", handleStartBattle)
	http.HandleFunc("/uploadRecord", handleUploadRecord)
	http.HandleFunc("/findEnemy", handleFindEnemy)

	log.Fatal(http.ListenAndServe(":14000", nil))
}
