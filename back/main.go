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
	server := &Server{}
	r.GET("/create-room", server.CreateRoomHandler)
	r.GET("/delete-room", server.DeleteRoomHandler)
	r.GET("/connect-room", server.ConnectRoomHandler)
	r.POST("/update-qr-code", server.UpdateQRCodeHanlder)
	r.StaticFS("/ui", http.FS(&fsWithPrefix{dist}))
	r.GET("/", RedirectMainPage)
	r.Run("0.0.0.0:8888")
}
