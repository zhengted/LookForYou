package main

import "LookForYou/route"

func main() {
	r := route.Router()
	r.Run(":8080")
}
