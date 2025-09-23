package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"time"
)

const identifier = "gopeed"

type Message struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type Response struct {
	Code    int    `json:"code"`
	Data    any    `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

var apiMap = map[string]func(params json.RawMessage) (data any, err error){
	"ping": func(params json.RawMessage) (data any, err error) {
		conn, err := Dial()
		if err != nil {
			return false, err
		}
		defer conn.Close()
		return true, nil
	},
	"create": func(params json.RawMessage) (data any, err error) {
		var strParams string
		if err = json.Unmarshal(params, &strParams); err != nil {
			return
		}

		conn, err := Dial()
		if err != nil {
			return false, err
		}
		defer conn.Close()

		client := rpc.NewClientWithCodec(jsonrpc.NewClientCodec(conn))
		defer client.Close()

		var result bool
		err = client.Call("create", map[string]any{
			"params": strParams,
		}, &result)
		return
	},
}

// go build -ldflags="-s -w" -o ui/flutter/assets/exec/ github.com/GopeedLab/gopeed/cmd/host

func main() {
	doReq := func() {
		client := &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return Dial()
				},
			},
			Timeout: 10 * time.Second,
		}

		body := fmt.Sprintf(`{"id":1,"method":"register","params":{"identifier":"%s"}}`, identifier)
		resp, err := client.Post("http://gopeed/test", "application/json", bytes.NewBufferString(body))
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		fmt.Println("Response status:", resp.Status)
	}

	for i := 0; i < 10; i++ {
		go doReq()
	}

	for {
		// Read message length (first 4 bytes)
		var length uint32
		if err := binary.Read(os.Stdin, binary.NativeEndian, &length); err != nil {
			if err == io.EOF {
				// Connection closed by Chrome
				return
			}
			sendError("Failed to read message length: " + err.Error())
			return
		}

		// Read the message
		input := make([]byte, length)
		if _, err := io.ReadFull(os.Stdin, input); err != nil {
			sendError("Failed to read message: " + err.Error())
			return
		}

		// Parse message
		var message Message
		if err := json.Unmarshal(input, &message); err != nil {
			sendError("Failed to parse message: " + err.Error())
			return
		}

		// Handle request
		var data any
		var err error
		if handler, ok := apiMap[message.Method]; ok {
			data, err = handler(message.Params)
		} else {
			err = errors.New("Unknown method: " + message.Method)
		}
		if err != nil {
			sendError(err.Error())
			continue
		}
		sendResponse(0, data, "")
	}
}

func sendResponse(code int, data interface{}, message string) {
	response := Response{
		Code:    code,
		Data:    data,
		Message: message,
	}

	// Encode response
	responseBytes, err := json.Marshal(response)
	if err != nil {
		sendError("Failed to encode response: " + err.Error())
		return
	}

	// Write message length
	binary.Write(os.Stdout, binary.NativeEndian, uint32(len(responseBytes)))
	// Write message
	os.Stdout.Write(responseBytes)
}

func sendError(msg string) {
	sendResponse(1, nil, msg)
}
