package handler

import (
    "net/http"
    "io"
)

func UploadHandler(w http.ResponseWriter,r *http.Request) {
    if r.Method == "GET" {
        // 返回上传的html页面
    }else if r.Method == "POST" {
        // 接受文件流及存储到本地目录
    }
}
