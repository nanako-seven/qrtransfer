package main

import (
	"errors"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

const DefaultMaxTime = 2 * time.Minute

type ClientUUID uuid.UUID

type ClientInCall struct {
	Cli  *Client
	UUID ClientUUID
}

type Room struct {
	lastActiveTime  time.Time
	maxTime         time.Duration
	qrCode          string
	clients         map[ClientUUID]*Client
	CloseSignal     chan bool
	ClientInSignal  chan *ClientInCall
	ClientOutSignal chan ClientUUID
	SetQRCodeSignal chan string
	GetQRCodeSignal chan chan string
}

func NewRoom(maxTimeAllowed time.Duration) *Room {
	room := &Room{
		maxTime:         maxTimeAllowed,
		clients:         make(map[ClientUUID]*Client),
		CloseSignal:     make(chan bool),
		ClientInSignal:  make(chan *ClientInCall),
		ClientOutSignal: make(chan ClientUUID),
		SetQRCodeSignal: make(chan string),
		GetQRCodeSignal: make(chan chan string),
	}
	room.run()
	return room
}

func (r *Room) run() {
	for {
		r.lastActiveTime = time.Now()
		select {
		case <-r.CloseSignal:
			for _, v := range r.clients {
				v.CloseSignal <- true
			}
			return
		case x := <-r.ClientInSignal:
			r.clients[x.UUID] = x.Cli
		case x := <-r.ClientOutSignal:
			delete(r.clients, x)
		case x := <-r.SetQRCodeSignal:
			r.qrCode = x
			for _, v := range r.clients {
				v.QRCodeChangedSignal <- true
			}
		case x := <-r.GetQRCodeSignal:
			x <- r.qrCode
		}
		if time.Since(r.lastActiveTime) > r.maxTime {
			return
		}
	}
}

type RoomPassword uint32

type CreateRoomCall struct {
	Name   string
	Passwd chan RoomPassword
	Err    chan error
}

type DeleteRoomCall struct {
	Name   string
	Passwd RoomPassword
	Err    chan error
}

type RoomInfo struct {
	room   *Room
	passwd RoomPassword
}

type RoomPool struct {
	pool             map[string]*RoomInfo
	CreateRoomSignal chan *CreateRoomCall
	DeleteRoomSignal chan *DeleteRoomCall
	GetRoomsSignal   chan chan []string
}

func NewRoomPool() *RoomPool {
	pool := &RoomPool{
		pool:             make(map[string]*RoomInfo),
		CreateRoomSignal: make(chan *CreateRoomCall),
		DeleteRoomSignal: make(chan *DeleteRoomCall),
		GetRoomsSignal:   make(chan chan []string),
	}
	pool.run()
	return pool
}

func (p *RoomPool) run() {
	for {
		select {
		case x := <-p.CreateRoomSignal:
			_, ok := p.pool[x.Name]
			if ok {
				x.Passwd <- 0
				x.Err <- errors.New("room name already exists")
				break
			}
			passwd := RoomPassword(rand.Uint32())
			p.pool[x.Name] = &RoomInfo{
				room:   NewRoom(DefaultMaxTime),
				passwd: passwd,
			}
			x.Passwd <- passwd
			x.Err <- nil
		case x := <-p.DeleteRoomSignal:
			room, ok := p.pool[x.Name]
			if !ok {
				x.Err <- errors.New("could not delete a non-exist room")
				break
			}
			if room.passwd != x.Passwd {
				x.Err <- errors.New("wrong password")
				break
			}
			room.room.CloseSignal <- true
			delete(p.pool, x.Name)
			x.Err <- nil
		case x := <-p.GetRoomsSignal:
			var names []string
			for k, _ := range p.pool {
				names = append(names, k)
			}
			x <- names
		}
	}
}

type Client struct {
	CloseSignal         chan bool
	QRCodeChangedSignal chan bool
}
