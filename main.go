package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var connections = []*websocket.Conn{}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	CheckOrigin: func(w *http.Request) bool {
		return true
	},
}

type user struct {
	name string
	pass string
}

var users = []user{
	{"nikita", "12345"},
	{"elya", "67890"},
}

func authUser(conn *websocket.Conn, users []user) bool {
	var login, password []byte
	err := conn.WriteMessage(websocket.TextMessage, []byte("Введите имя пользователя"))
	if err != nil {
		log.Println("Error while reading")
		return false
	}
	_, login, err = conn.ReadMessage()
	if err != nil {
		log.Println("Error while reading")
		return false
	}
	err = conn.WriteMessage(websocket.TextMessage, []byte("Введите пароль"))
	if err != nil {
		log.Println("Error while reading")
		return false
	}
	_, password, err = conn.ReadMessage()
	if err != nil {
		log.Println("Error while reading")
		return false
	}
	for _, val := range users {
		if val.name == string(login) {
			if val.pass == string(password) {
				return true
			}
		}
	}
	return false
}

func removeConnection(q []*websocket.Conn, conn *websocket.Conn) []*websocket.Conn {
	var res []*websocket.Conn
	for _, c := range q {
		if c != conn {
			res = append(res, c)
		}
	}
	log.Println("Connection Closed", len(res))
	conn.Close()
	return res
}

func handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Not websocket")
		return
	}
	connections = append(connections, conn)
	log.Println("Client connected", len(connections))
	for {
		for {
			res := authUser(conn, users)
			if res {
				break
			}
		}
		messageType, r, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error while reading message!")
			connections = removeConnection(connections, conn)
			return
		} else {
			log.Println("Message handled")
		}
		for _, c := range connections {
			if c != conn {
				err := c.WriteMessage(messageType, r)
				if err != nil {
					log.Println("Error while writing message!")
					connections = removeConnection(connections, conn)
					return
				}
			}
		}
	}
}

func main() {
	http.HandleFunc("/ws", handler)
	log.Println("Server started at port 8000")
	http.ListenAndServe(":8000", nil)
}
