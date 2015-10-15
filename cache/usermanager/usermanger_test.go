package usermanager

import (
	"strconv"
	"testing"
	"time"
)

func TestLoadData(t *testing.T) {
	userManager := GetInstance()
	err := userManager.LoadUser()
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}

	println("total user:" + strconv.Itoa(userManager.GetTotalUser()))
	_, err = userManager.GetUser("rex")
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}

}

func TestAddUser(t *testing.T) {
	userManager := GetInstance()
	err := userManager.LoadUser()
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}

	err = userManager.AddUser("testUser", "888719012", "00000")
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}

	err = userManager.AddUser("testUser2", "111111", "00000")
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}

	userManager.UpdateUserToDB()
	time.Sleep(time.Second * 10)
}

func TestUpdateUser(t *testing.T) {
	userManager := GetInstance()
	err := userManager.LoadUser()
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}

	user, err := userManager.GetUser("rex")
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}

	user.PasswordSum = "newPsw"
	userManager.MarkUserChange(user.UserName)
	userManager.UpdateUserToDB()

	time.Sleep(time.Second * 10)
}
