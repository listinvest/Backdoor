package tcp

import (
	"bytes"
	"errors"
	"net"
	"strconv"
	"sync"
	"time"
)

//ConnectionHandler contains essential information for handling a connection
type ConnectionHandler struct {
	connection    net.Conn
	msgBuffer     bytes.Buffer
	counterBuffer bytes.Buffer
	mux           sync.Mutex
	dead          bool
}

//Listen listens to the port number entered as input and accepts incoming connections
//It returns a channel bywhich passes the connection handler objects and boolean variable which states
//whether the program is listenening or not.
//this bool variable can be set to false in order to stop the subrutines from listening
func Listen(port int) (chan *ConnectionHandler, *bool, error) {
	var err error
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return nil, nil, errors.New("The listen operation failed")
	}
	ch := make(chan *ConnectionHandler, 16)
	listening := true
	go func() {
		for listening {
			conn, err := ln.Accept()
			if err != nil {
				err = errors.New("The server failed to accept new connection")
				listening = false
			} else {
				var n int64
				var hndl ConnectionHandler
				hndl.msgBuffer.Grow(1024)
				hndl.counterBuffer.Grow(8)
				hndl.connection = conn
				go func() {
					for err == nil {
						hndl.mux.Lock()
						n, err = hndl.msgBuffer.ReadFrom(hndl.connection)
						if err == nil {
							hndl.counterBuffer.WriteByte(byte(n))
						}
						hndl.mux.Unlock()
					}
					for !hndl.BufferIsEmpty() {
						time.Sleep(15 * time.Millisecond)
					}
					hndl.dead = true
				}()
				ch <- &hndl
			}
		}
		ln.Close()
	}()
	return ch, &listening, nil
}

//Call calls a remote address over tcp protocol
func Call(target string) (*ConnectionHandler, error) {
	var hndl ConnectionHandler
	hndl.msgBuffer.Grow(1024)
	hndl.counterBuffer.Grow(8)
	var n int64
	var err error
	conn, err := net.Dial("tcp", target)
	if err != nil {
		return nil, errors.New("The dial to the specified address was unsuccessful")
	}
	hndl.connection = conn
	go func() {
		for err == nil {
			hndl.mux.Lock()
			n, err = hndl.msgBuffer.ReadFrom(hndl.connection)
			if err == nil {
				hndl.counterBuffer.WriteByte(byte(n))
			}
			hndl.mux.Unlock()
		}
		for !hndl.BufferIsEmpty() {
			time.Sleep(15 * time.Millisecond)
		}
		hndl.dead = true
	}()
	return &hndl, nil
}

//BufferIsEmpty determines if the buffer of the connection is empty
func (hndl *ConnectionHandler) BufferIsEmpty() bool {
	return hndl.counterBuffer.Len() == 0
}

//Read reads from the buffer of connection in a first in first out order
func (hndl *ConnectionHandler) Read() ([]byte, error) {
	if hndl.dead == true {
		return nil, errors.New("This connection is dead")
	}
	hndl.mux.Lock()
	defer hndl.mux.Unlock()
	if hndl.BufferIsEmpty() {
		return nil, errors.New("The connection buffer is empty hence no message is available")
	}
	length, err := hndl.counterBuffer.ReadByte()
	if err != nil {
		return nil, err
	}
	buff := make([]byte, length)
	n, err := hndl.counterBuffer.Read(buff)
	if err != nil {
		return nil, err
	}
	if n != int(length) {
		return nil, errors.New("The lenght of the byte read from the buffer is not as much as expected")
	}
	return buff, nil
}
