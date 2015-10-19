package recordmanager

import (
	"github.com/RexGene/sqlproxy"
	"math/rand"
	"strconv"
)

const (
	defaultEventListSize  = 128
	defaultZoneSize       = 1024
	defaultZoneRecordSize = 512
)

var instance *RecordManager

type Record struct {
	UserName    string
	RoleId      uint
	PetId       uint
	MountId     uint
	WeapinId    uint
	EquipmentId uint
	Scores      uint
	Records     string
	Uuid        uint64
}

type RecordManager struct {
	cmdEventList []*Record
	sqlProxy     *sqlproxy.SqlProxy
	zoneRecords  map[uint][]*Record
}

func newInstance() *RecordManager {
	return &RecordManager{
		cmdEventList: make([]*Record, 0, defaultEventListSize),
		zoneRecords:  make(map[uint][]*Record),
	}
}

func GetInstance() *RecordManager {
	if instance == nil {
		instance = newInstance()
	}

	return instance
}

func (this *RecordManager) AddRecord(cmd *Record) error {
	this.cmdEventList = append(this.cmdEventList, cmd)

	zoneLen := uint(defaultZoneSize)
	index := cmd.Scores / zoneLen
	if index >= zoneLen {
		index = zoneLen - 1
	}

	zoneRecords := this.zoneRecords
	if zoneRecords[index] == nil {
		zoneRecords[index] = make([]*Record, 0, defaultZoneRecordSize)
	}

	zoneRecords[index] = append(zoneRecords[index], cmd)
	return nil
}

func (this *RecordManager) GetRecord(scores uint, fix int) *Record {
	zoneLen := uint(defaultZoneSize)
	index := scores / zoneLen

	if fix < 0 {
		value := uint(-fix)
		if index >= value {
			index -= value
		} else {
			index = 0
		}

	} else if fix > 0 {
		value := uint(fix)
		index += value
	}

	if index >= zoneLen {
		index = zoneLen - 1
	}

	list := this.zoneRecords[index]
	if list == nil {
		return nil
	} else {
		return list[rand.Int()%len(list)]
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
			Value: strconv.FormatUint(uint64(recordCmd.WeapinId), 10),
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
			Name:  "Records",
			Value: recordCmd.Records,
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
	proxy := sqlproxy.NewSqlProxy("root", "Uking1881982050~!@", "123.59.24.181", "3306", "game")
	err := proxy.Connect()
	if err != nil {
		return err
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
		record.WeapinId = uint(value)

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

		value, err = strconv.Atoi(dataMap["scores"])
		if err != nil {
			return err
		}
		record.Scores = uint(value)

		zoneLen := uint(defaultZoneSize)
		index := record.Scores / zoneLen
		if index >= zoneLen {
			index = zoneLen - 1
		}

		zoneRecords := this.zoneRecords
		if zoneRecords[index] == nil {
			zoneRecords[index] = make([]*Record, 0, defaultZoneRecordSize)
		}

		zoneRecords[index] = append(zoneRecords[index], record)
	}

	this.sqlProxy = proxy
	return nil
}
