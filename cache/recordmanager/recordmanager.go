package recordmanager

import (
	"../configmanager"
	"errors"
	"github.com/RexGene/sqlproxy"
	"log"
	"math/rand"
	"strconv"
)

var (
	ErrUserNotFound = errors.New("not found user")
)

const (
	defaultEventListSize  = 128
	defaultZoneRecordSize = 512
	defaultUserRecordSize = 512
)

const (
	goldType    = 1
	diamondType = 2
)

var instance *RecordManager

type Record struct {
	UserName    string
	RoleId      uint
	PetId       uint
	MountId     uint
	WeaponId    uint
	EquipmentId uint
	Scores      uint
	Records     string
	Uuid        uint64
	Type        uint
	TotalDay    int64
}

type RecordSlice []*Record

func (this RecordSlice) Len() int {
	return len(this)
}

func (this RecordSlice) Less(i, j int) bool {
	return this[i].Scores < this[j].Scores
}

func (this RecordSlice) Swap(i, j int) {
	temp := this[i]
	this[i] = this[j]
	this[j] = temp
}

type RecordManager struct {
	cmdEventList      []*Record
	sqlProxy          *sqlproxy.SqlProxy
	zoneRecords       map[uint][]*Record
	diamonZoneRecords map[uint][]*Record
	userRecords       map[uint64][]*Record
	diamonUserRecords map[uint64][]*Record
	maxLevel          int
	levelMap          map[int][]uint
}

func newInstance() *RecordManager {
	return &RecordManager{
		cmdEventList:      make([]*Record, 0, defaultEventListSize),
		zoneRecords:       make(map[uint][]*Record),
		diamonZoneRecords: make(map[uint][]*Record),
		userRecords:       make(map[uint64][]*Record),
		diamonUserRecords: make(map[uint64][]*Record),
		levelMap:          make(map[int][]uint),
		maxLevel:          0,
	}
}

func (this *RecordManager) initZoneConfigData() error {
	zoneConfig, err := configmanager.GetInstance().GetConfig("config/zone.csv")
	if err != nil {
		return err
	}

	config, err := configmanager.GetInstance().GetConfig("config/const.csv")
	if err != nil {
		return err
	}

	zoneLen := config["ZoneRange"]["value"].Uint(1)

	maxLevel := 0
	for index, value := range zoneConfig {
		indexValue, err := strconv.ParseUint(index, 10, 32)
		if err != nil {
			return err
		}

		scores := uint(indexValue) * zoneLen
		level := value["level"].Int(1)
		if maxLevel < level {
			maxLevel = level
		}

		list := this.levelMap[level]
		if list == nil {
			list = make([]uint, 0, 32)
			this.levelMap[level] = list
		}

		this.levelMap[level] = append(list, scores)
	}

	this.maxLevel = maxLevel

	return nil
}

func (this *RecordManager) GetScoresByLevel(level int) (uint, error) {
	list := this.levelMap[level]
	if list == nil {
		err := errors.New("level not found:" + strconv.FormatInt(int64(level), 10))
		log.Println("[Error]", err)
		return 0, err
	}

	listLen := len(list)
	if listLen == 0 {
		err := errors.New("list is empty")
		log.Println("[Error]", err)
		return 0, err
	}

	return list[rand.Int()%len(list)], nil

}

func GetInstance() *RecordManager {
	if instance == nil {
		instance = newInstance()
		err := instance.initZoneConfigData()
		if err != nil {
			log.Fatal(err)
		}
	}

	return instance
}

func shuffle(list []*Record) {
	size := len(list)
	if size == 0 {
		return
	}

	index := rand.Int() % size
	temp := list[0]
	list[0] = list[index]
	list[index] = temp

	shuffle(list[1:])
}

func (this *RecordManager) GetUserRecords(uuid uint64, t int) ([]*Record, error) {
	var recordsMap map[uint64][]*Record
	switch t {
	case goldType:
		recordsMap = this.userRecords
	case diamondType:
		recordsMap = this.diamonUserRecords
	default:
		return nil, errors.New("type invalid:" + strconv.FormatInt(int64(t), 10))
	}

	result := recordsMap[uuid]
	if result == nil {
		return nil, nil
	}

	return result, nil
}

