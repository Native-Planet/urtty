package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os/exec"
	"os/user"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"github.com/stevelacy/go-urbit/noun"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal("WebSocket Upgrade error:", err)
		return
	}
	defer ws.Close()
	currentUser, err := user.Current()
	if err != nil {
		log.Fatal("Couldn't get user")
	}
	execCmd := "bash"
	if currentUser.Username == "root" {
		execCmd = "login"
	}
	c := exec.Command(execCmd)
	ptmx, err := pty.Start(c)
	if err != nil {
		log.Fatal("Failed to start pty:", err)
		return
	}
	if err := pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80}); err != nil {
		log.Fatal(err)
	}
	defer func() { _ = ptmx.Close() }()
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("error reading from pty: %v", err)
				}
				return
			}
			err = ws.WriteMessage(websocket.BinaryMessage, buf[:n])
			if err != nil {
				log.Printf("error writing to ws: %v", err)
				return
			}
		}
	}()
	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("ws closed: %v", err)
			}
			break
		}
		ptmx.Write(message)
	}
}

func toBytes(num *big.Int) []byte {
	var padded []byte
	// version: 0
	version := []byte{0}

	// length: 4 bytes
	length := noun.ByteLen(num)
	lenBytes, err := int64ToLittleEndianBytes(length)
	if err != nil {
		fmt.Println(fmt.Sprintf("%v", err))
	}
	bytes := makeBytes(num)
	padded = append(padded, version...)
	padded = append(padded, lenBytes...)
	padded = append(padded, bytes...)

	return padded
}

func makeBytes(num *big.Int) []byte {
	byteSlice := num.Bytes()
	// Reverse the slice for little-endian
	for i, j := 0, len(byteSlice)-1; i < j; i, j = i+1, j-1 {
		byteSlice[i], byteSlice[j] = byteSlice[j], byteSlice[i]
	}
	return byteSlice
}

func int64ToLittleEndianBytes(num int64) ([]byte, error) {
	buf := new(bytes.Buffer)
	// uint32 for 4 bytes
	err := binary.Write(buf, binary.LittleEndian, uint32(num))
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func sendBroadcast(conn net.Conn, broadcast string) (net.Conn, error) {
	nounType := noun.Cell{
		Head: noun.MakeNoun("tty"),
		Tail: noun.MakeNoun(broadcast),
	}
	n := noun.MakeNoun(nounType)
	jBytes := toBytes(noun.Jam(n))
	if conn != nil {
		_, err := conn.Write(jBytes)
		if err != nil {
			fmt.Println(fmt.Sprintf("Send tty error: %v", err))
			return nil, err
		}
	}
	return conn, nil
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)
	err := http.ListenAndServe(":8088", nil)
	if err != nil {
		log.Fatal("ListenAndServe error:", err)
	}
}
