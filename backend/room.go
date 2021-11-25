package main

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const DefaultMaxTime = 2 * time.Minute

type ClientUUID uuid.UUID

type clientInCall struct {
	Cli  *Client
	UUID ClientUUID
}

type Room struct {
	lastActiveTime  time.Time
	maxTime         time.Duration
	qrCode          string
	clients         map[ClientUUID]*Client
	closeSignal     chan struct{}
	clientInSignal  chan *clientInCall
	clientOutSignal chan ClientUUID
	setQRCodeSignal chan string
	getQRCodeSignal chan chan string
	ClosedSignal    chan struct{}
}

func NewRoom(maxTimeAllowed time.Duration) *Room {
	room := &Room{
		lastActiveTime:  time.Now(),
		maxTime:         maxTimeAllowed,
		clients:         make(map[ClientUUID]*Client),
		closeSignal:     make(chan struct{}),
		clientInSignal:  make(chan *clientInCall),
		clientOutSignal: make(chan ClientUUID),
		setQRCodeSignal: make(chan string),
		getQRCodeSignal: make(chan chan string),
		ClosedSignal:    make(chan struct{}),
	}
	go room.run()
	return room
}

func (r *Room) run() {
loop:
	for {
		if time.Since(r.lastActiveTime) > r.maxTime {
			break loop
		}
		noSignal := false
		select {
		case <-r.closeSignal:
			break loop
		case x := <-r.clientInSignal:
			r.clients[x.UUID] = x.Cli
		case x := <-r.clientOutSignal:
			delete(r.clients, x)
		case x := <-r.setQRCodeSignal:
			r.qrCode = x
			for _, v := range r.clients {
				v.QRCodeChangedSignal <- true
			}
		case x := <-r.getQRCodeSignal:
			x <- r.qrCode
		default:
			noSignal = true
		}
		if !noSignal {
			r.lastActiveTime = time.Now()
		}
	}
	for _, v := range r.clients {
		v.CloseSignal <- true
	}
	close(r.ClosedSignal)
}
func (r *Room) Close() {
	r.closeSignal <- struct{}{}
}

func (r *Room) ClientIn(cli *Client, uuid ClientUUID) {
	r.clientInSignal <- &clientInCall{
		Cli:  cli,
		UUID: uuid,
	}
}

func (r *Room) ClientOut(uuid ClientUUID) {
	r.clientOutSignal <- uuid
}

func (r *Room) SetQRCode(code string) {
	r.setQRCodeSignal <- code
}

func (r *Room) GetQRCode() string {
	ch := make(chan string)
	r.getQRCodeSignal <- ch
	return <-ch
}

type RoomPassword uint32

type createRoomCall struct {
	Name     string
	Password RoomPassword
	Err      chan error
}

type getRoomRetVal struct {
	Element *RoomInfo
	Err     error
}

type getRoomCall struct {
	Name string
	Ret  chan *getRoomRetVal
}

type RoomInfo struct {
	room   *Room
	passwd RoomPassword
}

type RoomPool struct {
	pool             map[string]*RoomInfo
	createRoomSignal chan *createRoomCall
	deleteRoomSignal chan string
	getRoomSignal    chan *getRoomCall
}

func NewRoomPool() *RoomPool {
	pool := &RoomPool{
		pool:             make(map[string]*RoomInfo),
		createRoomSignal: make(chan *createRoomCall),
		deleteRoomSignal: make(chan string),
		getRoomSignal:    make(chan *getRoomCall),
	}
	go pool.run()
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
		case x := <-p.createRoomSignal:
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
		case x := <-p.getRoomSignal:
			room, ok := p.pool[x.Name]
			if !ok {
				err := errors.New("could not get a non-exist room")
				x.Ret <- &getRoomRetVal{
					Element: nil,
					Err:     err,
				}
				break
			}
			x.Ret <- &getRoomRetVal{
				Element: room,
				Err:     nil,
			}
		}
	}
}

func (p *RoomPool) GetRoom(name string) (*RoomInfo, error) {
	call := &getRoomCall{
		Name: name,
		Ret:  make(chan *getRoomRetVal),
	}
	p.getRoomSignal <- call
	r := <-call.Ret
	return r.Element, r.Err
}

func (p *RoomPool) CreateRoom(name string, passwd RoomPassword) error {
	call := &createRoomCall{
		Name:     name,
		Password: passwd,
		Err:      make(chan error),
	}
	p.createRoomSignal <- call
	err := <-call.Err
	return err
}

type Client struct {
	CloseSignal         chan bool
	QRCodeChangedSignal chan bool
}

func NewClient() *Client {
	return &Client{
		CloseSignal:         make(chan bool),
		QRCodeChangedSignal: make(chan bool),
	}
}
