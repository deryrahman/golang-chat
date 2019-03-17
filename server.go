package main

import (
	"encoding/binary"
	"log"
	"net"
)

// ServerMode : run on server mode
func ServerMode(host string) {
	listener, err := net.Listen("tcp", host)
	checkError(err)
	log.Println("Server listen on", host)

	clientManager := ClientManager{
		ClientsRoom: make(map[int]map[*Client]bool, 8),
		Register:    make(chan *Client),
		Deregister:  make(chan *Client),
	}

	// start client manager
	go clientManager.start()

	// listen if any client join
	for {
		conn, err := listener.Accept()
		checkError(err)

		client := &Client{
			Conn: conn,
		}
		clientManager.prepareClient(client)

		clientManager.Register <- client

		go clientManager.serveClient(client)
	}
}

// PrepareClient : prepare client to fetch roomID
func (clientManager *ClientManager) prepareClient(client *Client) {
	// Get roomID first
	bs := make([]byte, 4)
	_, err := client.Conn.Read(bs)
	checkError(err)
	roomID := int(binary.LittleEndian.Uint16(bs))
	client.RoomID = roomID
	log.Printf("Client %v join\n", *client)
}

// Start : Start client manager for register and deregister any client
func (clientManager *ClientManager) start() {
	for {
		select {
		case client := <-clientManager.Register:
			log.Printf("Register client %v\n", *client)
			if _, ok := clientManager.ClientsRoom[client.RoomID]; !ok {
				clientManager.ClientsRoom[client.RoomID] = make(map[*Client]bool, 4)
			}
			clientManager.ClientsRoom[client.RoomID][client] = true
		case client := <-clientManager.Deregister:
			log.Printf("Deregister client %v\n", *client)
			client.Conn.Close()
			delete(clientManager.ClientsRoom[client.RoomID], client)
		}
	}
}

// ServeClient : Serving for incoming message from client and forward it on all clients with same roomID
func (clientManager *ClientManager) serveClient(client *Client) {
	log.Printf("Serving client %v\n", *client)
	for {
		buff := make([]byte, 4096)
		_, err := client.Conn.Read(buff)
		if err != nil {
			clientManager.Deregister <- client
			break
		}

		// Broadcast to every client in roomID
		for anotherClient := range clientManager.ClientsRoom[client.RoomID] {
			if client != anotherClient {
				anotherClient.Conn.Write(buff)
			}
		}
	}
}
