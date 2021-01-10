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
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// 处理文件上传
func UploadHandler(c *gin.Context) {
	data, err := ioutil.ReadFile("./static/view/upload.html")
	if err != nil {
		c.String(404, `网页不存在`)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}

// DoUploadHandler ： 处理文件上传
func DoUploadHandler(c *gin.Context) {
	errCode := 0
	defer func() {
		if errCode < 0 {
			c.JSON(http.StatusOK, gin.H{
				"code": errCode,
				"msg":  "Upload failed",
			})
		}
	}()

	// 1. 从form表单中获得文件内容句柄
	file, head, err := c.Request.FormFile("file")
	if err != nil {
		fmt.Printf("Failed to get form data, err:%s\n", err.Error())
		errCode = -1
		return
	}
	defer file.Close()

	// 2. 把文件内容转为[]byte
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		fmt.Printf("Failed to get file data, err:%s\n", err.Error())
		errCode = -2
		return
	}

	// 3. 构建文件元信息
	fileMeta := meta.FileMeta{
		FileName: head.Filename,
		FileSha1: util.Sha1(buf.Bytes()), //　计算文件sha1
		FileSize: int64(len(buf.Bytes())),
		UploadAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	// 4. 将文件写入临时存储位置
	fileMeta.Location = cfg.TempLocalRootDir + fileMeta.FileSha1 // 临时存储地址
	newFile, err := os.Create(fileMeta.Location)
	if err != nil {
		fmt.Printf("Failed to create file, err:%s\n", err.Error())
		errCode = -3
		return
	}
	defer newFile.Close()

	nByte, err := newFile.Write(buf.Bytes())
	if int64(nByte) != fileMeta.FileSize || err != nil {
		fmt.Printf("Failed to save data into file, writtenSize:%d, err:%s\n", nByte, err.Error())
		errCode = -4
		return
	}

	// 5. 同步或异步将文件转移到Ceph/OSS
	newFile.Seek(0, 0) // 游标重新回到文件头部
	if cfg.CurrentStoreType == cmn.StoreCeph {
		// 文件写入Ceph存储
		data, _ := ioutil.ReadAll(newFile)
		cephPath := "/ceph/" + fileMeta.FileSha1
		_ = ceph.PutObject("userfile", cephPath, data)
		fileMeta.Location = cephPath
	} else if cfg.CurrentStoreType == cmn.StoreOSS {
		// 文件写入OSS存储
		ossPath := "oss/" + fileMeta.FileSha1
		// 判断写入OSS为同步还是异步
		if !cfg.AsyncTransferEnable {
			// TODO: 设置oss中的文件名，方便指定文件名下载
			err = oss.Bucket().PutObject(ossPath, newFile)
			if err != nil {
				fmt.Println(err.Error())
				errCode = -5
				return
			}
			fileMeta.Location = ossPath
		} else {
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

	//6.  更新文件表记录
	_ = meta.UpdateFileMetaDB(fileMeta)

	// 7. 更新用户文件表
	username := c.Request.FormValue("username")
	suc := db.OnUserFileUploadFinished(username, fileMeta.FileSha1,
		fileMeta.FileName, fileMeta.FileSize)
	if suc {
		c.Redirect(http.StatusFound, "/static/view/home.html")
	} else {
		errCode = -6
	}
}

// UploadSucHandler : 上传已完成
func UploadSucHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"msg":  "Upload finished!",
		"code": 0,
	})
}

// GetFileMetaHandler: 获取文件元信息
func GetFileMetaHandler(c *gin.Context) {
	filehash := c.Request.FormValue("filehash")
	fMeta, err := meta.GetFileMetaDB(filehash)
	if err != nil {
		c.JSON(http.StatusInternalServerError,
			gin.H{
				"code": -2,
				"msg":  "Upload failed!",
			})
		return
	}

	if fMeta != (meta.FileMeta{}) {
		data, err := json.Marshal(fMeta)
		if err != nil {
			c.JSON(http.StatusInternalServerError,
				gin.H{
					"code": -3,
					"msg":  "Upload failed!",
				})
			return
		}
		c.Data(http.StatusOK, "application/json", data)
	} else {
		c.JSON(http.StatusOK,
			gin.H{
				"code": -4,
				"msg":  "No such file",
			})
	}
}

// FileQueryHandler: 查询批量的文件元信息
func FileQueryHandler(c *gin.Context) {
	limitCnt, _ := strconv.Atoi(c.Request.FormValue("limit"))
	username := c.Request.FormValue("username")
	//fileMetas := meta.GetLastFileMetas(limitCnt)
	fileMetas, err := db.QueryUserFileMetas(username, limitCnt)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError,
			gin.H{
				"code": -1,
				"msg":  "File Query failed!",
			})
		return
	}
	bytes, err := json.Marshal(fileMetas)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError,
			gin.H{
				"code": -2,
				"msg":  "File Query failed!",
			})
		return
	}
	c.Data(http.StatusOK, "application/json", bytes)

}

