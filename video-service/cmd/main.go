package main

import "video-service/router"

func main() {
	r := router.Setup()
	
	r.Run(":8080")
}
