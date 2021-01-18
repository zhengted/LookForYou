package api

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"LookForYou/common"
	cfg "LookForYou/config"
	dbcli "LookForYou/service/dbproxy/client"
	"LookForYou/store/ceph"
	"LookForYou/store/oss"
	// dlcfg "LookForYou/service/download/config"
)

// DownloadURLHandler : 生成文件的下载地址
func DownloadURLHandler(c *gin.Context) {
	filehash := c.Request.FormValue("filehash")
	// 从文件表查找记录
	dbResp, err := dbcli.GetFileMeta(filehash)
	if err != nil {
		c.JSON(
			http.StatusOK,
			gin.H{
				"code": common.StatusServerError,
				"msg":  "server error",
			})
		return
	}

	tblFile := dbcli.ToTableFile(dbResp.Data)

	// TODO: 判断文件存在OSS，还是Ceph，还是在本地
	if strings.HasPrefix(tblFile.FileAddr.String, cfg.MergeLocalRootDir) ||
		strings.HasPrefix(tblFile.FileAddr.String, cfg.CephRootDir) {
		username := c.Request.FormValue("username")
		token := c.Request.FormValue("token")
		tmpURL := fmt.Sprintf("http://%s/file/download?filehash=%s&username=%s&token=%s",
			c.Request.Host, filehash, username, token)
		c.Data(http.StatusOK, "application/octet-stream", []byte(tmpURL))
	} else if strings.HasPrefix(tblFile.FileAddr.String, cfg.OSSRootDir) {
		// oss下载url
		signedURL := oss.DownloadURL(tblFile.FileAddr.String)
		log.Println(tblFile.FileAddr.String)
		c.Data(http.StatusOK, "application/octet-stream", []byte(signedURL))
	} else {
		c.Data(http.StatusOK, "application/octet-stream", []byte("Error: 下载链接暂时无法生成"))
	}
}

// DownloadHandler : 文件下载接口
func DownloadHandler(c *gin.Context) {
	fsha1 := c.Request.FormValue("filehash")
	username := c.Request.FormValue("username")
	// TODO: 处理异常情况
	fResp, ferr := dbcli.GetFileMeta(fsha1)
	ufResp, uferr := dbcli.QueryUserFileMeta(username, fsha1)
	if ferr != nil || uferr != nil || !fResp.Suc || !ufResp.Suc {
		c.JSON(
			http.StatusOK,
			gin.H{
				"code": common.StatusServerError,
				"msg":  "server error",
			})
		return
	}
	uniqFile := dbcli.ToTableFile(fResp.Data)
	userFile := dbcli.ToTableUserFile(ufResp.Data)

	if strings.HasPrefix(uniqFile.FileAddr.String, cfg.MergeLocalRootDir) {
		// 本地文件， 直接下载
		c.FileAttachment(uniqFile.FileAddr.String, userFile.FileName)
	} else if strings.HasPrefix(uniqFile.FileAddr.String, cfg.CephRootDir) {
		fmt.Println("to download file from ceph...")
		// ceph中的文件，通过ceph api先下载
		bucket := ceph.GetCephBucket("userfile")
		data, _ := bucket.Get(uniqFile.FileAddr.String)
		//	c.Header("content-type", "application/octect-stream")
		c.Header("content-disposition", "attachment; filename=\""+userFile.FileName+"\"")
		c.Data(http.StatusOK, "application/octect-stream", data)
	} else if strings.HasPrefix(uniqFile.FileAddr.String, cfg.OSSRootDir) {
		fmt.Println("to download file from oss...")
		var err1 error
		var err2 error
		var fd io.ReadCloser
		var fileData []byte
		fd, err1 = oss.Bucket().GetObject(uniqFile.FileAddr.String)
		if err1 == nil {
			fileData, err2 = ioutil.ReadAll(fd)
			if err2 == nil {
				c.Header("content-disposition", "attachment; filename=\""+userFile.FileName+"\"")
				c.Data(http.StatusOK, "application/octect-stream", fileData)
			}
		}
		if err1 != nil || err2 != nil {
			c.Data(http.StatusInternalServerError, "application/octect-stream", []byte("Intern server error."))
			return
		}
	} else {
		c.Data(http.StatusNotFound, "application/octect-stream", []byte("File not found."))
		return
	}
}

// RangeDownloadHandler : 支持断点的文件下载接口
func RangeDownloadHandler(c *gin.Context) {
	fsha1 := c.Request.FormValue("filehash")
	username := c.Request.FormValue("username")

	fResp, ferr := dbcli.GetFileMeta(fsha1)
	ufResp, uferr := dbcli.QueryUserFileMeta(username, fsha1)
	if ferr != nil || uferr != nil || !fResp.Suc || !ufResp.Suc {
		c.JSON(
			http.StatusOK,
			gin.H{
				"code": common.StatusServerError,
				"msg":  "server error",
			})
		return
	}
	//         uniqFile := dbcli.ToTableFile(fResp.Data)
	userFile := dbcli.ToTableUserFile(ufResp.Data)

	// 使用本地目录文件
	fpath := cfg.MergeLocalRootDir + fsha1
	fmt.Println("range-download-fpath: " + fpath)

	f, err := os.Open(fpath)
	if err != nil {
		c.JSON(
			http.StatusOK,
			gin.H{
				"code": common.StatusServerError,
				"msg":  "server error",
			})
		return
	}
	defer f.Close()

	c.Writer.Header().Set("Content-Type", "application/octect-stream")
	c.Writer.Header().Set("content-disposition", "attachment; filename=\""+userFile.FileName+"\"")
	c.File(fpath)
}
