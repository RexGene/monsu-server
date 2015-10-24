package configmanager

import (
	"testing"
)

func TestReadConfig(t *testing.T) {
	dataMap, err := GetInstance().GetConfig("test.csv")
	if err != nil {
		t.Log(err)
		t.Fatal()
		return
	}

	for key, fields := range dataMap {
		println(key)
		for fieldName, value := range fields {
			println(fieldName, value)
		}
	}

}
