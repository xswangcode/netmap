package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// NetMap 协议魔数。
var magic = [4]byte{'N', 'M', 'A', 'P'}

// 协议版本。
const version byte = 1

// Command NetMap 请求类型。
type Command byte

const (
	// CommandConnect 建立 TCP 连接。
	CommandConnect Command = 1
)

// ConnectStatus CONNECT 请求处理结果。
type ConnectStatus byte

const (
	// ConnectSuccess 目标连接成功。
	ConnectSuccess ConnectStatus = 0

	// ConnectFailed 目标连接失败。
	ConnectFailed ConnectStatus = 1
)

// ConnectRequest TCP 连接请求。
//
// Client 发送给 Relay，告诉 Relay：
// “请帮我连接这个目标地址”。
type ConnectRequest struct {
	Host string
	Port uint16
}

// ConnectResponse TCP 连接响应。
//
// Relay 连接目标服务后，将结果返回给 Client。
type ConnectResponse struct {
	Status ConnectStatus
}

// WriteConnectRequest 向连接中写入 CONNECT 请求。
func WriteConnectRequest(w io.Writer, request *ConnectRequest) error {
	host := []byte(request.Host)

	if len(host) == 0 {
		return fmt.Errorf("target host cannot be empty")
	}

	if len(host) > 65535 {
		return fmt.Errorf("target host is too long")
	}

	// 协议格式：
	//
	// Magic      4 bytes
	// Version    1 byte
	// Command    1 byte
	// HostLen    2 bytes
	// Host       N bytes
	// Port       2 bytes

	header := make([]byte, 8)

	copy(header[0:4], magic[:])
	header[4] = version
	header[5] = byte(CommandConnect)

	binary.BigEndian.PutUint16(
		header[6:8],
		uint16(len(host)),
	)

	if _, err := w.Write(header); err != nil {
		return fmt.Errorf(
			"write connect request header failed: %w",
			err,
		)
	}

	if _, err := w.Write(host); err != nil {
		return fmt.Errorf(
			"write target host failed: %w",
			err,
		)
	}

	port := make([]byte, 2)

	binary.BigEndian.PutUint16(
		port,
		request.Port,
	)

	if _, err := w.Write(port); err != nil {
		return fmt.Errorf(
			"write target port failed: %w",
			err,
		)
	}

	return nil
}

// ReadConnectRequest 从连接中读取 CONNECT 请求。
func ReadConnectRequest(r io.Reader) (*ConnectRequest, error) {
	header := make([]byte, 8)

	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf(
			"read connect request header failed: %w",
			err,
		)
	}

	if string(header[0:4]) != string(magic[:]) {
		return nil, fmt.Errorf(
			"invalid netmap protocol magic",
		)
	}

	if header[4] != version {
		return nil, fmt.Errorf(
			"unsupported netmap protocol version: %d",
			header[4],
		)
	}

	if Command(header[5]) != CommandConnect {
		return nil, fmt.Errorf(
			"unsupported netmap command: %d",
			header[5],
		)
	}

	hostLength := binary.BigEndian.Uint16(header[6:8])

	if hostLength == 0 {
		return nil, fmt.Errorf(
			"target host cannot be empty",
		)
	}

	host := make([]byte, hostLength)

	if _, err := io.ReadFull(r, host); err != nil {
		return nil, fmt.Errorf(
			"read target host failed: %w",
			err,
		)
	}

	portBytes := make([]byte, 2)

	if _, err := io.ReadFull(r, portBytes); err != nil {
		return nil, fmt.Errorf(
			"read target port failed: %w",
			err,
		)
	}

	port := binary.BigEndian.Uint16(portBytes)

	if port == 0 {
		return nil, fmt.Errorf(
			"target port cannot be zero",
		)
	}

	return &ConnectRequest{
		Host: string(host),
		Port: port,
	}, nil
}

// WriteConnectResponse 向连接中写入 CONNECT 响应。
func WriteConnectResponse(
	w io.Writer,
	response *ConnectResponse,
) error {
	// 响应格式：
	//
	// Magic      4 bytes
	// Version    1 byte
	// Status     1 byte

	data := make([]byte, 6)

	copy(data[0:4], magic[:])
	data[4] = version
	data[5] = byte(response.Status)

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf(
			"write connect response failed: %w",
			err,
		)
	}

	return nil
}

// ReadConnectResponse 从连接中读取 CONNECT 响应。
func ReadConnectResponse(r io.Reader) (*ConnectResponse, error) {
	data := make([]byte, 6)

	if _, err := io.ReadFull(r, data); err != nil {
		return nil, fmt.Errorf(
			"read connect response failed: %w",
			err,
		)
	}

	if string(data[0:4]) != string(magic[:]) {
		return nil, fmt.Errorf(
			"invalid netmap protocol magic",
		)
	}

	if data[4] != version {
		return nil, fmt.Errorf(
			"unsupported netmap protocol version: %d",
			data[4],
		)
	}

	status := ConnectStatus(data[5])

	if status != ConnectSuccess &&
		status != ConnectFailed {
		return nil, fmt.Errorf(
			"unsupported connect status: %d",
			status,
		)
	}

	return &ConnectResponse{
		Status: status,
	}, nil
}
