package main

import (
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Server struct {
	roomPool RoomPool
}

type CreateRoomQuery struct {
	RoomId string
}

func (s *Server) CreateRoomHandler(c *gin.Context) {
	query := CreateRoomQuery{}
	err := c.ShouldBind(&query)
	if err != nil {
		logErr(err, "invalid query in CreateRoomHandler")
		c.String(http.StatusBadRequest, "")
		return
	}
	id := query.RoomId
	passwd := RoomPassword(rand.Uint32())
	ch := make(chan error)
	s.roomPool.createRoomSignal <- &createRoomCall{
		Name:     id,
		Password: passwd,
		Err:      ch,
	}
	err = <-ch
	if err != nil {
		logErr(err, "could not create room")
		c.String(http.StatusBadRequest, "")
		return
	}
	c.JSON(http.StatusOK, gin.H{"password": passwd})
}

type DeleteRoomQuery struct {
	RoomId   string
	Password RoomPassword
}

func (s *Server) getRoom(id string, passwd RoomPassword) (*RoomInfo, error) {
	ele, err := s.roomPool.GetRoom(id)
	if err != nil {
		return nil, err
	}
	if passwd != ele.passwd {
		return nil, errors.New("wrong password")
	}
	return ele, nil
}

func (s *Server) DeleteRoomHandler(c *gin.Context) {
	query := DeleteRoomQuery{}
	err := c.ShouldBind(&query)
	if err != nil {
		logErr(err, "invalid query in DeleteRoomHandler")
		c.String(http.StatusBadRequest, "")
		return
	}
	id := query.RoomId
	passwd := query.Password
	ele, err := s.getRoom(id, passwd)
	if err != nil {
		logErr(err, "could not delete room")
		c.String(http.StatusBadRequest, "")
		return
	}
	ele.room.Close()
	c.String(http.StatusOK, "")
}

func RedirectMainPage(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "ui")
}

type UpdateQRCodeQuery struct {
	RoomId   string       `json:"roomId"`
	QRCode   string       `json:"qr"`
	Password RoomPassword `json:"password"`
}

func (s *Server) UpdateQRCodeHanlder(c *gin.Context) {
	query := UpdateQRCodeQuery{}
	err := c.BindJSON(&query)
	if err != nil {
		logErr(err, "invalid query in UpdateQRCodeHanlder")
		c.JSON(http.StatusBadRequest, gin.H{})
		return
	}
	id := query.RoomId
	passwd := query.Password
	ele, err := s.getRoom(id, passwd)
	if err != nil {
		logErr(err, "could not update QR code")
		c.String(http.StatusBadRequest, "")
		return
	}
	ele.room.setQRCodeSignal <- query.QRCode
	c.JSON(http.StatusOK, gin.H{})
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type ConnectRequest struct {
	RoomId string
}

const (
	ConnectOK = iota
	ConnectRefused
	QRCodeChanged
	ConnectClose
)

type ConnectResponse struct {
	Type    int
	Message string
}

func (s *Server) ConnectRoomHandler(c *gin.Context) {
	if !c.IsWebsocket() {
		c.JSON(http.StatusBadRequest, gin.H{})
		return
	}
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logErr(err, "could not upgrade to websocket")
		return
	}
	defer ws.Close()
	mt, message, err := ws.ReadMessage()
	if err != nil {
		logErr(err, "could not read message")
		return
	}
	req := ConnectRequest{}
	err = json.Unmarshal(message, &req)
	if err != nil {
		logErr(err, "invalid request")
		return
	}
	roomInfo, _ := s.roomPool.GetRoom(req.RoomId)
	room := roomInfo.room
	id := uuid.New()
	cli := NewClient()
	call2 := &clientInCall{
		Cli:  cli,
		UUID: ClientUUID(id),
	}
	room.clientInSignal <- call2
	defer func() {
		room.clientOutSignal <- ClientUUID(id)
	}()
	msg, _ := json.Marshal(&ConnectResponse{
		Type: ConnectOK,
	})
	err = ws.WriteMessage(mt, msg)
	if err != nil {
		logErr(err, "could not write message")
		return
	}
	for {
		select {
		case <-cli.QRCodeChangedSignal:
			ch := make(chan string)
			room.getQRCodeSignal <- ch
			code := <-ch
			msg, _ := json.Marshal(&ConnectResponse{
				Type:    QRCodeChanged,
				Message: code,
			})
			err = ws.WriteMessage(mt, msg)
			if err != nil {
				logErr(err, "could not write message")
				return
			}
		case <-cli.CloseSignal:
			msg, _ := json.Marshal(&ConnectResponse{
				Type: ConnectClose,
			})
			err = ws.WriteMessage(mt, msg)
			if err != nil {
				logErr(err, "could not write message")
			}
			return
		}
	}
}
