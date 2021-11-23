package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
)


type CreateRoomQuery struct {
	RoomId string
}

func CreateRoomHandler(c *gin.Context) {
	query := CreateRoomQuery{}
	err := c.ShouldBind(&query)
	if err != nil {
		logErr(err, "invalid query in CreateRoomHandler")
	}
	id := query.RoomId
	rooms.lock.Lock()
	defer rooms.lock.Unlock()
	_, ok := rooms.pool[id]
	if ok {
		logErr(nil, "room id already exists")
		c.String(http.StatusBadRequest, "")
		return
	}
	rooms.pool[id] = &Room{}
	c.String(http.StatusOK, "")
}

type DeleteRoomQuery struct {
	RoomId string
}

func DeleteRoomHandler(c *gin.Context) {
	query := DeleteRoomQuery{}
	err := c.ShouldBind(&query)
	if err != nil {
		logErr(err, "invalid query in DeleteRoomHandler")
	}
	id := query.RoomId
	rooms.lock.Lock()
	defer rooms.lock.Unlock()
	_, ok := rooms.pool[id]
	if !ok {
		logErr(nil, "cannot delete non-exist room")
		c.String(http.StatusBadRequest, "")
		return
	}
	rooms.pool[id].CloseRoom()
	delete(rooms.pool, id)
	c.String(http.StatusOK, "")
}

func GetRoomsHandler(c *gin.Context) {
	roomList := make([]string, 0)
	for k, _ := range rooms.pool {
		roomList = append(roomList, k)
	}
	c.JSON(http.StatusOK, gin.H{"rooms": roomList})
}

func RedirectMainPage(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "ui")
}

func UpdateQRCodeHanlder(c *gin.Context) {
	type req struct {
		QR string `json:"qr"`
	}
	var r req
	err := c.BindJSON(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{})
	}
	roomLock.Lock()
	room.UpdateQRCode(r.QR)
	roomLock.Unlock()
	c.JSON(200, gin.H{})
}

func GetQRCodeHandler(c *gin.Context) {
	if !c.IsWebsocket() {
		c.JSON(http.StatusBadRequest, gin.H{})
		return
	}
	websocket.Handler(func(conn *websocket.Conn) {
		defer conn.Close()
		signal := make(chan (int))
		report := make(chan (error))

		roomLock.Lock()
		_, err := conn.Write([]byte(room.QR))
		if err != nil {
			roomLock.Unlock()
			return
		}
		room.AddClient(&Client{CloseSignal: signal, Report: report, Alive: true})
		log.Println("adding client")
		roomLock.Unlock()
		for {
			s := <-signal
			if s == 1 {
				log.Println("closing client")
				return
			}
			_, err := conn.Write([]byte(room.QR))
			report <- err
			if err != nil {
				return
			}
		}
	}).ServeHTTP(c.Writer, c.Request)
}
