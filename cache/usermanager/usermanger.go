package usermanager

import (
	"errors"
	"github.com/RexGene/sqlproxy"
	"strconv"
	"time"
)

type User struct {
	UserName                 string
	PasswordSum              string
	Uuid                     uint64
	MacAddr                  string
	LastUpdateDay            uint
	GoldCount                uint8
	DiamondCount             uint8
	GoldRank                 uint
	GoldWinAmount            uint
	GoldLoseAmount           uint
	DiamondRank              uint
	DiamondWinAmount         uint
	DiamondLoseAmount        uint
	GoldAvailableBuyCount    uint
	DiamondAvailableBuyCount uint
	IsNew                    bool
	Token                    string
	FixLevel                 int
}

const (
	DefaultGoldCount      = 5
	DefaultDiamondCount   = 5
	defaultUpdateUserSize = 1024
)

type UserManager struct {
	userMap        map[string]*User
	userUuidMap    map[uint64]*User
	maxUserId      uint64
	updateUserList []*User
	sqlProxy       *sqlproxy.SqlProxy
}

var instance *UserManager

func newInstance() *UserManager {
	return &UserManager{
		maxUserId:      0,
		updateUserList: make([]*User, 0, defaultUpdateUserSize),
		userMap:        make(map[string]*User),
		userUuidMap:    make(map[uint64]*User),
	}
}

func GetInstance() *UserManager {
	if instance == nil {
		instance = newInstance()
	}

	return instance
}

func (this *UserManager) ChangeName(uuid uint64, userName string) error {
	user := this.userUuidMap[uuid]
	if user == nil {
		return errors.New("user not found:" + strconv.FormatUint(uuid, 10))
	}

	oldName := user.UserName
	user.UserName = userName

	delete(this.userMap, oldName)
	this.userMap[userName] = user

	return nil
}

func (this *UserManager) GetUserByUuid(uuid uint64) (*User, error) {
	user := this.userUuidMap[uuid]
	if user == nil {
		return user, errors.New("user not found:" + strconv.FormatUint(uuid, 10))
	}

	return user, nil
}

func (this *UserManager) GetUser(userName string) (*User, error) {
	user := this.userMap[userName]
	if user == nil {
		return user, errors.New("user not found:" + userName)
	}

	return user, nil
}

func (this *UserManager) MarkUserChange(userName string) error {
	user := this.userMap[userName]
	if user == nil {
		return errors.New("user not found:" + userName)
	}

	this.updateUserList = append(this.updateUserList, user)
	return nil
}

func (this *UserManager) UpdateUserToDB() {
	cmdList := this.sqlProxy.GetSaveCmdList()
	var saveCmd *sqlproxy.SaveCmd

	for _, user := range this.updateUserList {
		condition := &sqlproxy.FieldData{
			Name:  "uuid",
			Value: strconv.FormatUint(uint64(user.Uuid), 10),
		}

		fields := make([]*sqlproxy.FieldData, 0, 32)

		field := &sqlproxy.FieldData{
			Name:  "user_name",
			Value: user.UserName,
		}
		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "password",
			Value: user.PasswordSum,
		}
		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "uuid",
			Value: strconv.FormatUint(user.Uuid, 10),
		}
		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "mac_addr",
			Value: user.MacAddr,
		}
		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "gold_count",
			Value: strconv.FormatUint(uint64(user.GoldCount), 10),
		}
		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "diamond_count",
			Value: strconv.FormatUint(uint64(user.DiamondCount), 10),
		}
		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "diamond_available_buy_count",
			Value: strconv.FormatUint(uint64(user.DiamondAvailableBuyCount), 10),
		}
		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "gold_available_buy_count",
			Value: strconv.FormatUint(uint64(user.GoldAvailableBuyCount), 10),
		}
		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "last_update_day",
			Value: strconv.FormatUint(uint64(user.LastUpdateDay), 10),
		}
		fields = append(fields, field)

		saveCmd = &sqlproxy.SaveCmd{
			TableName: "users",
			IsNew:     user.IsNew,
			Condition: condition,
			Fields:    fields[:],
		}

		cmdList <- saveCmd
		user.IsNew = false
	}

	if len(this.updateUserList) != 0 {
		this.updateUserList = make([]*User, 0, defaultUpdateUserSize)
	}
}

func (this *UserManager) AddUser(userName string, passwordSum string, macAddr string) error {
	userMap := this.userMap
	if userMap[userName] != nil {
		return errors.New("user already regist")
	}

	this.maxUserId++
	newUser := &User{
		UserName:      userName,
		PasswordSum:   passwordSum,
		MacAddr:       macAddr,
		GoldCount:     DefaultGoldCount,
		DiamondCount:  DefaultDiamondCount,
		LastUpdateDay: uint((time.Now().Unix() - (3600 * 8)) / 86400),
		Uuid:          this.maxUserId,
		FixLevel:      0,
		IsNew:         true,
	}

	userMap[userName] = newUser
	this.userUuidMap[this.maxUserId] = newUser
	this.updateUserList = append(this.updateUserList, newUser)

	return nil
}

func (this *UserManager) GetTotalUser() int {
	return len(this.userMap)
}

