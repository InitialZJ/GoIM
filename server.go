package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip   string
	Port int

	OnlineMap map[string]*User
	mapLock   sync.RWMutex

	Message chan string
}

func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}
	return server
}

func (this *Server) ListenMessager() {
	for {
		msg := <-this.Message
		this.mapLock.Lock()
		for _, cli := range this.OnlineMap {
			cli.C <- msg
		}
		this.mapLock.Unlock()
	}
}

func (this *Server) BroadCast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg
	this.Message <- sendMsg
}

func (this *Server) Handler(conn net.Conn) {
	// fmt.Println("链接建立成功")

	user := NewUser(conn, this)
	user.Online()

	isLive := make(chan bool)

	go func() {
		buf := make([]byte, 4096)
		for {
			conn.Read(buf)
			for {
				n, err := conn.Read(buf)
				if n == 0 {
					user.Offline()
					return
				}

				if err != nil && err != io.EOF {
					fmt.Println("Conn Read err:", err)
					return
				}

				msg := string(buf[:n-1])
				user.DoMessage(msg)

				isLive <- true
			}
		}
	}()

	for {
		select {
		case <-isLive:
			// 活跃
		case <-time.After(time.Second * 100):
			// 超时
			user.SendMsg("你被踢了")
			close(user.C)
			conn.Close()
			return
		}
	}
}

func (this *Server) Start() {
	listner, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port))
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}
	defer listner.Close()

	go this.ListenMessager()

	for {
		conn, err := listner.Accept()
		if err != nil {
			fmt.Println("listner accept err:", err)
			continue
		}

		go this.Handler(conn)
	}
}
