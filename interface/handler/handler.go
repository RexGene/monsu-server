package handler

import (
	"../../cache/configmanager"
	"../../cache/recordmanager"
	"../../cache/usermanager"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	goldCostType    = 1
	diamondCostType = 2
)

const (
	notFound = -1
)

const (
	defaultRank      = 5000
	defualtLimit     = 3
	defaultAvgAmount = 4
)

type JsonInfo struct {
	Id         string
	Error_Code string
	Error      string
}

var synChan chan int
var tokenMap map[string]string
var enemyMap map[uint64]*enemyInfo
var orderMap map[string]bool

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
	isDouble    uint
}

func checkUserUpdate(uuid uint64) (bool, error) {
	config, err := configmanager.GetInstance().GetConfig("config/const.csv")
	if err != nil {
		return false, err
	}

	user, err := usermanager.GetInstance().GetUserByUuid(uuid)
	if err != nil {
		return false, err
	}

	totalDay := uint(getTotalDay())
	log.Println("[debug]", "LastUpdateDay:", user.LastUpdateDay, "totalDay:", totalDay)

	//reset user data
	if user.LastUpdateDay != totalDay {
		goldRank, err := calcLastDayRank(uuid, goldCostType)
		if err != nil {
			return false, err
		}

		diamondRank, err := calcLastDayRank(uuid, diamondCostType)
		if err != nil {
			return false, err
		}

		user.GoldCount = uint8(config["DefaultGoldCount"]["value"].Uint(5))
		user.DiamondCount = uint8(config["DefaultDiamondCount"]["value"].Uint(5))
		user.GoldRank = uint(goldRank)
		user.DiamondRank = uint(diamondRank)
		user.GoldWinAmount = 0
		user.GoldLoseAmount = 0
		user.DiamondWinAmount = 0
		user.DiamondLoseAmount = 0
		user.GoldAvailableBuyCount = config["GoldAvailableBuyCount"]["value"].Uint(3)
		user.DiamondAvailableBuyCount = config["DiamondAvailableBuyCount"]["value"].Uint(3)
		user.FixLevel = 0
		user.DiamondFixLevel = 0

		user.LastUpdateDay = totalDay

		usermanager.GetInstance().MarkUserChange(user.UserName)
		return true, nil
	}

	return false, nil
}

func getTotalDay() int64 {
	return (time.Now().Unix() + 3600*8) / 86400
}

func getOldRank(t int, user *usermanager.User, rank *int, err *error) {
	config, e := configmanager.GetInstance().GetConfig("config/const.csv")
	if e != nil {
		*err = e
		return
	}

	defaultRank := config["DefaultRank"]["value"].Int(defaultRank)

	if t == goldCostType {
		*rank = int(user.GoldRank)
		if *rank < defaultRank {
			*rank = defaultRank
		}
	} else if t == diamondCostType {
		*rank = int(user.DiamondRank)
		if *rank < defaultRank {
			*rank = defaultRank
		}
	} else {
		*err = errors.New("cost type invalid:" + strconv.FormatInt(int64(t), 10))
	}
}

