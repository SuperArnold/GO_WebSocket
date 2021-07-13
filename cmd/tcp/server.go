package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

type User struct {
	ID             int
	Addr           string
	EnterAt        time.Time
	MessageChannel chan string
}

var (
	globalID int
	idocker  sync.Mutex
	//新使用者進來
	enteringChannel = make(chan *User)
	//使用者離開
	leavingChannel = make(chan *User)
	//傳送訊息
	messageChannel = make(chan string, 8)
)

func main() {
	listener, err := net.Listen("tcp", ":2020")
	if err != nil {
		panic(err)
	}

	go broadcaster()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go handleConn(conn)
	}
}

//boradcaster 用於記錄聊天室的使用者，並進行訊息廣播
//1.新使用者進來; 2.使用者普通訊息; 3.使用者離開
func broadcaster() {
	users := make(map[*User]struct{})

	for {
		select {
		case user := <-enteringChannel:
			//新使用者進來
			users[user] = struct{}{}
		case user := <-leavingChannel:
			//使用者離開
			delete(users, user)
			close(user.MessageChannel)
		case msg := <-messageChannel:
			//給所有人發送訊息
			for user := range users {
				user.MessageChannel <- msg
			}
		}
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	// 1. 新使用者進來
	user := &User{
		ID:             GenUserID(),
		Addr:           conn.RemoteAddr().String(),
		EnterAt:        time.Now(),
		MessageChannel: make(chan string, 8),
	}

	// 2.
	go sendMessage(conn, user.MessageChannel)

	// 3.
	user.MessageChannel <- "Welcome, " + strconv.Itoa(user.ID)
	messageChannel <- "user:`" + strconv.Itoa(user.ID) + "` has enter"

	// 4.
	enteringChannel <- user
	// 5.
	input := bufio.NewScanner(conn)
	for input.Scan() {
		messageChannel <- strconv.Itoa(user.ID) + ":" + input.Text()
	}

	if err := input.Err(); err != nil {
		log.Println("讀取錯誤: ", err)
	}

	// 6.
	leavingChannel <- user
	messageChannel <- "user:`" + strconv.Itoa(user.ID) + "` has left"

}

func sendMessage(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintln(conn, msg)
	}
}

func GenUserID() int {
	idocker.Lock()
	defer idocker.Unlock()

	globalID++
	return globalID
}
