package db

import (
	mydb "LookForYou/db/mysql"
	"fmt"
	"log"
)

// UserSignup: 通过用户名及密码完成的用户注册
func UserSignup(username string, passwd string) bool {
	stmt, err := mydb.DBConn().Prepare(
		"insert ignore into tbl_user(`user_name`,`user_pwd`) values (?,?)")
	if err != nil {
		log.Println("Failed to insert, err:" + err.Error())
		return false
	}
	defer stmt.Close()
	ret, err := stmt.Exec(username, passwd)
	if err != nil {
		log.Println("Failed to insert,err:" + err.Error())
		return false
	}
	if rowsAffected, err := ret.RowsAffected(); nil == err && rowsAffected > 0 {
		return true
	}
	return false
}

func UserSignin(username string, encpwd string) bool {
	stmt, err := mydb.DBConn().Prepare(
		"select * from tbl_user where user_name=? limit 1")
	if err != nil {
		log.Println(err.Error())
		return false
	}
	fmt.Println("After prepare ")
	defer stmt.Close()
	rows, err := stmt.Query(username)
	if err != nil {
		log.Println(err.Error())
		return false
	} else if rows == nil {
		log.Println("username not found:" + username)
		return false
	}
	fmt.Println("After query ", rows, err)
	pRows := mydb.ParseRows(rows)
	if len(pRows) > 0 && string(pRows[0]["user_pwd"].([]byte)) == encpwd {
		return true
	}
	log.Println("Do not find" + username + ",or pwd is not correct")
	return false
}

func UpdateToken(username string, token string) bool {
	stmt, err := mydb.DBConn().Prepare(
		"replace into tbl_user_token(`user_name`,`user_token`) values (?,?)")
	if err != nil {
		log.Println(err.Error())
		return false
	}
	defer stmt.Close()
	_, err = stmt.Exec(username, token)
	if err != nil {
		log.Println(err.Error())
		return false
	}
	return true
}