func calcLastDayRank(uuid uint64, t int) (rank int, err error) {
	config, err := configmanager.GetInstance().GetConfig("config/const.csv")
	if err != nil {
		return
	}

	records, err := recordmanager.GetInstance().GetUserRecords(uuid, t)
	if err != nil {
		return
	}

	defaultRank := config["DefaultRank"]["value"].Int(defaultRank)

	recordLen := 0
	if records != nil {
		recordLen = len(records)
	}

	if recordLen == 0 {
		log.Println("[info]", "record len is 0")
		rank = defaultRank
		return
	}

	today := getTotalDay()
	user, err := usermanager.GetInstance().GetUserByUuid(uuid)
	if err != nil {
		return
	}

	//sort
	yesterday := today - 1
	log.Println("[info]", "yesterday:", yesterday)

	//find yesterday record range
	beginIndex := notFound
	endIndex := notFound
	for i := recordLen - 1; i >= 0; i-- {
		totalDay := records[i].TotalDay
		log.Println("[info]", "record total day:", totalDay)
		if totalDay == yesterday {
			if endIndex == notFound {
				endIndex = i + 1
			}

			beginIndex = i
		}
	}

	log.Println("[info]", "beginIndex:", beginIndex)
	if beginIndex != notFound {
		log.Println("[info]", "beginIndex:", beginIndex, " endIndex:", endIndex)
		//yesterday exist
		s := records[beginIndex:endIndex]
		log.Println("[info]", "record len:", len(s))
		dl := config["DefualtLimit"]["value"].Int(defualtLimit)
		log.Println("[Info]", "DefualtLimit:", dl)
		if len(s) < config["DefualtLimit"]["value"].Int(defualtLimit) {
			getOldRank(t, user, &rank, &err)
		}

		sort.Sort(sort.Reverse(recordmanager.RecordSlice(s)))

		total := uint(0)
		size := config["DefaultAvgAmount"]["value"].Uint(defaultAvgAmount)
		log.Println("[info]", "DefaultAvgAmount:", size)
		l := len(s)
		if size > uint(l) {
			size = uint(l)
		}

		for _, r := range s[:size] {
			total += r.Scores
		}

		rank = int(total / size)
	} else {
		//not exist
		getOldRank(t, user, &rank, &err)
	}

	if rank < defaultRank {
		rank = defaultRank
	}

	return
}

func isStringValid(str string) bool {
	str = strings.TrimSpace(str)
	str = strings.ToLower(str)

	strlen := len(str)

	if strlen == 0 || strlen > 64 {
		return false
	}

	for _, c := range str {
		if (c < '0' || c > '9') && (c < 'a' || c > 'z') && c != '_' && c != '-' && c != '=' && c != '/' && c != '+' && c < 128 {
			return false
		}
	}

	return true
}

