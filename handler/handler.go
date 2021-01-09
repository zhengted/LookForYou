package handler

import (
	cmn "LookForYou/common"
	cfg "LookForYou/config"
	"LookForYou/db"
	"LookForYou/meta"
	"LookForYou/mq"
	"LookForYou/store/ceph"
	"LookForYou/store/oss"
	"LookForYou/util"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

// 处理文件上传
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// 返回上传的html页面
		data, err := ioutil.ReadFile("./static/view/index.html")
		if err != nil {
			io.WriteString(w, "internal server error")
			return
		}
		io.WriteString(w, string(data))
	} else if r.Method == "POST" {
		// 接受文件流及存储到本地目录
		file, head, err := r.FormFile("file")
		if err != nil {
			fmt.Printf("Failed to get data, err:%s\n", err.Error())
			return
		}
		defer file.Close()

		tmpPath := cfg.TempLocalRootDir + head.Filename
		fileMeta := meta.FileMeta{
			FileName: head.Filename,
			Location: tmpPath,
			UploadAt: time.Now().Format("2006-01-02 15:04:05"),
		}

		newFile, err := os.Create(fileMeta.Location)
		if err != nil {
			fmt.Printf("Failed to create file, err:%s\n", err.Error())
			return
		}
		defer newFile.Close()

		fileMeta.FileSize, err = io.Copy(newFile, file)
		if err != nil {
			fmt.Printf("Failed to save data into file, err:%s\n", err.Error())
			return
		}

		newFile.Seek(0, 0)
		fileMeta.FileSha1 = util.FileSha1(newFile)

		newFile.Seek(0, 0) // 游标重新回到文件头部
		mergePath := cfg.MergeLocalRootDir + fileMeta.FileSha1
		if cfg.CurrentStoreType == cmn.StoreCeph {
			// 文件写入Ceph存储
			data, _ := ioutil.ReadAll(newFile)
			cephPath := "/ceph/" + fileMeta.FileSha1
			err = ceph.PutObject("userfile", cephPath, data)
			if err != nil {
				fmt.Println("upload ceph err: " + err.Error())
				w.Write([]byte("Upload failed!"))
				return
			}
			fileMeta.Location = cephPath
		} else if cfg.CurrentStoreType == cmn.StoreOSS {
			// 文件写入OSS存储
			ossPath := "oss/" + fileMeta.FileSha1
			// 判断写入OSS为同步还是异步
			if !cfg.AsyncTransferEnable {
				// 文件写入OSS存储
				ossPath := "oss/" + fileMeta.FileSha1
				err = oss.Bucket().PutObject(ossPath, newFile)
				if err != nil {
					fmt.Println("upload oss err: " + err.Error())
					w.Write([]byte("Upload failed!"))
					return
				}
				fileMeta.Location = ossPath
			} else {
				// 文件尚未转移，暂存于本地mergePath
				fileMeta.Location = mergePath
				// 写入异步转移任务队列
				data := mq.TransferData{
					FileHash:      fileMeta.FileSha1,
					CurLocation:   fileMeta.Location,
					DestLocation:  ossPath,
					DestStoreType: cmn.StoreOSS,
				}
				pubData, _ := json.Marshal(data)
				pubSuc := mq.Publish(
					cfg.TransExchangeName,
					cfg.TransOssRoutingKey,
					pubData,
				)
				if !pubSuc {
					// TODO: 当前发送转移信息失败，稍后重试
				}
			}
		}
		_ = meta.UpdateFileMetaDB(fileMeta)

		// 写到tbl_user_file
		r.ParseForm()
		username := r.Form.Get("username")

		suc := db.OnUserFileUploadFinished(username, fileMeta.FileSha1, fileMeta.FileName, fileMeta.FileSize)
		if suc {
			http.Redirect(w, r, "/static/view/home.html", http.StatusFound) // 这里写错了一次，file前面的“/”没加上
		} else {
			w.Write([]byte("Upload Failed."))
		}

	}
}

