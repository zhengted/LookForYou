package db

import (
	mydb "LookForYou/db/mysql"
	"log"
	"time"
)

// UserFile:用户文件表结构体
type UserFile struct {
	UserName    string
	FileHash    string
	FileName    string
	FileSize    int64
	UploadAt    string
	LastUpdated string
}

func OnUserFileUploadFinished(username, filehash, filename string, filesize int64) bool {
	stmt, err := mydb.DBConn().Prepare(
		"insert ignore into tbl_user_file(`user_name`,`file_sha1`,`file_name`," +
			"`file_size`,upload_at) values (?,?,?,?,?)")
	if err != nil {
		return false
	}
	defer stmt.Close()
	_, err = stmt.Exec(username, filehash, filename, filesize, time.Now())
	if err != nil {
		return false
	}
	return true
}

// QueryUserFileMetas: 批量获取文件元信息
func QueryUserFileMetas(username string, limit int) ([]UserFile, error) {
	stmt, err := mydb.DBConn().Prepare(
		"select file_sha1,file_name,file_size,upload_at,last_update from tbl_user_file where user_name=? limit ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, err := stmt.Query(username, limit)
	if err != nil {
		return nil, err
	}
	var userFiles []UserFile
	for rows.Next() {
		ufile := UserFile{}
		err = rows.Scan(&ufile.FileHash, &ufile.FileName, &ufile.FileSize, &ufile.UploadAt, &ufile.LastUpdated)
		if err != nil {
			log.Println(err.Error())
			break
		}
		userFiles = append(userFiles, ufile)
	}
	return userFiles, nil
}

// IsUserFileUploaded : 用户文件是否已经上传过
func IsUserFileUploaded(username string, filehash string) bool {
	stmt, err := mydb.DBConn().Prepare(
		"select 1 from tbl_user_file where user_name=? and file_sha1=? and status=1 limit 1")
	rows, err := stmt.Query(username, filehash)
	if err != nil {
		return false
	} else if rows == nil || !rows.Next() {
		return false
	}
	return true
}