func handleRegist(w http.ResponseWriter, r *http.Request) {
	log.Println("[request]", r.URL)

	synChan <- 1
	defer func() { <-synChan }()

	result := 0
	responseStr := ""
	certHexStr := ""
	msg := "success"
	var err error

	defer func() {
		responseStr = fmt.Sprintf("{\"result\":%d, \"cert\":\"%s\", \"msg\":\"%s\"}", result, certHexStr, msg)
		log.Println("[response]", responseStr)
		w.Write([]byte(responseStr))
	}()

	userName := r.FormValue("userName")
	if !isStringValid(userName) {
		msg = "userName invalid"
		log.Println("[error]", msg)
		return
	}

	macAddr := r.FormValue("macAddr")
	if !isStringValid(macAddr) {
		msg = "macAddr invalid"
		log.Println("[error]", msg)
		return
	}

	timeStamp := r.FormValue("timeStamp")
	if !isStringValid(timeStamp) {
		msg = "timeStamp invalid"
		log.Println("[error]", msg)
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
	log.Println("[request]", r.URL)

	synChan <- 1
	defer func() { <-synChan }()

	result := 0
	token := ""
	msg := "success"
	var surplusAmount [2]int
	var availableCount [2]int

	defer func() {
		responseStr := fmt.Sprintf("{\"result\":%d, \"surplusAmount\":[%d, %d], \"availableCount\":[%d, %d], \"token\":\"%s\", \"msg\":\"%s\"}",
			result, surplusAmount[0], surplusAmount[1], availableCount[0], availableCount[1], token, msg)
		log.Println("[response]", responseStr)
		w.Write([]byte(responseStr))
	}()

	userName := r.FormValue("userName")
	if !isStringValid(userName) {
		msg = "userName invalid"
		log.Println("[error]", msg)
		return
	}

	timeStamp := r.FormValue("timeStamp")
	if !isStringValid(timeStamp) {
		msg = "timeStamp invalid"
		log.Println("[error]", msg)
		return
	}

	token = r.FormValue("token")
	if !isStringValid(token) {
		msg = "token invalid"
		log.Println("[error]", msg)
		return
	}

	userManager := usermanager.GetInstance()
	user, err := userManager.GetUser(userName)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	sumStr := userName + timeStamp + user.PasswordSum

	sumValue := md5.Sum([]byte(sumStr))
	sumHexStr := ""

	for _, b := range sumValue {
		sumHexStr += fmt.Sprintf("%.2x", b)
	}

	if sumHexStr != token {
		msg = "token not same"
		token = ""
		log.Println("sumHexStr:" + sumHexStr)
		log.Println("[error]", msg)
		return
	}

	if user.Token != "" {
		delete(tokenMap, user.Token)
	}

	_, err = checkUserUpdate(user.Uuid)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	user.Token = sumHexStr
	tokenMap[sumHexStr] = userName

	result = 1
	surplusAmount[0] = int(user.GoldCount)
	surplusAmount[1] = int(user.DiamondCount)
	availableCount[0] = int(user.GoldAvailableBuyCount)
	availableCount[1] = int(user.DiamondAvailableBuyCount)
}

func handleChangeUserName(w http.ResponseWriter, r *http.Request) {
	synChan <- 1
	defer func() { <-synChan }()

	log.Println("[request]", r.URL)

	msg := "success"
	result := 0
	defer func() {
		responseStr := fmt.Sprintf("{\"result\":%d, \"msg\":\"%s\"}", result, msg)
		log.Println(responseStr)
		w.Write([]byte(responseStr))
	}()

	token := r.FormValue("token")
	if !isStringValid(token) {
		msg = "token invalid"
		log.Println("[error]", msg)
		return
	}

	userName := tokenMap[token]
	if !isStringValid(userName) {
		msg = "userName invalid"
		log.Println("[error]", msg)
		return
	}

	newUserName := r.FormValue("newUserName")
	if !isStringValid(newUserName) {
		msg = "newUserName invalid"
		log.Println("[error]", msg)
		return
	}

	userManager := usermanager.GetInstance()
	user, err := userManager.GetUser(userName)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	err = userManager.ChangeName(user.Uuid, newUserName)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	tokenMap[token] = newUserName
	result = 1

	userManager.MarkUserChange(user.UserName)
}

func handleGetTime(w http.ResponseWriter, r *http.Request) {
	resultData := fmt.Sprintf("{\"timeStamp\":%d }", time.Now().Unix())
	w.Write([]byte(resultData))
}

func handleFindEnemy(w http.ResponseWriter, r *http.Request) {
	log.Println("[request]", r.URL)
	synChan <- 1
	defer func() { <-synChan }()

	msg := "success"
	result := 0

	goldCount := 0
	diamondCount := 0

	var info *enemyInfo

	defer func() {
		if info == nil {
			info = new(enemyInfo)
		}

		if info.records == "" {
			info.records = "null"
		}

		responseStr := fmt.Sprintf("{\"result\":%d, \"enemyName\":\"%s\", \"enemyRoleId\":%d, \"enemyMountId\":%d, \"enemyWeaponId\":%d, \"enemyEquipmentId\":%d, \"enemyPetId\":%d, \"records\":%s, \"scores\":%d , \"goldCount\":%d, \"diamondCount\":%d,  \"isRobot\":%d, \"msg\":\"%s\"}",
			result, info.name, info.roleId, info.mountId, info.weaponId, info.equipmentId,
			info.petId, info.records, info.scores, goldCount, diamondCount, info.isRobot, msg)
		log.Println("[response]", responseStr)
		w.Write([]byte(responseStr))
	}()

	token := r.FormValue("token")
	if !isStringValid(token) {
		msg = "token invalid"
		log.Println("[error]", msg)
		return
	}

	costTypeStr := r.FormValue("costType")
	costType, err := strconv.ParseInt(costTypeStr, 10, 32)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	log.Println("[info]type", costTypeStr)

	isDoubleStr := r.FormValue("isDouble")
	isDouble, err := strconv.ParseInt(isDoubleStr, 10, 8)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	userName := tokenMap[token]
	if userName == "" {
		msg = "token not found user"
		log.Println("[error]", msg)
		return
	}

	user, err := usermanager.GetInstance().GetUser(userName)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	switch costType {
	case goldCostType:
		if user.GoldCount == 0 {
			err = errors.New("gold count not enough")
			msg = err.Error()
			log.Println("[error]", msg)
			return
		}

		info, err = getEnemyData(user.GoldRank, user.FixLevel, int(costType), user.Uuid)
		if err != nil {
			msg = err.Error()
			log.Println("[error]", msg)
			return
		}

		user.GoldCount--
		info.isDouble = uint(isDouble)
	case diamondCostType:
		if user.DiamondCount == 0 {
			err = errors.New("diamond count not enough")
			msg = err.Error()
			log.Println("[error]", msg)
			return
		}

		info, err = getEnemyData(user.DiamondRank, user.DiamondFixLevel, int(costType), user.Uuid)
		if err != nil {
			msg = err.Error()
			log.Println("[error]", msg)
			return
		}

		user.DiamondCount--
		info.isDouble = uint(isDouble)
	default:
		msg = "invalid costType"
		log.Println("[error]", msg)
		return
	}

	enemyMap[user.Uuid] = info
	diamondCount = int(user.DiamondCount)
	goldCount = int(user.GoldCount)
	usermanager.GetInstance().MarkUserChange(userName)
	result = 1
}

func getEnemyData(scores uint, fix int, costType int, uuid uint64) (*enemyInfo, error) {
	config, err := configmanager.GetInstance().GetConfig("config/const.csv")
	if err != nil {
		return nil, err
	}

	nameConfig, err := configmanager.GetInstance().GetConfig("config/name.csv")
	if err != nil {
		return nil, err
	}

	record, err := recordmanager.GetInstance().GetRecord(uint(scores), fix, int(costType), uuid)
	if err != nil {
		if err == recordmanager.ErrUserNotFound {
			configLen := len(nameConfig)
			name := ""
			if configLen == 0 {
				name = "Guest"
			} else {
				name = nameConfig[strconv.FormatInt(int64(1+rand.Int()%configLen), 10)]["name"].Str()
			}

			zoneLen := config["ZoneRange"]["value"].Uint(1)
			index, err := recordmanager.GetInstance().GetIndex(uint(scores), fix)
			if err != nil {
				return nil, err
			}

			scores, err = recordmanager.GetInstance().GetScoresByLevel(int(index))
			if err != nil {
				return nil, err
			}

			defaultRank := config["DefaultRank"]["value"].Uint(defaultRank)
			if scores < defaultRank {
				scores = defaultRank
			}

			ex := uint(rand.Uint32() % uint32(zoneLen))

			enemyInfo := &enemyInfo{
				name: name,
				roleId: config["MinRoleId"]["value"].Uint(0) +
					uint(rand.Uint32())%config["RoleIdRange"]["value"].Uint(1),
				petId: config["MinPetId"]["value"].Uint(0) +
					uint(rand.Uint32())%config["PetIdRange"]["value"].Uint(1),
				mountId: config["MinMountId"]["value"].Uint(0) +
					uint(rand.Uint32())%config["MountIdRange"]["value"].Uint(1),
				weaponId: config["MinWeaponId"]["value"].Uint(0) +
					uint(rand.Uint32())%config["WeaponIdRange"]["value"].Uint(1),
				equipmentId: config["MinEquiptmentId"]["value"].Uint(0) +
					uint(rand.Uint32())%config["EquiptmentRange"]["value"].Uint(1),
				isRobot: 1,
				scores:  scores + ex,
				records: "",
			}

			return enemyInfo, nil
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

func handleBuyBattleAmount(w http.ResponseWriter, r *http.Request) {
	log.Println("[request]", r.URL)
	synChan <- 1
	defer func() { <-synChan }()

	msg := "success"
	result := 0
	var surplusAmount [2]int
	var availableCount [2]int

	defer func() {
		responseStr := fmt.Sprintf("{\"result\":%d, \"surplusAmount\":[%d, %d], \"availableCount\":[%d, %d], \"msg\":\"%s\"}", result, surplusAmount[0], surplusAmount[1], availableCount[0], availableCount[1], msg)
		log.Println("[response]", responseStr)
		w.Write([]byte(responseStr))
	}()

	buyType, err := strconv.ParseInt(r.FormValue("buyType"), 10, 32)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	token := r.FormValue("token")
	if token == "" {
		msg = "token invalid"
		log.Println("[error]", msg)
		return
	}

	userName := tokenMap[token]
	if userName == "" {
		msg = "token not found user"
		log.Println("[error]", msg)
		return
	}

	user, err := usermanager.GetInstance().GetUser(userName)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	switch buyType {
	case goldCostType:
		if user.GoldAvailableBuyCount == 0 {
			msg = "gold available buy count not enough"
			log.Println("[error]", msg)
			return
		} else {
			user.GoldAvailableBuyCount--
			user.GoldCount++
		}
	case diamondCostType:
		if user.DiamondAvailableBuyCount == 0 {
			msg = "diamond available buy count not enough"
			log.Println("[error]", msg)
			return
		} else {
			user.DiamondAvailableBuyCount--
			user.DiamondCount++
		}
	default:
		msg = "buy type invalid"
		log.Println("[error]", msg)
		return
	}

	surplusAmount[0] = int(user.GoldCount)
	surplusAmount[1] = int(user.DiamondCount)
	availableCount[0] = int(user.GoldAvailableBuyCount)
	availableCount[1] = int(user.DiamondAvailableBuyCount)

	usermanager.GetInstance().MarkUserChange(userName)
	result = 1
}

func handleUploadRecord(w http.ResponseWriter, r *http.Request) {
	log.Println("[request]", r.URL)
	synChan <- 1
	defer func() { <-synChan }()

	record := new(recordmanager.Record)
	msg := "success"
	result := 0
	costType := 0
	isDouble := 0
	userName := ""
	enemyName := ""
	scores := uint(0)
	enemyScores := uint(0)
	isWin := 0

	defer func() {
		responseStr := fmt.Sprintf("{\"result\":%d, \"costType\":%d, \"isDouble\":%d, \"userName\":\"%s\", \"enemyName\":\"%s\", \"scores\":%d, \"enemyScores\":%d, \"isWin\":%d, \"msg\":\"%s\"}",
			result, costType, isDouble, userName, enemyName, scores, enemyScores, isWin, msg)
		log.Println("[response]", responseStr)
		w.Write([]byte(responseStr))
	}()

	config, e := configmanager.GetInstance().GetConfig("config/const.csv")
	if e != nil {
		msg = e.Error()
		log.Println("[error]", msg)
		return
	}

	token := r.PostFormValue("token")
	if token == "" {
		msg = "token invalid"
		log.Println("[error]", msg)
		return
	}

	log.Println("[Info]token:", token)

	userName = tokenMap[token]
	if userName == "" {
		msg = "token not found user"
		log.Println("[error]", msg)
		return
	}

	record.UserName = userName

	user, err := usermanager.GetInstance().GetUser(userName)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	record.Uuid = user.Uuid

	enemyInfo := enemyMap[user.Uuid]
	if enemyInfo == nil {
		msg = "user not find enemy"
		log.Println("[error]", msg)
		return
	}

	value, err := strconv.ParseInt(r.PostFormValue("costType"), 10, 32)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	costType = int(value)
	record.Type = uint(value)

	value, err = strconv.ParseInt(r.PostFormValue("roleId"), 10, 32)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	record.RoleId = uint(value)

	value, err = strconv.ParseInt(r.PostFormValue("petId"), 10, 32)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	record.PetId = uint(value)

	value, err = strconv.ParseInt(r.PostFormValue("equipmentId"), 10, 32)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	record.EquipmentId = uint(value)

	value, err = strconv.ParseInt(r.PostFormValue("weaponId"), 10, 32)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	record.WeaponId = uint(value)

	value, err = strconv.ParseInt(r.PostFormValue("mountId"), 10, 32)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	record.MountId = uint(value)

	totalScoresStr := r.PostFormValue("totalScores")
	value, err = strconv.ParseInt(totalScoresStr, 10, 32)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	record.Scores = uint(value)
	scores = record.Scores

	record.Records = r.PostFormValue("records")
	log.Println("[Info]records:", record.Records)

	recordSum := r.PostFormValue("recordSum")
	log.Println("[Info]recordSum:", recordSum)

	sumStr := totalScoresStr + record.Records + user.PasswordSum

	sumValue := md5.Sum([]byte(sumStr))
	sumHexStr := ""

	for _, b := range sumValue {
		sumHexStr += fmt.Sprintf("%.2x", b)
	}

	if sumHexStr != recordSum {
		msg = "sum not match"
		log.Println("sumHexStr:" + sumHexStr)
		log.Println("[error]", msg)
		return
	}

	record.TotalDay = getTotalDay()

	err = recordmanager.GetInstance().AddRecord(record)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	isDouble = int(enemyInfo.isDouble)
	enemyName = enemyInfo.name
	enemyScores = enemyInfo.scores
	if record.Scores > enemyInfo.scores {
		isWin = 1
	}
	goldUpLimit := config["GoldUpLimit"]["value"].Uint(1)
	diamondUpLimit := config["DiamondUpLimit"]["value"].Uint(1)

	if isWin != 0 {
		switch costType {
		case goldCostType:
			user.GoldWinAmount++
			if user.GoldWinAmount >= goldUpLimit {
				if user.FixLevel < 0 {
					user.FixLevel = 0
				}

				user.FixLevel++
			}
		case diamondCostType:
			user.DiamondWinAmount++
			if user.DiamondWinAmount >= diamondUpLimit {
				if user.DiamondFixLevel < 0 {
					user.DiamondFixLevel = 0
				}

				user.DiamondFixLevel++
			}
		}

	} else {
		switch costType {
		case goldCostType:
			user.GoldLoseAmount++
			if user.GoldWinAmount < goldUpLimit {
				rate := uint32(config["GoldLessRate"]["value"].Uint(0))
				log.Println("GoldLessRate:", rate)
				if rate > rand.Uint32()%100 {
					user.FixLevel--
				}
			}
		case diamondCostType:
			user.DiamondLoseAmount++
			if user.DiamondWinAmount < diamondUpLimit {
				rate := uint32(config["DiamondLessRate"]["value"].Uint(0))
				log.Println("DiamondLessRate:", rate)
				if rate > rand.Uint32()%100 {
					user.DiamondFixLevel--
				}
			}
		}
	}

	usermanager.GetInstance().MarkUserChange(user.UserName)

	result = 1

	delete(enemyMap, user.Uuid)
}

func handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	log.Println("[request]", r.URL)
	synChan <- 1
	defer func() { <-synChan }()

	msg := "success"
	result := 0

	defer func() {
		responseStr := fmt.Sprintf("{\"result\":%d, \"msg\":\"%s\"}", result, msg)
		log.Println("[response]", responseStr)
		w.Write([]byte(responseStr))
	}()

	key := r.FormValue("key")
	if key != "UKing888" {
		msg = "key not match"
		log.Println("[error]", msg)
		return
	}

	result = 1
	configmanager.GetInstance().Clear()
}

func handleGetCloudSaveFile(w http.ResponseWriter, r *http.Request) {
	log.Println("[request]", r.URL)
	synChan <- 1
	defer func() { <-synChan }()

	msg := "success"
	result := 0
	data := ""

	defer func() {
		responseStr := fmt.Sprintf("{\"result\":%d, \"data\":\"%s\", \"msg\":\"%s\"}", result, data, msg)
		log.Println("[response]", responseStr)
		w.Write([]byte(responseStr))
	}()

	token := r.FormValue("tpToken")
	if !isStringValid(token) {
		msg = "tp token invalid:" + token
		log.Println("[error]", msg)
		return
	}

	request := fmt.Sprintf("https://openapi.360.cn/user/me.json?access_token=%s&fields=id", token)
	log.Println("[info]", request)
	resp, err := http.Get(request)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	jsonInfo := new(JsonInfo)
	json.Unmarshal(body, jsonInfo)

	id := jsonInfo.Id
	if jsonInfo.Error_Code != "" {
		msg = jsonInfo.Error
		log.Println("[error]", msg)
		return
	}

	fileInfo, err := ioutil.ReadFile("saveFiles/" + id)
	if err == nil {
		data = base64.StdEncoding.EncodeToString(fileInfo)
		if err != nil {
			msg = err.Error()
			log.Println("[error]", msg)
			return
		}
	}

	result = 1
}

func handlePayCallback(w http.ResponseWriter, r *http.Request) {
	log.Println("[request]", r.URL)
	synChan <- 1
	defer func() { <-synChan }()

	responseStr := ""
	defer func() {
		w.Write([]byte(responseStr))
	}()

	orderStr := r.FormValue("order_id")
	if !isStringValid(orderStr) {
		responseStr = "order id invalid"
		log.Println("[error]", responseStr)
		return
	}

	if !orderMap[orderStr] {
		orderMap[orderStr] = true
		responseStr = "ok"
	} else {
		responseStr = "order id already recv"
	}
}

func handleUploadSaveFile(w http.ResponseWriter, r *http.Request) {
	log.Println("[request]", r.URL)
	synChan <- 1
	defer func() { <-synChan }()

	msg := "success"
	result := 0

	defer func() {
		responseStr := fmt.Sprintf("{\"result\":%d,\"msg\":\"%s\"}", result, msg)
		log.Println("[response]", responseStr)
		w.Write([]byte(responseStr))
	}()

	token := r.PostFormValue("tpToken")
	if !isStringValid(token) {
		msg = "tp token invalid:" + token
		log.Println("[error]", msg)
		return
	}

	dataStr := r.PostFormValue("data")
	dataStr = strings.Replace(dataStr, " ", "+", -1)
	if !isStringValid(dataStr) {
		msg = "data invalid:" + dataStr
		log.Println("[error]", msg)
		return
	}

	request := fmt.Sprintf("https://openapi.360.cn/user/me.json?access_token=%s&fields=id", token)
	log.Println("[info]", request)
	resp, err := http.Get(request)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	jsonInfo := new(JsonInfo)
	json.Unmarshal([]byte(body), jsonInfo)

	id := jsonInfo.Id
	if jsonInfo.Error_Code != "" {
		msg = jsonInfo.Error
		log.Println("[error]", msg)
		return
	}

	log.Println("[info]", "dataStr:", dataStr)
	data, err := base64.StdEncoding.DecodeString(dataStr)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	err = ioutil.WriteFile("saveFiles/"+id, data, os.ModePerm)
	if err != nil {
		msg = err.Error()
		log.Println("[error]", msg)
		return
	}

	result = 1
}

func Init(synChannel chan int) {
	synChan = synChannel
	tokenMap = make(map[string]string)
	enemyMap = make(map[uint64]*enemyInfo)
	orderMap = make(map[string]bool)

	http.HandleFunc("/regist", handleRegist)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/getTime", handleGetTime)
	http.HandleFunc("/changeUserName", handleChangeUserName)
	http.HandleFunc("/findEnemy", handleFindEnemy)
	http.HandleFunc("/buyBattleAmount", handleBuyBattleAmount)
	http.HandleFunc("/updateConfig", handleUpdateConfig)
	http.HandleFunc("/getCloudSaveFile", handleGetCloudSaveFile)
	http.HandleFunc("/uploadSaveFile", handleUploadSaveFile)
	http.HandleFunc("/payCallback", handlePayCallback)
	http.HandleFunc("/uploadRecord", handleUploadRecord)

	log.Fatal(http.ListenAndServe(":14000", nil))
}
