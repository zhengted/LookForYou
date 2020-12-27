package db

import (
	mydb "LookForYou/db/mysql"
	"fmt"
	"log"
)

// OnFileUploadFinished: 文件上传完成
func OnFileUploadFinished(filehash string, filename string,
	filesize int64, fileaddr string) bool {
	stmt, err := mydb.DBConn().Prepare(
		"insert ignore into tbl_file(`file_sha1`,`file_name`," +
			"`file_size`,`file_addr`),status values(?,?,?,?,1)")
	if err != nil {
		log.Println("Failed to connect db")
		return false
	}
	defer stmt.Close()

	ret, err := stmt.Exec(filehash, filename, filesize, fileaddr)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	if rf, err := ret.RowsAffected(); nil == err {
		if rf <= 0 {
			log.Printf("File with hash:%s has been uploaded before", filehash)
		}
		return true
	}
	return false
}
