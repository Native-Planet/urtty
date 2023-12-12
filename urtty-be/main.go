package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"math/big"
	"net"
	"os/exec"
	"os/user"

	"github.com/creack/pty"
	"github.com/stevelacy/go-urbit/noun"
)

var (
	ptmx *os.File
)

type Broadcast struct {
	Broadcast string `json:"broadcast"`
}
type Action struct {
	Action string `json:"action"`
}

func makeShell() {
	currentUser, err := user.Current()
	if err != nil {
		log.Fatal("Couldn't get user")
	}
	// create a bash shell, or login shell if root
	execCmd := "bash"
	if currentUser.Username == "root" {
		execCmd = "login"
	}
	c := exec.Command(execCmd)
	ptmx, err = pty.Start(c)
	if err != nil {
		log.Fatal("Failed to start pty:", err)
		return
	}
	if err := pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80}); err != nil {
		log.Fatal(err)
	}
	defer func() { _ = ptmx.Close() }()
}

func connectToIPC(socketPath string) (net.Conn, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	go func() {
		buf := make([]byte, 1024)
		for {
			if ptmx != nil {
				n, err := ptmx.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("error reading from pty: %v", err)
					}
					return
				}
				encoded := base64.StdEncoding.EncodeToString(buf[:n])
				msg := Broadcast{Broadcast: encoded}
				jsonMsg, _ := json.Marshal(msg)
				jsonStr := string(jsonMsg)
				if err != nil {
					log.Printf("error marshalling json: %v", err)
					return
				}
				conn, err = sendBroadcast(conn, jsonStr)
				if err != nil {
					log.Printf("error writing to socket: %v", err)
					return
				}
			}
		}
	}()
	decoder := json.NewDecoder(conn)
	for {
		var msg Action
		if err := decoder.Decode(&msg); err != nil {
			if err != io.EOF {
				log.Printf("error reading from socket: %v", err)
			}
			break
		}
		data, err := base64.StdEncoding.DecodeString(msg.Action)
		if err != nil {
			log.Printf("error decoding base64: %v", err)
			continue
		}
		if string(data) == "init" {
			makeShell()
		} else if ptmx != nil {
			ptmx.Write(data)
		}
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
		Head: noun.MakeNoun("broadcast"),
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
    sockPath := "../zod/.urb/dev/urtty/urtty.sock"
    conn, err := connectToIPC(sockPath)
    if err != nil {
        log.Printf("Dial error: %v", err)
        return
    }
    defer conn.Close()
    go handleConnection(conn)
}
