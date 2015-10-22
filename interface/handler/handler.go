package handler

import (
	"../../cache/recordmanager"
	"../../cache/usermanager"
	"crypto/md5"
	"fmt"
	_ "github.com/RexGene/csvparser"
	"log"
	"net/http"
	_ "sort"
	"strconv"
	"strings"
	"time"
)

const (
	goldCostType    = 1
	diamondCostType = 2
)

var synChan chan int
var tokenMap map[string]string
var enemyMap map[uint64]*enemyInfo

type enemyInfo struct {
	name        string
	roleId      uint
	mountId     uint
	petId       uint
	weaponId    uint
	equipmentId uint
	scores      uint
	isRobot     uint
	records     string
}

func getTotalDay() (int64, error) {
	return (time.Now().Unix() - 3600*8) / 86400, nil
}

func calcLastDayRank(uuid uint64, t int) (rank int, err error) {
	records, err := recordmanager.GetInstance().GetUserRecords(uuid, t)
	if err != nil {
		return
	}

	recordLen := len(records)
	if recordLen == 0 {
		return
	}

	/*
		today, err := getTotalDay()
		if err != nil {
			return
		}

		//sort
		yesterday := today - 1

		beginIndex := -1
		endIndex := -1
		for i := recordLen - 1; i == 0; i-- {
			totalDay := records[i].TotalDay

			if totalDay < yesterday {
				beginIndex = i + 1
			} else if totalDay == yesterday {
				if endIndex == -1 {
					endIndex = i + 1
				}
			}
		}*/

	return
}

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
	msg := "success"
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
	msg := "success"
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

	if sumHexStr != token {
		msg = "token not same"
		token = ""
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

func handleChangeUserName(w http.ResponseWriter, r *http.Request) {
	synChan <- 1
	defer func() { <-synChan }()

	log.Println(r.URL)

	msg := "success"
	result := 0
	defer func() {
		responseStr := fmt.Sprintf("{\"result\":%d, \"msg\":\"%s\"}", result, msg)
		w.Write([]byte(responseStr))
	}()

	userName := r.FormValue("userName")
	if !isStringValid(userName) {
		msg = "userName invalid"
		log.Println(msg)
		return
	}

	newUserName := r.FormValue("newUserName")
	if !isStringValid(newUserName) {
		msg = "newUserName invalid"
		log.Println(msg)
		return
	}

	token := r.FormValue("token")
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

	if token != user.Token {
		msg = "token not match"
		log.Println(msg)
		return
	}

	err = userManager.ChangeName(user.Uuid, newUserName)
	if err != nil {
		msg = err.Error()
		log.Println(msg)
		return
	}

	result = 1

	userManager.MarkUserChange(user.UserName)
}

func handleGetTime(w http.ResponseWriter, r *http.Request) {
	resultData := fmt.Sprintf("{\"timeStamp\":%d }", time.Now().Unix())
	w.Write([]byte(resultData))
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

	msg := "success"
	result := 0

	var info *enemyInfo

	defer func() {
		if info == nil {
			info = new(enemyInfo)
		}
		responseStr := fmt.Sprintf("{\"result\":%d, \"enemyName\":\"%s\", \"enemyRoleId\":%d, \"enemyMountId\":%d, \"enemyWeaponId\":%d, \"enemyEquipmentId\":%d, \"enemyPetId\":%d, \"records\":\"%s\", \"scores\":%d , \"isRobot\":%d, \"msg\":\"%s\"}",
			result, info.name, info.roleId, info.mountId, info.weaponId, info.equipmentId,
			info.petId, info.records, info.scores, info.isRobot, msg)
		w.Write([]byte(responseStr))
	}()

	token := r.FormValue("token")
	if isStringValid(token) {
		msg = "token invalid"
		log.Println(msg)
		return
	}

	costTypeStr := r.FormValue("costType")
	costType, err := strconv.ParseInt(costTypeStr, 10, 32)
	if err != nil {
		msg = err.Error()
		log.Println(msg)
		return
	}

	userName := tokenMap[token]
	if userName != "" {
		msg = "token not found user"
		log.Println(msg)
		return
	}

	user, err := usermanager.GetInstance().GetUser(userName)
	if err != nil {
		msg = err.Error()
		log.Println(msg)
		return
	}

	if user.Token != token {
		msg = "token not match"
		log.Println(msg)
		return
	}

	switch costType {
	case goldCostType:
		info, err = getEnemyData(user.GoldRank, int(costType))
		if err != nil {
			msg = err.Error()
			log.Println(msg)
			return
		}
	case diamondCostType:
		info, err = getEnemyData(user.DiamondRank, int(costType))
		if err != nil {
			msg = err.Error()
			log.Println(msg)
			return
		}
	default:
		msg = "invalid costType"
		log.Println(msg)
		return
	}

	result = 1
}

func getEnemyData(scores uint, costType int) (*enemyInfo, error) {
	record, err := recordmanager.GetInstance().GetRecord(uint(scores), 0, int(costType))
	if err != nil {
		if err == recordmanager.ErrUserNotFound {
			return &enemyInfo{
				name:        "Guest",
				roleId:      3,
				petId:       1,
				mountId:     1,
				weaponId:    4,
				equipmentId: 10,
				isRobot:     1,
				scores:      199999,
				records:     "",
			}, nil
		} else {
			return nil, err
		}
	} else {
		return &enemyInfo{
			name:        record.UserName,
			roleId:      record.RoleId,
			petId:       record.PetId,
			mountId:     record.MountId,
			weaponId:    record.WeaponId,
			equipmentId: record.EquipmentId,
			scores:      record.Scores,
			records:     record.Records,
			isRobot:     0,
		}, nil
	}
}

func Init(synChannel chan int) {
	synChan = synChannel
	tokenMap = make(map[string]string)

	http.HandleFunc("/regist", handleRegist)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/startBattle", handleStartBattle)
	http.HandleFunc("/uploadRecord", handleUploadRecord)
	http.HandleFunc("/getTime", handleGetTime)
	http.HandleFunc("/changeUserName", handleChangeUserName)
	http.HandleFunc("/findEnemy", handleFindEnemy)

	log.Fatal(http.ListenAndServe(":14000", nil))
}