func (this *RecordManager) AddRecord(cmd *Record) error {
	config, err := configmanager.GetInstance().GetConfig("config/const.csv")
	if err != nil {
		return err
	}

	zoneConfig, err := configmanager.GetInstance().GetConfig("config/zone.csv")
	if err != nil {
		return err
	}

	if len(zoneConfig) == 0 {
		return errors.New("zone config data empty")
	}

	this.cmdEventList = append(this.cmdEventList, cmd)
	zoneLen := config["ZoneRange"]["value"].Uint(1)
	zoneMax := uint(len(zoneConfig) - 1)
	i := cmd.Scores / zoneLen
	if i > zoneMax {
		i = zoneMax
	}
	key := strconv.FormatUint(uint64(i), 10)
	index := zoneConfig[key]["level"].Uint(0)

	log.Println("add record index:", index, " type:", cmd.Type)

	var zoneRecords map[uint][]*Record
	var userRecords map[uint64][]*Record

	if cmd.Type == goldType {
		zoneRecords = this.zoneRecords
		userRecords = this.userRecords
	} else if cmd.Type == diamondType {
		zoneRecords = this.diamonZoneRecords
		userRecords = this.diamonUserRecords
	} else {
		return errors.New("unknow type:" + strconv.FormatUint(uint64(cmd.Type), 10))
	}

	if zoneRecords[index] == nil {
		zoneRecords[index] = make([]*Record, 0, defaultZoneRecordSize)
	}
	zoneRecords[index] = append(zoneRecords[index], cmd)

	uuid := cmd.Uuid
	if userRecords[uuid] == nil {
		userRecords[uuid] = make([]*Record, 0, defaultUserRecordSize)
	}
	userRecords[uuid] = append(userRecords[uuid], cmd)

	return nil
}

func (this *RecordManager) GetIndex(scores uint, fix int) (int, error) {
	configManager := configmanager.GetInstance()
	config, err := configManager.GetConfig("config/const.csv")
	if err != nil {
		return 0, err
	}

	zoneConfig, err := configManager.GetConfig("config/zone.csv")
	if err != nil {
		return 0, err
	}

	if len(zoneConfig) == 0 {
		return 0, errors.New("zone config data empty")
	}

	zoneRangeConfig, err := configManager.GetConfig("config/zoneRange.csv")
	if err != nil {
		return 0, err
	}

	if len(zoneRangeConfig) == 0 {
		return 0, errors.New("zone range config empty")
	}

	zoneLen := config["ZoneRange"]["value"].Uint(1)
	zoneMax := uint(len(zoneConfig) - 1)
	i := scores / zoneLen
	if i > zoneMax {
		i = zoneMax
	}

	key := strconv.FormatUint(uint64(i), 10)
	index := zoneConfig[key]["level"].Int(1)
	if index == 0 {
		index = 1
	}

	if fix < 0 {
		maxLevel := this.maxLevel
		valueKey := strconv.FormatUint(uint64(-fix), 10)
		rangeConfig := zoneRangeConfig[valueKey]
		beginIndex := rangeConfig["LoseBeginIndex"].Int(1)
		indexRange := rangeConfig["LoseIndexRange"].Int(1)

		value := beginIndex + rand.Int()%indexRange
		log.Println("[Info]down random base:", beginIndex, " random range:", indexRange)

		index += value
		if index < 1 {
			index = 1
		} else if index > maxLevel {
			index = maxLevel
		}

	} else if fix > 0 {
		maxLevel := config["MaxLevel"]["value"].Int(1)

		valueKey := strconv.FormatUint(uint64(fix), 10)
		rangeConfig := zoneRangeConfig[valueKey]
		beginIndex := rangeConfig["WinBeginIndex"].Int(1)
		indexRange := rangeConfig["WinIndexRange"].Int(1)

		value := beginIndex + rand.Int()%indexRange

		log.Println("[Info]up random base:", beginIndex, " random range:", indexRange)
		index += value

		if index < 1 {
			index = 1
		} else if index > maxLevel {
			index = maxLevel
		}
	}

	return index, nil
}

func (this *RecordManager) GetRecord(scores uint, fix int, t int, uuid uint64) (*Record, error) {
	index, err := this.GetIndex(scores, fix)
	if err != nil {
		return nil, err
	}

	var zoneRecords map[uint][]*Record
	if t == goldType {
		zoneRecords = this.zoneRecords
	} else if t == diamondType {
		zoneRecords = this.diamonZoneRecords
	} else {
		return nil, errors.New("unknow type:" + strconv.FormatInt(int64(t), 10))
	}

	list := zoneRecords[uint(index)]
	log.Println("read record index:", index, " type:", t)

	if list == nil {
		return nil, ErrUserNotFound
	} else {
		shuffle(list)
		for _, r := range list {
			if r.Uuid != uuid {
				return r, nil
			}
		}

		return nil, ErrUserNotFound
	}
}

