package ceph

import (
	"gopkg.in/amz.v1/aws"
	"gopkg.in/amz.v1/s3"
)

var cephConn *s3.S3

//GetCephConnection: 获取ceph连接
func GetCephConnection() *s3.S3 {
	if cephConn != nil {
		return cephConn
	}
	// 1. 初始化ceph的一些信息
	auth := aws.Auth{
		AccessKey: "EYDI419I4G8WOQE40JI0",
		SecretKey: "DYguzeWjX8fZWK1xDGBRwG6JqSVtylui4Mzb164N",
	}

	curRegion := aws.Region{
		Name:                 "default",
		EC2Endpoint:          "http://127.0.0.1:9080",
		S3Endpoint:           "http://127.0.0.1:9080",
		S3BucketEndpoint:     "",
		S3LocationConstraint: false,
		S3LowercaseBucket:    false,
		Sign:                 aws.SignV2,
	}

	// 2. 新建cephConn
	return s3.New(auth, curRegion)
}

// GetCephBucket:获取指定的bucket对象
func GetCephBucket(bucket string) *s3.Bucket {
	conn := GetCephConnection()
	return conn.Bucket(bucket)
}

// PutObject : 上传文件到ceph集群
func PutObject(bucket string, path string, data []byte) error {
	return GetCephBucket(bucket).Put(path, data, "octet-stream", s3.PublicRead)
}
