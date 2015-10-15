package main

import (
	"github.com/RexGene/sqlproxy"
)

func main() {
	db := sqlproxy.NewSqlProxy("root", "123456", "111.59.24.181", "3306", "game")
	err := db.Connect()
	if err != nil {
		panic(err)
	}

}
