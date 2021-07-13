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

type Message struct {
	OwnerID int
	Content string
}

var (
	globalID int
	idocker  sync.Mutex
	//新使用者進來
	enteringChannel = make(chan *User)
	//使用者離開
	leavingChannel = make(chan *User)
	//傳送訊息
	messageChannel = make(chan *Message)
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
				if user.ID == msg.OwnerID {
					continue
				}
				user.MessageChannel <- msg.Content
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
	userMessage := &Message{
		OwnerID: user.ID,
		Content: "",
	}

	// 2.
	go sendMessage(conn, user.MessageChannel)

	// 3.
	user.MessageChannel <- "Welcome, " + strconv.Itoa(user.ID)
	userMessage.Content = "user:`" + strconv.Itoa(user.ID) + "` has enter"
	messageChannel <- userMessage

	// 4.
	enteringChannel <- user

	//閒置使用者
	var userActive = make(chan struct{})
	go func() {
		d := 15 * time.Second
		timer := time.NewTimer(d)
		for {
			select {
			case <-timer.C:
				conn.Close()
			case <-userActive:
				timer.Reset(d)
			}
		}
	}()

	// 5.
	input := bufio.NewScanner(conn)
	for input.Scan() {
		userMessage.Content = strconv.Itoa(user.ID) + ":" + input.Text()
		messageChannel <- userMessage

		//使用者活著
		userActive <- struct{}{}
	}

	if err := input.Err(); err != nil {
		log.Println("讀取錯誤: ", err)
	}

	// 6.
	leavingChannel <- user
	userMessage.Content = "user:`" + strconv.Itoa(user.ID) + "` has left"
	messageChannel <- userMessage

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
