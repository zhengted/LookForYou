package db

import (
	"LookForYou/config"
	mydb "LookForYou/db/mysql"
	"database/sql"
	"fmt"
	"log"
)

// OnFileUploadFinished: 文件上传完成
func OnFileUploadFinished(filehash string, filename string,
	filesize int64, fileaddr string) bool {
	stmt, err := mydb.DBConn().Prepare(
		"insert ignore into tbl_file(`file_sha1`,`file_name`," +
			"`file_size`,`file_addr`,`status`) values(?,?,?,?,?)")
	if err != nil {
		log.Println("Failed to connect db")
		return false
	}
	defer stmt.Close()
	status := config.FileState_CanUse
	ret, err := stmt.Exec(filehash, filename, filesize, fileaddr, status)
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

type TableFile struct {
	FileHash string
	FileName sql.NullString
	FileSize sql.NullInt64
	FileAddr sql.NullString
}

// GetFileMeta: 获取文件元信息
func GetFileMeta(filehash string) (*TableFile, error) {
	stmt, err := mydb.DBConn().Prepare(
		"select file_sha1,file_addr,file_name," +
			"file_size from tbl_file " +
			"where file_sha1=? and status=1 limit 1")
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer stmt.Close()
	tfile := TableFile{}
	err = stmt.QueryRow(filehash).Scan(
		&tfile.FileHash, &tfile.FileAddr, &tfile.FileName, &tfile.FileSize)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return &tfile, nil
}

// DelFileMeta: 删除文件元信息，注意这里只是把数据库中的状态修改成FileStatus_Deleted
func DelFileMeta(filehash string) bool {
	stmt, err := mydb.DBConn().Prepare(
		"update tbl_file set status=? where file_sha1=?")
	if err != nil {
		log.Println(err.Error())
		return false
	}
	defer stmt.Close()
	newStatus := config.FileState_Deleted
	ret, err := stmt.Exec(newStatus, filehash)
	if err != nil {
		log.Println(err.Error())
		return false
	}
	if rf, err := ret.RowsAffected(); nil == err {
		if rf >= 0 {
			log.Printf("The file with hash %s has been modified to DELETED", filehash)
		}
		return true
	}
	return false
}

// UpdateFileLocation : 更新文件的存储地址(如文件被转移了)
func UpdateFileLocation(filehash string, fileaddr string) bool {
	stmt, err := mydb.DBConn().Prepare(
		"update tbl_file set`file_addr`=? where  `file_sha1`=? limit 1")
	if err != nil {
		fmt.Println("预编译sql失败, err:" + err.Error())
		return false
	}
	defer stmt.Close()

	ret, err := stmt.Exec(fileaddr, filehash)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	if rf, err := ret.RowsAffected(); nil == err {
		if rf <= 0 {
			fmt.Printf("更新文件location失败, filehash:%s", filehash)
		}
		return true
	}
	return false
}
