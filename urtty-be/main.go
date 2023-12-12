package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"os/exec"
	"os/user"
	"reflect"

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
	makeShell()
	defer func() { _ = ptmx.Close() }()
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
				err = sendBroadcast(conn, jsonStr)
				if err != nil {
					log.Printf("error writing to socket: %v", err)
					return
				}
			}
		}
	}()
	readBuf := make([]byte, 0, 4096)
	tmp := make([]byte, 1024)
	for {
		n, err := conn.Read(tmp)
		if err != nil {
			if err != io.EOF {
				log.Printf("error reading from socket: %v", err)
			}
			break
		}
		readBuf = append(readBuf, tmp[:n]...)
		decodedData := handleAction(readBuf)
		if string(decodedData) == "init" {
			fmt.Println("Initializing shell")
		} else if ptmx != nil {
			ptmx.Write(decodedData)
		}
		readBuf = readBuf[:0]
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

func handleAction(result []byte) []byte {
	stripped := result[5:]
	reversed := reverseLittleEndian(stripped)
	jam := new(big.Int).SetBytes(reversed)
	res := noun.Cue(jam)
	if reflect.TypeOf(res) == reflect.TypeOf(noun.Cell{}) {
		bytes, err := decodeAtom(noun.Slag(res, 1).String())
		if err != nil {
			fmt.Println(fmt.Sprintf("Failed to decode payload: %v", err))
			return []byte{}
		}
		return bytes
	}
	return []byte{}
}

func decodeAtom(atom string) ([]byte, error) {
	// Convert string to big.Int
	bigInt := new(big.Int)
	bigInt, ok := bigInt.SetString(atom, 10)
	if !ok {
		return []byte{}, fmt.Errorf("error converting string to big.Int")
	}

	// Convert big.Int to byte array
	byteArray := reverseLittleEndian(bigInt.Bytes())
	return byteArray, nil
}

func reverseLittleEndian(byteSlice []byte) []byte {
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

func sendBroadcast(conn net.Conn, broadcast string) error {
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
			return err
		}
	}
	return nil
}

func main() {
	sockPath := "../zod/.urb/dev/urtty/urtty.sock"
	conn, err := connectToIPC(sockPath)
	if err != nil {
		log.Printf("Dial error: %v", err)
		return
	}
	defer conn.Close()
	fmt.Println("Running")
	handleConnection(conn)
}