// UploadSucHandler : 上传已完成
func UploadSucHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Upload finished!")
}

// GetFileMetaHandler: 获取文件元信息
func GetFileMetaHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() // 解析form数据
	filehash := r.Form["filehash"][0]
	fMeta, err := meta.GetFileMetaDB(filehash)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	bytes, err := json.Marshal(fMeta)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(bytes)
}

// FileQueryHandler: 查询批量的文件元信息
func FileQueryHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	limitCnt, _ := strconv.Atoi(r.Form.Get("limit"))
	username := r.Form.Get("username")
	//fileMetas := meta.GetLastFileMetas(limitCnt)
	fileMetas, err := db.QueryUserFileMetas(username, limitCnt)
	if err != nil {
		log.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	bytes, err := json.Marshal(fileMetas)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(bytes)

}

func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	fsha1 := r.Form.Get("filehash")
	fm := meta.GetFileMeta(fsha1)
	f, err := os.Open(fm.Location)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octect-stream")
	w.Header().Set("Content-Disposition", "attachment;filename=\""+fm.FileName+"\"")

	w.Write(data)
}

// FileMetaUpdateHandler:修改文件名
func FileMetaUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
	r.ParseForm()

	opType := r.Form.Get("op")
	fileSha1 := r.Form.Get("filehash")
	newFileName := r.Form.Get("filename")

	if opType != "0" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	curFileMeta := meta.GetFileMeta(fileSha1)
	curFileMeta.FileName = newFileName
	meta.UpdateFileMeta(curFileMeta)

	data, err := json.Marshal(curFileMeta)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-type", "application/json")
	w.Write(data)
}

// FileDeleteHandler: 删除文件及其元信息
func FileDeleteHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	fileSha1 := r.Form.Get("filehash")

	fMeta := meta.GetFileMeta(fileSha1)
	os.Remove(fMeta.Location)
	if !meta.RemoveFileDB(fileSha1) {
		log.Println("FileDeleteHandler Remove file in db failed", fileSha1)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	meta.RemoveFileMeta(fileSha1)

	w.WriteHeader(http.StatusOK)
}

// TryFastUploadHandler: 尝试秒传接口
func TryFastUploadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	// 1. 解析请求参数
	username := r.Form.Get("username")
	filehash := r.Form.Get("filehash")
	filename := r.Form.Get("filename")
	filesize, _ := strconv.Atoi(r.Form.Get("filesize"))

	// 2. 从文件表中查询相同hash的文件记录
	filemeta, err := meta.GetFileMetaDB(filehash)
	if err != nil {
		log.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// 3. 查不到记录则返回秒传失败
	if filemeta.FileSha1 == "" {
		resp := util.RespMsg{
			-1,
			"秒传失败，请访问普通上传接口",
			"Fuck",
		}
		w.Write(resp.JSONBytes())
		return
	}
	// 4. 上传过则将文件信息写入用户文件表，返回成功
	suc := db.OnUserFileUploadFinished(username, filehash, filename, int64(filesize))
	if suc {
		resp := util.RespMsg{
			0,
			"秒传成功",
			"Suck",
		}
		w.Write(resp.JSONBytes())
		return
	} else {
		resp := util.RespMsg{
			-2,
			"秒传失败，请稍后重试",
			"Shit",
		}
		w.Write(resp.JSONBytes())
		return
	}
}

func DownloadURLHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	filehash := r.Form.Get("filehash")

	row, _ := db.GetFileMeta(filehash)

	// TODO: 判断文件存在OSS还是ceph

	var fileaddr string
	row.FileAddr.Scan(&fileaddr)
	signedURL := oss.DownloadURL(fileaddr)
	if signedURL == "" {
		w.Write(util.NewRespMsg(-1, "download failed", nil).JSONBytes())
		return
	}
	w.Write([]byte(signedURL))
}