func (this *RecordManager) UpdateToDB() error {
	cmdList := this.sqlProxy.GetSaveCmdList()
	var saveCmd *sqlproxy.SaveCmd

	for _, recordCmd := range this.cmdEventList {
		fields := make([]*sqlproxy.FieldData, 0, 32)

		field := &sqlproxy.FieldData{
			Name:  "user_name",
			Value: recordCmd.UserName,
		}
		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "role_id",
			Value: strconv.FormatUint(uint64(recordCmd.RoleId), 10),
		}
		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "pet_id",
			Value: strconv.FormatUint(uint64(recordCmd.PetId), 10),
		}
		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "mount_id",
			Value: strconv.FormatUint(uint64(recordCmd.MountId), 10),
		}
		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "weapon_id",
			Value: strconv.FormatUint(uint64(recordCmd.WeaponId), 10),
		}
		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "equipment_id",
			Value: strconv.FormatUint(uint64(recordCmd.EquipmentId), 10),
		}
		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "scores",
			Value: strconv.FormatUint(uint64(recordCmd.Scores), 10),
		}
		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "uuid",
			Value: strconv.FormatUint(recordCmd.Uuid, 10),
		}
		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "type",
			Value: strconv.FormatUint(uint64(recordCmd.Type), 10),
		}
		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "records",
			Value: recordCmd.Records,
		}
		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "total_day",
			Value: strconv.FormatInt(recordCmd.TotalDay, 10),
		}
		fields = append(fields, field)

		saveCmd = &sqlproxy.SaveCmd{
			TableName: "record",
			IsNew:     true,
			Condition: nil,
			Fields:    fields[:],
		}

		cmdList <- saveCmd
	}

	if len(this.cmdEventList) != 0 {
		this.cmdEventList = make([]*Record, 0, defaultEventListSize)
	}

	return nil
}

func (this *RecordManager) LoadData() error {
	proxy := sqlproxy.NewSqlProxy("root", "123456", "111.59.24.181", "3306", "game")
	err := proxy.Connect()
	if err != nil {
		return err
	}

	config, err := configmanager.GetInstance().GetConfig("config/const.csv")
	if err != nil {
		return err
	}

	zoneConfig, err := configmanager.GetInstance().GetConfig("config/zone.csv")
	if err != nil {
		return err
	}

	if len(zoneConfig) == 0 {
		return errors.New("zone config data empty")
	}

	fieldNames := make([]string, 0, 16)
	fieldNames = append(fieldNames, "user_name")
	fieldNames = append(fieldNames, "role_id")
	fieldNames = append(fieldNames, "pet_id")
	fieldNames = append(fieldNames, "mount_id")
	fieldNames = append(fieldNames, "weapon_id")
	fieldNames = append(fieldNames, "equipment_id")
	fieldNames = append(fieldNames, "uuid")
	fieldNames = append(fieldNames, "scores")
	fieldNames = append(fieldNames, "records")
	fieldNames = append(fieldNames, "total_day")
	fieldNames = append(fieldNames, "type")

	queryCmd := &sqlproxy.QueryCmd{
		TableName:  "record",
		FieldNames: fieldNames[:],
	}

	dataMapList, err := proxy.LoadData(queryCmd)
	if err != nil {
		return err
	}

	for _, dataMap := range dataMapList {
		record := &Record{
			UserName: dataMap["user_name"],
			Records:  dataMap["records"],
		}

		var value int
		value, err = strconv.Atoi(dataMap["role_id"])
		if err != nil {
			return err
		}
		record.RoleId = uint(value)

		v, err := strconv.ParseInt(dataMap["total_day"], 10, 32)
		if err != nil {
			return err
		}
		record.TotalDay = v

		value, err = strconv.Atoi(dataMap["pet_id"])
		if err != nil {
			return err
		}
		record.PetId = uint(value)

		value, err = strconv.Atoi(dataMap["mount_id"])
		if err != nil {
			return err
		}
		record.MountId = uint(value)

		value, err = strconv.Atoi(dataMap["weapon_id"])
		if err != nil {
			return err
		}
		record.WeaponId = uint(value)

		value, err = strconv.Atoi(dataMap["equipment_id"])
		if err != nil {
			return err
		}
		record.EquipmentId = uint(value)

		value_new, err := strconv.ParseUint(dataMap["uuid"], 10, 64)
		if err != nil {
			return err
		}
		record.Uuid = value_new

		value_new, err = strconv.ParseUint(dataMap["type"], 10, 32)
		if err != nil {
			return err
		}
		record.Type = uint(value_new)

		value, err = strconv.Atoi(dataMap["scores"])
		if err != nil {
			return err
		}
		record.Scores = uint(value)

		zoneLen := config["ZoneRange"]["value"].Uint(1)
		zoneMax := uint(len(zoneConfig) - 1)
		i := record.Scores / zoneLen
		if i > zoneMax {
			i = zoneMax
		}
		key := strconv.FormatUint(uint64(i), 10)
		index := zoneConfig[key]["level"].Uint(0)

		var zoneRecords map[uint][]*Record
		var userRecords map[uint64][]*Record

		if record.Type == goldType {
			zoneRecords = this.zoneRecords
			userRecords = this.userRecords
		} else if record.Type == diamondType {
			zoneRecords = this.diamonZoneRecords
			userRecords = this.diamonUserRecords
		} else {
			continue
		}

		if zoneRecords[index] == nil {
			zoneRecords[index] = make([]*Record, 0, defaultZoneRecordSize)
		}

		zoneRecords[index] = append(zoneRecords[index], record)

		uuid := record.Uuid
		if userRecords[uuid] == nil {
			userRecords[uuid] = make([]*Record, 0, defaultUserRecordSize)
		}
		userRecords[uuid] = append(userRecords[uuid], record)
	}

	this.sqlProxy = proxy
	return nil
}
