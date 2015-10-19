package usermanager

import (
	"errors"
	"fmt"
	"github.com/RexGene/sqlproxy"
	"strconv"
	"time"
)

type User struct {
	UserName       string
	PasswordSum    string
	Uuid           uint64
	MacAddr        string
	Cert           string
	LastUpdateTime time.Time
	GoldCount      uint8
	DiamondCount   uint8
	IsNew          bool
}

const (
	defaultGoldCount      = 5
	defaultDiamondCount   = 5
	defaultUpdateUserSize = 1024
)

type UserManager struct {
	userMap        map[string]*User
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
	}
}

func GetInstance() *UserManager {
	if instance == nil {
		instance = newInstance()
	}

	return instance
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
			Name:  "user_name",
			Value: user.UserName,
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
			Name:  "cert",
			Value: user.Cert,
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

		t := user.LastUpdateTime
		year, month, day := t.Date()
		h := t.Hour()
		m := t.Minute()
		s := t.Second()
		timeStr := fmt.Sprintf("%.4d-%.2d-%.2d %.2d:%.2d:%.2d", year, month, day, h, m, s)
		field = &sqlproxy.FieldData{
			Name:  "last_update_time",
			Value: timeStr,
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
		UserName:       userName,
		PasswordSum:    passwordSum,
		MacAddr:        macAddr,
		GoldCount:      defaultGoldCount,
		DiamondCount:   defaultDiamondCount,
		LastUpdateTime: time.Now(),
		Uuid:           this.maxUserId,
		IsNew:          true,
	}

	userMap[userName] = newUser
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
	fieldNames = append(fieldNames, "last_update_time")
	fieldNames = append(fieldNames, "gold_count")
	fieldNames = append(fieldNames, "diamond_count")
	fieldNames = append(fieldNames, "cert")

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
		newUser.Cert = dataMap["cert"]

		//last update time
		var year, month, day, h, m, s int
		fmt.Sscanf(dataMap["last_update_time"], "%d-%d-%d %d:%d:%d", &year, &month, &day, &h, &m, &s)
		newUser.LastUpdateTime = time.Date(year, time.Month(month), day, h, m, s, 0, time.UTC)

		//gold count
		temp, err := strconv.ParseUint(dataMap["gold_count"], 10, 8)
		if err != nil {
			this.userMap = make(map[string]*User)
			return err
		}

		newUser.GoldCount = uint8(temp)

		//diamond count
		temp, err = strconv.ParseUint(dataMap["diamond_count"], 10, 8)
		if err != nil {
			this.userMap = make(map[string]*User)
			return err
		}

		newUser.DiamondCount = uint8(temp)

		//uuid
		uuid, err := strconv.ParseUint(dataMap["uuid"], 10, 64)
		if err != nil {
			this.userMap = make(map[string]*User)
			return err
		}

		if maxUuid < uuid {
			maxUuid = uuid
		}

		newUser.Uuid = uuid
		this.userMap[newUser.UserName] = newUser
	}

	this.maxUserId = maxUuid
	this.sqlProxy = proxy
	return nil
}
