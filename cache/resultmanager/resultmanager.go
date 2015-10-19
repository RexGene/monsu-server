package resultmanager

import (
	"github.com/RexGene/sqlproxy"
	"strconv"
)

const (
	defaultEventListSize = 128
	defaultResultSize    = 512
)

var instance *ResultManager

type Result struct {
	Uuid        uint64
	EnemyUuid   uint64
	UserName    string
	EnemyName   string
	Scores      uint
	EnemyScores uint
	RewardType  uint8
	Amount      uint
}

type ResultManager struct {
	cmdEventList []*Result
	sqlProxy     *sqlproxy.SqlProxy
	dataMap      map[string][]*Result
}

func newInstance() *ResultManager {
	return &ResultManager{
		cmdEventList: make([]*Result, 0, defaultEventListSize),
		dataMap:      make(map[string][]*Result),
	}
}

func GetInstance() *ResultManager {
	if instance == nil {
		instance = newInstance()
	}

	return instance
}

func (this *ResultManager) AddResult(cmd *Result) error {
	this.cmdEventList = append(this.cmdEventList, cmd)

	dataMap := this.dataMap
	list := dataMap[cmd.UserName]
	if list == nil {
		list = make([]*Result, 0, defaultResultSize)
	}

	dataMap[cmd.UserName] = append(list, cmd)
	return nil
}

func (this *ResultManager) GetResult(userName string) []*Result {
	return this.dataMap[userName]
}

func (this *ResultManager) UpdateToDB() error {
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
			Name:  "enemy_name",
			Value: recordCmd.EnemyName,
		}

		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "uuid",
			Value: strconv.FormatUint(recordCmd.Uuid, 10),
		}
		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "enemy_uuid",
			Value: strconv.FormatUint(recordCmd.EnemyUuid, 10),
		}
		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "scores",
			Value: strconv.FormatUint(uint64(recordCmd.Scores), 10),
		}
		fields = append(fields, field)

		field = &sqlproxy.FieldData{
			Name:  "enemy_scores",
			Value: strconv.FormatUint(uint64(recordCmd.EnemyScores), 10),
		}
		fields = append(fields, field)

		saveCmd = &sqlproxy.SaveCmd{
			TableName: "result",
			IsNew:     true,
			Condition: nil,
			Fields:    fields[:],
		}

		cmdList <- saveCmd
	}

	if len(this.cmdEventList) != 0 {
		this.cmdEventList = make([]*Result, 0, defaultEventListSize)
	}

	return nil
}

func (this *ResultManager) LoadData() error {
	proxy := sqlproxy.NewSqlProxy("root", "123456", "111.59.24.181", "3306", "game")
	err := proxy.Connect()
	if err != nil {
		return err
	}

	fieldNames := make([]string, 0, 16)
	fieldNames = append(fieldNames, "user_name")
	fieldNames = append(fieldNames, "enemy_name")
	fieldNames = append(fieldNames, "uuid")
	fieldNames = append(fieldNames, "enemy_uuid")
	fieldNames = append(fieldNames, "scores")
	fieldNames = append(fieldNames, "enemy_scores")
	fieldNames = append(fieldNames, "reward_type")
	fieldNames = append(fieldNames, "amount")

	queryCmd := &sqlproxy.QueryCmd{
		TableName:  "result",
		FieldNames: fieldNames[:],
	}

	dataMapList, err := proxy.LoadData(queryCmd)
	if err != nil {
		return err
	}

	for _, dataMap := range dataMapList {
		result := &Result{
			UserName:  dataMap["user_name"],
			EnemyName: dataMap["enemy_name"],
		}

		var value int
		value, err = strconv.Atoi(dataMap["uuid"])
		if err != nil {
			return err
		}
		result.Uuid = uint64(value)

		value, err = strconv.Atoi(dataMap["enemy_uuid"])
		if err != nil {
			return err
		}
		result.EnemyUuid = uint64(value)

		value, err = strconv.Atoi(dataMap["scores"])
		if err != nil {
			return err
		}
		result.Scores = uint(value)

		value, err = strconv.Atoi(dataMap["enemy_scores"])
		if err != nil {
			return err
		}
		result.EnemyScores = uint(value)

		value, err = strconv.Atoi(dataMap["reward_type"])
		if err != nil {
			return err
		}
		result.RewardType = uint8(value)

		value, err = strconv.Atoi(dataMap["amount"])
		if err != nil {
			return err
		}
		result.Amount = uint(value)

		dataMap := this.dataMap
		list := dataMap[result.UserName]
		if list == nil {
			list = make([]*Result, 0, defaultResultSize)
		}

		dataMap[result.UserName] = append(list, result)
	}

	this.sqlProxy = proxy
	return nil
}
