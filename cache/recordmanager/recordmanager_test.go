package recordmanager

import (
	"testing"
	"time"
)

func TestLoadData(t *testing.T) {
	recordManager := GetInstance()
	err := recordManager.LoadData()
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}
}

func TestAddRecord(t *testing.T) {
	recordManager := GetInstance()
	err := recordManager.LoadData()
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}

	record := &Record{
		UserName:    "hi",
		RoleId:      1,
		PetId:       2,
		MountId:     4,
		WeapinId:    5,
		EquipmentId: 6,
		Scores:      77777,
		Records:     "{}",
		Uuid:        27,
	}
	recordManager.AddRecord(record)
	recordManager.UpdateToDB()

	time.Sleep(time.Second * 5)
}

func TestGetRecord(t *testing.T) {
	recordManager := GetInstance()
	err := recordManager.LoadData()
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}

	record := recordManager.GetRecord(77777, 0)
	if record == nil {
		t.Log("not found record")
		t.Fail()
		return
	}
}
