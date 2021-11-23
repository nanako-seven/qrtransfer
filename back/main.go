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

var rooms RoomPool

func main() {
	const maxTime = time.Minute
	room := NewRoom(maxTime)
	go func() {
		ticker := time.Tick(600 * time.Second)
		for {
			<-ticker
			roomOld := room
			room = NewRoom(maxTime)
			roomOld.CloseSignal <- true
		}
	}()
	r := gin.Default()
	r.GET("/create-room", CreateRoomHandler)
	r.GET("/delete-room", DeleteRoomHandler)
	r.GET("/get-rooms", GetRoomsHandler)
	r.GET("/connect-room", ConnectRoomHandler)
	r.POST("/update-qr-code", UpdateQRCodeHanlder)
	r.GET("/get-qr-code", GetQRCodeHandler)
	r.StaticFS("/ui", http.FS(&fsWithPrefix{dist}))
	r.GET("/", RedirectMainPage)
	r.Run("0.0.0.0:8888")
}
