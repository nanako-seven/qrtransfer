package main

import (
	"errors"
	"log"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
)

type Server struct {
	rooms RoomPool
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
	s.rooms.CreateRoomSignal <- &CreateRoomCall{
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
	errCh := make(chan error)
	eleCh := make(chan *RoomInfo)
	s.rooms.GetRoomSignal <- &GetRoomCall{
		Name:    id,
		Element: eleCh,
		Err:     errCh,
	}
	ele := <-eleCh
	err := <-errCh
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
	ele.room.CloseSignal <- true
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
	ele.room.SetQRCodeSignal <- query.QRCode
	c.JSON(http.StatusOK, gin.H{})
}

// websocket 不是很会，我得学习一下
func (s *Server) ConnectRoomHandler(c *gin.Context) {
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
