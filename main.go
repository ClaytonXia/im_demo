package main

import "github.com/claytonxia/im_demo/server"

func main() {
	srv := server.NewIMServer()
	srv.Start()
}
