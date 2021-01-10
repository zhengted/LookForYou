package handler

import (
	rPool "LookForYou/cache/redis"
	cfg "LookForYou/config"
	dblayer "LookForYou/db"
	"LookForYou/util"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
	"log"
	"math"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

// 初始化信息结构体
type MultiPartUploadInfo struct {
	FileHash   string
	FileSize   int
	UploadID   string
	ChunkSize  int
	ChunkCount int
	// 已经上传完成的分块索引列表
	ChunkExists []int
}

const (
	// ChunkKeyPrefix: 分块信息对应的redis键前缀
	ChunkKeyPrefix = "MP_"
	// HashUpIDKeyPrefix: 文件hash映射uploadid对应的redis前缀
	HashUpIDKeyPrefix = "HASH_UPID_"
)

func init() {
	if err := os.MkdirAll(cfg.ChunckLocalRootDir, 0744); err != nil {
		fmt.Println("无法指定目录用于存储分块文件:"+cfg.ChunckLocalRootDir, err.Error())
		os.Exit(1)
	}
	if err := os.MkdirAll(cfg.MergeLocalRootDir, 0744); err != nil {
		fmt.Println("无法指定目录用于存储合并后文件:"+cfg.MergeLocalRootDir, err.Error())
		os.Exit(1)
	}
}

// InitialMultipartUploadHandler : 初始化分块上传
func InitialMultipartUploadHandler(c *gin.Context) {
	// 1. 解析用户请求参数
	username := c.Request.FormValue("username")
	filehash := c.Request.FormValue("filehash")
	filesize, err := strconv.Atoi(c.Request.FormValue("filesize"))
	if err != nil {
		c.JSON(
			http.StatusOK,
			gin.H{
				"code": -1,
				"msg":  "params invalid",
			})
		return
	}

	// 判断文件是否存在
	if dblayer.IsUserFileUploaded(username, filehash) {
		c.JSON(
			http.StatusOK,
			gin.H{
				"code": -2,
				"msg":  "params invalid",
			})
		return
	}

	// 2. 获得redis的一个连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	uploadID := ""

	// 3. 通过文件hash判断是否断点续传，并获取uploadID
	keyExists, _ := redis.Bool(rConn.Do("EXISTS", HashUpIDKeyPrefix+filehash))
	if keyExists {
		uploadID, err = redis.String(rConn.Do("GET", HashUpIDKeyPrefix+filehash))
		if err != nil {
			log.Println(err.Error())
			resp := util.RespMsg{
				-2,
				"Upload part failed",
				nil,
			}
			c.Data(http.StatusOK, "application/json", resp.JSONBytes())
			return
		}
	}

	// 4.1 首次上传则新建uploadID
	// 4.2 断点续传则根据uploadID获取已上传的文件分块列表
	chunksExist := []int{}
	if uploadID == "" {
		uploadID = username + fmt.Sprintf("%x", time.Now().UnixNano())
	} else {
		chunks, err := redis.Values(rConn.Do("HGETALL", ChunkKeyPrefix+uploadID))
		if err != nil {
			log.Println(err.Error())
			resp := util.RespMsg{
				-3,
				"Upload part failed",
				nil,
			}
			c.Data(http.StatusOK, "application/json", resp.JSONBytes())
			return
		}
		for i := 0; i < len(chunks); i += 2 {
			k := string(chunks[i].([]byte))
			v := string(chunks[i+1].([]byte))
			if strings.HasPrefix(k, "chkidx_") && v == "1" {
				// chkidx_6 -> 6
				chunkIdx, _ := strconv.Atoi(k[7:len(k)])
				chunksExist = append(chunksExist, chunkIdx)
			}
		}
	}

	// 5. 生成分块上传的初始化信息
	upInfo := MultiPartUploadInfo{
		FileHash:    filehash,
		FileSize:    filesize,
		UploadID:    uploadID,
		ChunkSize:   5 * 1024 * 1024, // 5MB
		ChunkCount:  int(math.Ceil(float64(filesize) / (5 * 1024 * 1024))),
		ChunkExists: chunksExist,
	}

	// 6. 将初始化信息写入到redis缓存
	if len(upInfo.ChunkExists) <= 0 {
		hkey := ChunkKeyPrefix + upInfo.UploadID
		rConn.Do("HSET", hkey, "chunkcount", upInfo.ChunkCount)
		rConn.Do("HSET", hkey, "filehash", upInfo.FileHash)
		rConn.Do("HSET", hkey, "filesize", upInfo.FileSize)
		rConn.Do("EXPIRE", hkey, 43200)
		rConn.Do("SET", HashUpIDKeyPrefix+filehash, upInfo.UploadID, "EX", 43200)
	}

	// 7. 将响应初始化数据返回到客户端
	c.JSON(
		http.StatusOK,
		gin.H{
			"code": 0,
			"msg":  "OK",
			"data": upInfo,
		})
}

// UploadPartHandler : 上传文件分块
func UploadPartHandler(c *gin.Context) {
	// 1. 解析用户请求参数
	//	username := r.Form.Get("username")
	uploadID := c.Request.FormValue("uploadid")
	chunkSha1 := c.Request.FormValue("chkhash")
	chunkIndex := c.Request.FormValue("index")

	// 2. 获得redis连接池中的一个连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	// 3. 获得文件句柄，用于存储分块内容
	//fmt.Println("UploadPartHandler Get file handle")
	fpath := cfg.ChunckLocalRootDir + uploadID + "/" + chunkIndex
	os.MkdirAll(path.Dir(fpath), 0744)
	fd, err := os.Create(fpath)
	if err != nil {
		log.Println(err.Error())
		resp := util.RespMsg{
			-1,
			"Upload part failed",
			nil,
		}
		c.Data(http.StatusOK, "application/json", resp.JSONBytes())
		return
	}
	defer fd.Close()

	buf := make([]byte, 1024*1024)
	for {
		n, err := c.Request.Body.Read(buf)
		fd.Write(buf[:n])
		if err != nil {
			break
		}
	}

	// 校验分块hash (updated at 2020-05)
	cmpSha1, err := util.ComputeSha1ByShell(fpath)
	if err != nil || cmpSha1 != chunkSha1 {
		log.Printf("Verify chunk sha1 failed, compare OK: %t, err:%+v\n",
			cmpSha1 == chunkSha1, err)
		log.Println(err.Error())
		resp := util.RespMsg{
			-2,
			"Upload part failed",
			nil,
		}
		c.Data(http.StatusOK, "application/json", resp.JSONBytes())
		return
	}

	// 4. 更新redis缓存状态
	rConn.Do("HSET", ChunkKeyPrefix+uploadID, "chkidx_"+chunkIndex, 1)
	fmt.Println("Update Redis status success", ChunkKeyPrefix+uploadID, "chkidx_"+chunkIndex)

	// 5. 返回处理结果到客户端
	c.JSON(
		http.StatusOK,
		gin.H{
			"code": 0,
			"msg":  "OK",
			"data": nil,
		})

}

// CompleteUploadHandler : 通知上传合并
func CompleteUploadHandler(c *gin.Context) {
	// 1. 解析请求参数
	uploadID := c.Request.FormValue("uploadid")
	username := c.Request.FormValue("username")
	filehash := c.Request.FormValue("filehash")
	filesize := c.Request.FormValue("filesize")
	filename := c.Request.FormValue("filename")

	// 2. 获得redis连接池连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	// 3. 通过uploadid查询redis并判断是否所有分块上传完成
	data, err := redis.Values(rConn.Do("HGETALL", "MP_"+uploadID))
	if err != nil {
		fmt.Println("CompleteUploadHandler Check redis status failed", data)
		resp := util.RespMsg{
			-1,
			"Complete upload failed",
			nil,
		}
		c.Data(http.StatusOK, "application/json", resp.JSONBytes())
		return
	}
	totalCount := 0
	chunkCount := 0
	for i := 0; i < len(data); i += 2 {
		k := string(data[i].([]byte))
		v := string(data[i+1].([]byte))
		if k == "chunkcount" {
			totalCount, _ = strconv.Atoi(v)
		} else if strings.HasPrefix(k, "chkidx_") && v == "1" {
			chunkCount++
		}
	}
	if totalCount != chunkCount {
		resp := util.RespMsg{
			-2,
			"Complete upload failed invalid request",
			nil,
		}
		c.Data(http.StatusOK, "application/json", resp.JSONBytes())
		return
	}

	// 4. 合并分块 (备注: 更新于2020/04/01; 此合并逻辑非必须实现，因后期转移到ceph/oss时也可以通过分块方式上传)
	if mergeSuc := util.MergeChuncksByShell(cfg.ChunckLocalRootDir+uploadID, cfg.MergeLocalRootDir+filehash, filehash); !mergeSuc {
		resp := util.RespMsg{
			-3,
			"Complete upload failed",
			nil,
		}
		c.Data(http.StatusOK, "application/json", resp.JSONBytes())
		return
	}

	// 5. 更新唯一文件表及用户文件表
	fsize, _ := strconv.Atoi(filesize)
	// 更新于2020-04: 增加fileaddr参数的写入
	dblayer.OnFileUploadFinished(filehash, filename, int64(fsize), cfg.MergeLocalRootDir+filehash)
	dblayer.OnUserFileUploadFinished(username, filehash, filename, int64(fsize))

	// 更新于2020-04: 删除已上传的分块文件及redis分块信息
	_, delHashErr := rConn.Do("DEL", HashUpIDKeyPrefix+filehash)
	delUploadID, delUploadInfoErr := redis.Int64(rConn.Do("DEL", ChunkKeyPrefix+uploadID))
	if delUploadID != 1 || delUploadInfoErr != nil || delHashErr != nil {
		resp := util.RespMsg{
			-4,
			"Complete upload failed",
			nil,
		}
		c.Data(http.StatusOK, "application/json", resp.JSONBytes())
		return
	}

	delRes := util.RemovePathByShell(cfg.ChunckLocalRootDir + uploadID)
	if !delRes {
		fmt.Printf("Failed to delete chuncks as upload comoleted, uploadID: %s\n", uploadID)
	}

	// 6. 响应处理结果
	c.JSON(
		http.StatusOK,
		gin.H{
			"code": 0,
			"msg":  "OK",
			"data": nil,
		})

}

// CancelUploadHandler: 文件取消上传接口
func CancelUploadHandler(c *gin.Context) {
	// 1. 解析用户请求参数
	filehash := c.Request.FormValue("filehash")
	// 2. 获得redis连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()
	// 3. 检查UploadID是否存在，如果存在则删除
	uploadID, err := redis.String(rConn.Do("GET", HashUpIDKeyPrefix+filehash))
	if err != nil || uploadID == "" {
		resp := util.RespMsg{
			-1,
			"CancelUpload failed",
			nil,
		}
		c.Data(http.StatusOK, "application/json", resp.JSONBytes())
		return
	}

	_, delHashErr := rConn.Do("DEL", HashUpIDKeyPrefix+filehash)
	_, delUploadInfoErr := rConn.Do("DEL", ChunkKeyPrefix+uploadID)
	if delHashErr != nil || delUploadInfoErr != nil {
		resp := util.RespMsg{
			-2,
			"CancelUpload failed",
			nil,
		}
		c.Data(http.StatusOK, "application/json", resp.JSONBytes())
		return
	}

	// 4. 删除已上传的分块文件
	delChkRes := util.RemovePathByShell(cfg.ChunckLocalRootDir + uploadID)
	if !delChkRes {
		log.Println("Failed to delete chunks as upload canceled,uploadID:%s\n", uploadID)
	}

	// 5. 响应客户端
	c.JSON(
		http.StatusOK,
		gin.H{
			"code": 0,
			"msg":  "OK",
			"data": nil,
		})
}
