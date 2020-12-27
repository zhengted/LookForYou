package meta

import (
	mydb "LookForYou/db"
	"sort"
)

// FileMeta: 文件元信息结构
type FileMeta struct {
	FileSha1 string
	FileName string
	FileSize int64
	Location string
	UploadAt string
}

var fileMetas map[string]FileMeta

func init() {
	fileMetas = make(map[string]FileMeta)
}

// UpdateFileMeta: 新增/更新文件元信息
func UpdateFileMeta(meta FileMeta) {
	fileMetas[meta.FileSha1] = meta
}

// UpdateFileMetaDB: 新增/更新文件元信息到数据库
func UpdateFileMetaDB(meta FileMeta) bool {
	return mydb.OnFileUploadFinished(meta.FileSha1, meta.FileName,
		meta.FileSize, meta.Location)
}

// GetFileMeta:通过sha1值获取文件的元信息对象
func GetFileMeta(fileSha1 string) FileMeta {
	return fileMetas[fileSha1]
}

// GetLastFileMetas: 获取批量的文件元信息列表
func GetLastFileMetas(count int) []FileMeta {
	var fMetaArray []FileMeta
	for _, f := range fileMetas {
		fMetaArray = append(fMetaArray, f)
	}
	sort.Sort(ByUploadTime(fMetaArray))
	if len(fMetaArray) <= count {
		return fMetaArray
	}
	return fMetaArray[0:count]
}

// RemoveFileMeta: 删除文件元信息
func RemoveFileMeta(filesha1 string) {
	delete(fileMetas, filesha1)
}
