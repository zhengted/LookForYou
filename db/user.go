package db

import (
	mydb "LookForYou/db/mysql"
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
	defer stmt.Close()
	rows, err := stmt.Query(username)
	if err != nil {
		log.Println(err.Error())
		return false
	} else if rows == nil {
		log.Println("username not found:" + username)
		return false
	}
	pRows := mydb.ParseRows(rows)
	if len(pRows) > 0 && string(pRows[0]["user_pwd"].([]byte)) == encpwd {
		return true
	} else if len(pRows) > 0 {
		log.Println("Do not find,or pwd is not correct", username, string(pRows[0]["user_pwd"].([]byte)), encpwd)
	}
	log.Println("Do not find,or pwd is not correct", username, encpwd)
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

type User struct {
	Username     string
	Email        string
	Phone        string
	SignupAt     string
	LastActiveAt string
	Status       int
}

func GetUserInfo(username string) (User, error) {
	user := User{}
	stmt, err := mydb.DBConn().Prepare(
		"select user_name, signup_at from tbl_user where user_name=? limit 1")
	if err != nil {
		log.Println(err.Error())
		return user, err
	}
	defer stmt.Close()
	err = stmt.QueryRow(username).Scan(&user.Username, &user.SignupAt)
	if err != nil {
		return user, err
	}
	return user, nil
}

func GetTokenFromDB(username string) (string, error) {
	var ret string
	stmt, err := mydb.DBConn().Prepare(
		"select user_token where user_name=? limit 1")
	if err != nil {
		log.Println(err.Error())
		return "", err
	}
	defer stmt.Close()
	err = stmt.QueryRow(username).Scan(&ret)
	if err != nil {
		log.Println(err.Error())
		return "", err
	}
	return ret, nil
}
