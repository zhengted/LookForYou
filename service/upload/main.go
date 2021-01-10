package main

import (
	cfg "LookForYou/config"
	"LookForYou/route"
)

func main() {
	router := route.Router()
	router.Run(cfg.UploadServiceHost)
}
