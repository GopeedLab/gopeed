package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

type Message struct {
	Method  string          `json:"method"`
	Version string          `json:"version"`
	Params  json.RawMessage `json:"params"`
}

type Response struct {
	Code    int    `json:"code"`
	Data    any    `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

var apiMap = map[string]func(message *Message) (data any, err error){
	"ping": func(message *Message) (data any, err error) {
		conn, err := Dial()
		if err != nil {
			return false, err
		}
		defer conn.Close()
		return true, nil
	},
	"create": func(message *Message) (data any, err error) {
		buf, err := message.Params.MarshalJSON()
		if err != nil {
			return
		}

		client := &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return Dial()
				},
			},
			Timeout: 10 * time.Second,
		}
		req, err := http.NewRequest("POST", "http://127.0.0.1/create", bytes.NewBuffer(buf))
		if err != nil {
			return
		}
		req.Header.Set("X-Gopeed-Extension-Version", message.Version)
		_, err = client.Do(req)
		return
	},
}

// go build -ldflags="-s -w" -o ui/flutter/assets/exec/ github.com/GopeedLab/gopeed/cmd/host

func main() {
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
			data, err = handler(&message)
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
