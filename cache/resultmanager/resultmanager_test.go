package resultmanager

import (
	"testing"
	"time"
)

func TestLoadData(t *testing.T) {
	resultManager := GetInstance()
	err := resultManager.LoadData()
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}
}

func TestAddResult(t *testing.T) {
	resultManager := GetInstance()
	err := resultManager.LoadData()
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}

	result := &Result{
		Uuid:        1,
		EnemyUuid:   2,
		UserName:    "RexGene",
		EnemyName:   "Enemy",
		Scores:      99999,
		EnemyScores: 1111,
	}

	resultManager.AddResult(result)
	resultManager.UpdateToDB()

	time.Sleep(time.Second * 10)
}

func TestGetResult(t *testing.T) {
	resultManager := GetInstance()
	err := resultManager.LoadData()
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}

	list := resultManager.GetResult("RexGene")
	if list == nil {
		t.Log("user not found")
		t.Fail()
		return
	}

	for _, result := range list {
		println("userName:", result.UserName)
		println("scores:", result.Scores)
	}
}
