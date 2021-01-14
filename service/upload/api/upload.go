package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	cmn "LookForYou/common"
	cmnCfg "LookForYou/config"
	"LookForYou/mq"
	dbcli "LookForYou/service/dbproxy/client"
	"LookForYou/store/ceph"
	"LookForYou/store/oss"
	"LookForYou/util"
)

func init() {
	if err := os.MkdirAll(cmnCfg.TempLocalRootDir, 0744); err != nil {
		fmt.Println("无法指定目录用于存储临时文件: " + cmnCfg.TempLocalRootDir)
		os.Exit(1)
	}
	if err := os.MkdirAll(cmnCfg.MergeLocalRootDir, 0744); err != nil {
		fmt.Println("无法指定目录用于存储合并后文件: " + cmnCfg.MergeLocalRootDir)
		os.Exit(1)
	}
}

// DoUploadHandler ： 处理文件上传
func DoUploadHandler(c *gin.Context) {
	errCode := 0
	defer func() {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		if errCode < 0 {
			c.JSON(http.StatusOK, gin.H{
				"code": errCode,
				"msg":  "上传失败",
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"code": errCode,
				"msg":  "上传成功",
			})
		}
	}()

	// 1. 从form表单中获得文件内容句柄
	file, head, err := c.Request.FormFile("file")
	if err != nil {
		log.Printf("Failed to get form data, err:%s\n", err.Error())
		errCode = -1
		return
	}
	defer file.Close()

	// 2. 把文件内容转为[]byte
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		log.Printf("Failed to get file data, err:%s\n", err.Error())
		errCode = -2
		return
	}

	// 3. 构建文件元信息
	fileMeta := dbcli.FileMeta{
		FileName: head.Filename,
		FileSha1: util.Sha1(buf.Bytes()), //　计算文件sha1
		FileSize: int64(len(buf.Bytes())),
		UploadAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	// 4. 将文件写入临时存储位置
	fileMeta.Location = cmnCfg.MergeLocalRootDir + fileMeta.FileSha1 // 存储地址
	newFile, err := os.Create(fileMeta.Location)
	if err != nil {
		log.Printf("Failed to create file, err:%s\n", err.Error())
		errCode = -3
		return
	}
	defer newFile.Close()

	nByte, err := newFile.Write(buf.Bytes())
	if int64(nByte) != fileMeta.FileSize || err != nil {
		log.Printf("Failed to save data into file, writtenSize:%d, err:%s\n", nByte, err.Error())
		errCode = -4
		return
	}

	// 5. 同步或异步将文件转移到Ceph/OSS
	newFile.Seek(0, 0) // 游标重新回到文件头部
	if cmnCfg.CurrentStoreType == cmn.StoreCeph {
		// 文件写入Ceph存储
		data, _ := ioutil.ReadAll(newFile)
		cephPath := cmnCfg.CephRootDir + fileMeta.FileSha1
		_ = ceph.PutObject("userfile", cephPath, data)
		fileMeta.Location = cephPath
	} else if cmnCfg.CurrentStoreType == cmn.StoreOSS {
		// 文件写入OSS存储
		ossPath := cmnCfg.OSSRootDir + fileMeta.FileSha1
		// 判断写入OSS为同步还是异步
		if !cmnCfg.AsyncTransferEnable {
			// TODO: 设置oss中的文件名，方便指定文件名下载
			err = oss.Bucket().PutObject(ossPath, newFile)
			if err != nil {
				log.Println(err.Error())
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
				cmnCfg.TransExchangeName,
				cmnCfg.TransOssRoutingKey,
				pubData,
			)
			if !pubSuc {
				// TODO: 当前发送转移信息失败，稍后重试
			}
		}
	}

	//6.  更新文件表记录
	_, err = dbcli.OnFileUploadFinished(fileMeta)
	if err != nil {
		errCode = -6
		return
	}

	// 7. 更新用户文件表
	username := c.Request.FormValue("username")
	upRes, err := dbcli.OnUserFileUploadFinished(username, fileMeta)
	if err == nil && upRes.Suc {
		errCode = 0
	} else {
		errCode = -6
	}
}

// TryFastUploadHandler : 尝试秒传接口
func TryFastUploadHandler(c *gin.Context) {

	// 1. 解析请求参数
	username := c.Request.FormValue("username")
	filehash := c.Request.FormValue("filehash")
	filename := c.Request.FormValue("filename")
	// filesize, _ := strconv.Atoi(c.Request.FormValue("filesize"))

	// 2. 从文件表中查询相同hash的文件记录
	fileMetaResp, err := dbcli.GetFileMeta(filehash)
	if err != nil {
		log.Println(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	// 3. 查不到记录则返回秒传失败 (2020-05更新，判断Data == nil)
	if !fileMetaResp.Suc || fileMetaResp.Data == nil {
		resp := util.RespMsg{
			Code: -1,
			Msg:  "秒传失败，请访问普通上传接口",
		}
		c.Data(http.StatusOK, "application/json", resp.JSONBytes())
		return
	}

	// 4. 上传过则将文件信息写入用户文件表， 返回成功
	tblFile := dbcli.ToTableFile(fileMetaResp.Data)
	fmeta := dbcli.TableFileToFileMeta(tblFile)
	fmeta.FileName = filename
	upRes, err := dbcli.OnUserFileUploadFinished(username, fmeta)
	if err == nil && upRes.Suc {
		resp := util.RespMsg{
			Code: 0,
			Msg:  "秒传成功",
		}
		c.Data(http.StatusOK, "application/json", resp.JSONBytes())
		return
	}
	resp := util.RespMsg{
		Code: -2,
		Msg:  "秒传失败，请稍后重试",
	}
	c.Data(http.StatusOK, "application/json", resp.JSONBytes())
	return
}
