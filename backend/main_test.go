package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

func testChan(ch chan bool) {
	select {
	case <-ch:
		fmt.Println("get!")
	default:
		fmt.Println("empty!")
	}
}

func TestRoom(t *testing.T) {
	r := NewRoom(2 * time.Second)
	cli1 := NewClient()
	cli2 := NewClient()
	uuid1 := uuid.New()
	uuid2 := uuid.New()
	call1 := &clientInCall{
		Cli:  cli1,
		UUID: ClientUUID(uuid1),
	}
	call2 := &clientInCall{
		Cli:  cli2,
		UUID: ClientUUID(uuid2),
	}
	r.clientInSignal <- call1
	r.clientInSignal <- call2
	r.setQRCodeSignal <- "qrcode"
	time.Sleep(time.Second)
	testChan(cli1.QRCodeChangedSignal)
	testChan(cli2.QRCodeChangedSignal)
	r.clientOutSignal <- ClientUUID(uuid2)
	r.setQRCodeSignal <- "qrcode"
	time.Sleep(time.Second)
	testChan(cli1.QRCodeChangedSignal)
	testChan(cli2.QRCodeChangedSignal)
	ch := make(chan string)
	r.getQRCodeSignal <- ch
	s := <-ch
	fmt.Println(s)
	time.Sleep(3 * time.Second)
	testChan(cli1.CloseSignal)
	testChan(cli2.CloseSignal)
	<-r.ClosedSignal
}

func TestRoomPool(t *testing.T) {
	p := NewRoomPool()
	call1 := createRoomCall{
		Name:     "room1",
		Password: RoomPassword(1),
		Err:      make(chan error),
	}
	p.createRoomSignal <- &call1
	<-call1.Err
	e, _ := p.GetRoom("room1")
	fmt.Println(e.passwd)
	e.room.Close()
	time.Sleep(500 * time.Millisecond)
	_, err := p.GetRoom("room1")
	if err != nil {
		logErr(err, "room deleted")
	}
}
