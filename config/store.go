package config

import cmn "LookForYou/common"

const (
	// TempLocalRootDir : 本地临时存储地址的路径
	TempLocalRootDir = "/home/ubuntu/FileStoreData/tmp/"
	// MergeLocalRootDir : 本地存储地址的路径(包含普通上传及分块上传)
	MergeLocalRootDir = "/home/ubuntu/FileStoreData/merge/"
	// CephRootDir : Ceph的存储路径prefix
	CephRootDir = "/ceph"
	// ChunckLocalRootDir : 分块存储地址的路径
	ChunckLocalRootDir = "/home/ubuntu/FileStoreData/chunks/"
	// OSSRootDir : OSS的存储路径prefix
	OSSRootDir = "oss/"
	// CurrentStoreType : 设置当前文件的存储类型
	CurrentStoreType = cmn.StoreOSS
)
