package main

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed dist
var dist embed.FS

type fsWithPrefix struct {
	Fs embed.FS
}

func (f *fsWithPrefix) Open(p string) (fs.File, error) {
	return f.Fs.Open(path.Join("dist", p))
}

func testFunc(server *Server) {
	clock := time.Tick(time.Second)
	for {
		<-clock
		info, _ := server.roomPool.GetRoom("room1")
		info.room.SetQRCode("qrcode")
	}
}

func main() {
	r := gin.Default()
	server := NewServer()
	server.roomPool.CreateRoom("room1", RoomPassword(1))
	r.GET("/create-room", server.CreateRoomHandler)
	r.GET("/delete-room", server.DeleteRoomHandler)
	r.GET("/connect-room", server.ConnectRoomHandler)
	r.POST("/update-qr-code", server.UpdateQRCodeHanlder)
	r.StaticFS("/ui", http.FS(&fsWithPrefix{dist}))
	r.GET("/", RedirectMainPage)
	r.Run("0.0.0.0:8888")
}
