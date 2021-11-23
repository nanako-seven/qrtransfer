package main

import (
	"errors"
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
	ClosedSignal    chan struct{}
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
loop:
	for {
		r.lastActiveTime = time.Now()
		select {
		case <-r.CloseSignal:
			for _, v := range r.clients {
				v.CloseSignal <- true
			}
			break loop
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
			break loop
		}
	}
	close(r.ClosedSignal)
}

type RoomPassword uint32

type CreateRoomCall struct {
	Name     string
	Password RoomPassword
	Err      chan error
}

type GetRoomCall struct {
	Name    string
	Element chan *RoomInfo
	Err     chan error
}

type RoomInfo struct {
	room   *Room
	passwd RoomPassword
}

type RoomPool struct {
	pool             map[string]*RoomInfo
	CreateRoomSignal chan *CreateRoomCall
	deleteRoomSignal chan string
	GetRoomSignal    chan *GetRoomCall
}

func NewRoomPool() *RoomPool {
	pool := &RoomPool{
		pool:             make(map[string]*RoomInfo),
		CreateRoomSignal: make(chan *CreateRoomCall),
		deleteRoomSignal: make(chan string),
		GetRoomSignal:    make(chan *GetRoomCall),
	}
	pool.run()
	return pool
}

func (p *RoomPool) monitorRoom(name string) {
	room := p.pool[name]
	<-room.room.ClosedSignal
	p.deleteRoomSignal <- name
}

func (p *RoomPool) run() {
	for {
		select {
		case x := <-p.CreateRoomSignal:
			_, ok := p.pool[x.Name]
			if ok {
				x.Err <- errors.New("room name already exists")
				break
			}
			p.pool[x.Name] = &RoomInfo{
				room:   NewRoom(DefaultMaxTime),
				passwd: x.Password,
			}
			x.Err <- nil
			go p.monitorRoom(x.Name)
		case x := <-p.deleteRoomSignal:
			delete(p.pool, x)
		case x := <-p.GetRoomSignal:
			room, ok := p.pool[x.Name]
			if !ok {
				x.Err <- errors.New("could not get a non-exist room")
				break
			}
			x.Element <- room
		}
	}
}

type Client struct {
	CloseSignal         chan bool
	QRCodeChangedSignal chan bool
}