func DownloadHandler(c *gin.Context) {
	fsha1 := c.Request.FormValue("filehash")
	username := c.Request.FormValue("username")
	fm, err := meta.GetFileMetaDB(fsha1)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError,
			gin.H{
				"code": -1,
				"msg":  "Download failed!",
			})
		return
	}
	userFile, err := db.QueryUserFileMeta(username, fsha1)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError,
			gin.H{
				"code": -2,
				"msg":  "Download failed!",
			})
		return
	}
	if strings.HasPrefix(fm.Location, cfg.TempLocalRootDir) {
		// 本地存储，直接下载
		c.FileAttachment(fm.Location, userFile.FileName)
	} else if strings.HasPrefix(fm.Location, cfg.CephRootDir) {
		// ceph中的文件 通过ceph的API下载
		bucket := ceph.GetCephBucket("userfile")
		data, _ := bucket.Get(fm.Location)

		c.Header("content-disposition", "attachment; filename=\""+userFile.FileName+"\"")
		c.Data(http.StatusOK, "application/octect-stream", data)
	}
}

// FileMetaUpdateHandler:修改文件名 POST
func FileMetaUpdateHandler(c *gin.Context) {
	opType := c.Request.FormValue("op")
	fileSha1 := c.Request.FormValue("filehash")
	newFileName := c.Request.FormValue("filename")
	username := c.Request.FormValue("username")

	if opType != "0" || len(newFileName) < 1 {
		c.Status(http.StatusForbidden)
		return
	}

	// 更新用户文件表的tbl_user_file中的文件名，tbl_file的文件名不用修改
	_ = db.RenameFileName(username, fileSha1, newFileName)

	userFile, err := db.QueryUserFileMeta(username, fileSha1)
	if err != nil {
		log.Println(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(userFile)
	if err != nil {
		log.Println(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, data)
}

// FileDeleteHandler: 删除文件及其元信息
func FileDeleteHandler(c *gin.Context) {
	username := c.Request.FormValue("username")
	fileSha1 := c.Request.FormValue("filehash")

	fMeta, err := meta.GetFileMetaDB(fileSha1)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	// 删除本地文件
	os.Remove(fMeta.Location)

	// 删除文件表中的一条记录
	suc := db.DeleteUserFile(username, fileSha1)
	if !suc {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusOK)
}

// TryFastUploadHandler: 尝试秒传接口
func TryFastUploadHandler(c *gin.Context) {
	// 1. 解析请求参数
	username := c.Request.FormValue("username")
	filehash := c.Request.FormValue("filehash")
	filename := c.Request.FormValue("filename")
	filesize, _ := strconv.Atoi(c.Request.FormValue("filesize"))

	// 2. 从文件表中查询相同hash的文件记录
	filemeta, err := meta.GetFileMetaDB(filehash)
	if err != nil {
		log.Println(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}
	// 3. 查不到记录则返回秒传失败
	if filemeta.FileSha1 == "" {
		resp := util.RespMsg{
			-1,
			"秒传失败，请访问普通上传接口",
			"Fuck",
		}
		c.Data(http.StatusOK, "application/json", resp.JSONBytes())
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
		c.Data(http.StatusOK, "application/json", resp.JSONBytes())
		return
	} else {
		resp := util.RespMsg{
			-2,
			"秒传失败，请稍后重试",
			"Shit",
		}
		c.Data(http.StatusOK, "application/json", resp.JSONBytes())
		return
	}
}

func DownloadURLHandler(c *gin.Context) {
	filehash := c.Request.FormValue("filehash")

	row, _ := db.GetFileMeta(filehash)

	// TODO: 判断文件存在OSS，还是Ceph，还是在本地
	if strings.HasPrefix(row.FileAddr.String, cfg.TempLocalRootDir) ||
		strings.HasPrefix(row.FileAddr.String, cfg.CephRootDir) {
		username := c.Request.FormValue("username")
		token := c.Request.FormValue("token")
		tmpURL := fmt.Sprintf("http://%s/file/download?filehash=%s&username=%s&token=%s",
			c.Request.Host, filehash, username, token)
		c.Data(http.StatusOK, "octet-stream", []byte(tmpURL))
	} else if strings.HasPrefix(row.FileAddr.String, "oss/") {
		// oss下载url
		signedURL := oss.DownloadURL(row.FileAddr.String)
		fmt.Println(row.FileAddr.String)
		c.Data(http.StatusOK, "octet-stream", []byte(signedURL))
	}
}
