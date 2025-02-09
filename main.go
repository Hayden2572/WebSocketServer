package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
)

var connections = map[string]*websocket.Conn{}

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

var users = []user{}

func connectDB() []user {
	connStr := "user=sxneatxr password=root port=5432 host=localhost dbname=chatInfo sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Println("Error while connecting database")
	}
	log.Println("Succesfully connected to database")
	rows, err := db.Query("SELECT * FROM public.users")
	if err != nil {
		log.Println("Error while reading data from database")
	}
	users := make([]user, 0)
	for rows.Next() {
		var id int
		var name, pass string
		err = rows.Scan(&id, &name, &pass)
		if err != nil {
			log.Println("Error while reading row")
		}
		users = append(users, user{name, pass})
	}
	return users
}

func connectionLimit(conn *websocket.Conn, connections map[string]*websocket.Conn, n int) bool {
	if len(connections) >= n {
		conn.WriteMessage(websocket.TextMessage, []byte("Достигнут лимит на количество подключенных пользователей, иди нахуй"))
		conn.Close()
		return false
	} else {
		return true
	}
}

type auth struct {
	result bool
	name   string
}

func authUser(conn *websocket.Conn, users []user) auth {
	var login, password []byte
	err := conn.WriteMessage(websocket.TextMessage, []byte("Введите имя пользователя"))
	if err != nil {
		log.Println("Error while reading")
		return auth{false, ""}
	}
	_, login, err = conn.ReadMessage()
	if err != nil {
		log.Println("Error while reading")
		return auth{false, ""}
	}
	err = conn.WriteMessage(websocket.TextMessage, []byte("Введите пароль"))
	if err != nil {
		log.Println("Error while reading")
		return auth{false, ""}
	}
	_, password, err = conn.ReadMessage()
	if err != nil {
		log.Println("Error while reading")
		return auth{false, ""}
	}
	for _, val := range users {
		if val.name == string(login) {
			if val.pass == string(password) {
				conn.WriteMessage(websocket.TextMessage, []byte("Добро пожаловать "+string(login)))
				return auth{true, val.name}
			}
		}
	}
	conn.WriteMessage(websocket.TextMessage, []byte("Неправильный логин или пароль"))
	return auth{false, ""}
}

func removeConnection(q map[string]*websocket.Conn, conn *websocket.Conn) map[string]*websocket.Conn {
	for name, c := range q {
		if c == conn {
			delete(q, name)
		}
	}
	log.Println("Connection Closed", len(q))
	err := conn.Close()
	if err != nil {
		log.Println("Error while closing websocket")
	}
	return q
}

func handler(w http.ResponseWriter, r *http.Request) {
	//upgrade to WebSocket
	users = connectDB()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Not websocket")
		return
	}
	//check for empty places
	pass := connectionLimit(conn, connections, 4)
	if !pass {
		return
	}
	//auth process
	for {
		if err = conn.WriteMessage(websocket.PingMessage, []byte("")); err != nil {
			return
		}
		res := authUser(conn, users)
		if res.result {
			connections[res.name] = conn
			break
		}
	}
	//websocket listening
	for {
		err := conn.WriteMessage(websocket.PingMessage, []byte(""))
		if err != nil {
			connections = removeConnection(connections, conn)
		}
		messageType, r, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error while reading message!")
			connections = removeConnection(connections, conn)
			return
		} else {
			log.Println("Message handled")
		}
		answer := ""
		for name, c := range connections {
			if c == conn {
				answer = name + ": " + string(r)
				break
			}
		}
		for name, c := range connections {
			if c != conn {
				log.Println(name)
				err := c.WriteMessage(messageType, []byte(answer))
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