func (this *UserManager) LoadUser() error {
	proxy := sqlproxy.NewSqlProxy("root", "Uking1881982050~!@", "123.59.24.181", "3306", "game")
	err := proxy.Connect()
	if err != nil {
		return err
	}

	fieldNames := make([]string, 0, 16)
	fieldNames = append(fieldNames, "user_name")
	fieldNames = append(fieldNames, "password")
	fieldNames = append(fieldNames, "uuid")
	fieldNames = append(fieldNames, "mac_addr")
	fieldNames = append(fieldNames, "last_update_day")
	fieldNames = append(fieldNames, "gold_count")
	fieldNames = append(fieldNames, "diamond_count")

	fieldNames = append(fieldNames, "gold_rank")
	fieldNames = append(fieldNames, "gold_win_amount")
	fieldNames = append(fieldNames, "gold_lose_amount")
	fieldNames = append(fieldNames, "diamond_rank")
	fieldNames = append(fieldNames, "diamond_win_amount")
	fieldNames = append(fieldNames, "diamond_lose_amount")

	fieldNames = append(fieldNames, "diamond_available_buy_count")
	fieldNames = append(fieldNames, "gold_available_buy_count")

	queryCmd := &sqlproxy.QueryCmd{
		TableName:  "users",
		FieldNames: fieldNames[:],
	}

	dataMapList, err := proxy.LoadData(queryCmd)
	if err != nil {
		return err
	}

	var maxUuid uint64 = 0
	for _, dataMap := range dataMapList {
		newUser := new(User)

		newUser.UserName = dataMap["user_name"]
		newUser.PasswordSum = dataMap["password"]
		newUser.MacAddr = dataMap["mac_addr"]

		//last update time
		temp, err := strconv.ParseUint(dataMap["last_update_day"], 10, 32)
		if err != nil {
			this.userMap = make(map[string]*User)
			this.userUuidMap = make(map[uint64]*User)
			return err
		}

		newUser.LastUpdateDay = uint(temp)

		//gold count
		temp, err = strconv.ParseUint(dataMap["gold_count"], 10, 8)
		if err != nil {
			this.userMap = make(map[string]*User)
			this.userUuidMap = make(map[uint64]*User)
			return err
		}

		newUser.GoldCount = uint8(temp)

		//diamond count
		temp, err = strconv.ParseUint(dataMap["diamond_count"], 10, 8)
		if err != nil {
			this.userMap = make(map[string]*User)
			this.userUuidMap = make(map[uint64]*User)
			return err
		}

		newUser.DiamondCount = uint8(temp)

		temp, err = strconv.ParseUint(dataMap["gold_rank"], 10, 32)
		if err != nil {
			this.userMap = make(map[string]*User)
			this.userUuidMap = make(map[uint64]*User)
			return err
		}

		newUser.GoldRank = uint(temp)

		temp, err = strconv.ParseUint(dataMap["gold_win_amount"], 10, 32)
		if err != nil {
			this.userMap = make(map[string]*User)
			this.userUuidMap = make(map[uint64]*User)
			return err
		}

		newUser.GoldWinAmount = uint(temp)

		temp, err = strconv.ParseUint(dataMap["gold_lose_amount"], 10, 32)
		if err != nil {
			this.userMap = make(map[string]*User)
			this.userUuidMap = make(map[uint64]*User)
			return err
		}

		newUser.GoldLoseAmount = uint(temp)

		temp, err = strconv.ParseUint(dataMap["diamond_rank"], 10, 32)
		if err != nil {
			this.userMap = make(map[string]*User)
			this.userUuidMap = make(map[uint64]*User)
			return err
		}

		newUser.DiamondRank = uint(temp)

		temp, err = strconv.ParseUint(dataMap["diamond_win_amount"], 10, 32)
		if err != nil {
			this.userMap = make(map[string]*User)
			this.userUuidMap = make(map[uint64]*User)
			return err
		}

		newUser.DiamondWinAmount = uint(temp)

		temp, err = strconv.ParseUint(dataMap["diamond_lose_amount"], 10, 32)
		if err != nil {
			this.userMap = make(map[string]*User)
			this.userUuidMap = make(map[uint64]*User)
			return err
		}

		newUser.DiamondLoseAmount = uint(temp)

		temp, err = strconv.ParseUint(dataMap["gold_available_buy_count"], 10, 32)
		if err != nil {
			this.userMap = make(map[string]*User)
			this.userUuidMap = make(map[uint64]*User)
			return err
		}

		newUser.GoldAvailableBuyCount = uint(temp)

		temp, err = strconv.ParseUint(dataMap["diamond_available_buy_count"], 10, 32)
		if err != nil {
			this.userMap = make(map[string]*User)
			this.userUuidMap = make(map[uint64]*User)
			return err
		}

		newUser.DiamondAvailableBuyCount = uint(temp)

		//uuid
		uuid, err := strconv.ParseUint(dataMap["uuid"], 10, 64)
		if err != nil {
			this.userMap = make(map[string]*User)
			this.userUuidMap = make(map[uint64]*User)
			return err
		}

		if maxUuid < uuid {
			maxUuid = uuid
		}

		newUser.Uuid = uuid
		newUser.FixLevel = 0
		this.userMap[newUser.UserName] = newUser
		this.userUuidMap[uuid] = newUser
	}

	this.maxUserId = maxUuid
	this.sqlProxy = proxy
	return nil
}
