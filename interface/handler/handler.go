package handler

import (
	"../../cache/usermanager"
	"crypto/md5"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

func handleRegist(w http.ResponseWriter, r *http.Request) {
	log.Println("request:" + r.URL.Path)
	userName := r.FormValue("userName")
	macAddr := r.FormValue("macAddr")
	timeStamp := r.FormValue("timeStamp")
	result := 0
	responseStr := ""

	certStr := userName + macAddr + timeStamp
	cert := md5.Sum([]byte(certStr))
	certHexStr := ""

	for _, b := range cert {
		certHexStr += strconv.FormatInt(int64(b), 16)
	}

	userManager := usermanager.GetInstance()
	err := userManager.AddUser(userName, certHexStr, macAddr)
	if err == nil {
		result = 1
	}

	fmt.Sprintf(responseStr, "{result:'%d', cert:'%s', error:'%d'}", result, certHexStr, err)

	w.Write([]byte(responseStr))
}

func handleLogin(w http.ResponseWriter, r *http.Request) {

}

func handleGetReward(w http.ResponseWriter, r *http.Request) {

}

func handleGetResult(w http.ResponseWriter, r *http.Request) {

}

func handleStartBattle(w http.ResponseWriter, r *http.Request) {

}

func handleUploadRecord(w http.ResponseWriter, r *http.Request) {

}

func handleFindEnemy(w http.ResponseWriter, r *http.Request) {

}

func Init() {
	http.HandleFunc("/regist", handleRegist)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/getReward", handleRegist)
	http.HandleFunc("/getResult", handleGetResult)
	http.HandleFunc("/startBattle", handleStartBattle)
	http.HandleFunc("/uploadRecord", handleUploadRecord)
	http.HandleFunc("/findEnemy", handleFindEnemy)

	log.Fatal(http.ListenAndServe(":14000", nil))
}
