package protocol

import (
	"fmt"
	"io"
	"net"
)

// SOCKS5 协议版本。
const socks5Version byte = 0x05

// SOCKS5 命令。
const (
	// socks5CommandConnect TCP CONNECT。
	socks5CommandConnect byte = 0x01
)

// SOCKS5 地址类型。
const (
	// IPv4 地址。
	socks5AddressIPv4 byte = 0x01

	// 域名。
	socks5AddressDomain byte = 0x03

	// IPv6 地址。
	socks5AddressIPv6 byte = 0x04
)

// SOCKS5 认证方式。
const (
	// 不需要认证。
	socks5AuthNone byte = 0x00

	// 没有可接受的认证方式。
	socks5AuthNoAcceptable byte = 0xFF
)

// SOCKS5Response SOCKS5 响应状态。
const (
	// 连接成功。
	socks5ReplySuccess byte = 0x00

	// 一般 SOCKS5 连接失败。
	socks5ReplyGeneralFailure byte = 0x01

	// 不支持的命令。
	socks5ReplyCommandNotSupported byte = 0x07

	// 不支持的地址类型。
	socks5ReplyAddressTypeNotSupported byte = 0x08
)

// SOCKS5Request SOCKS5 CONNECT 请求。
type SOCKS5Request struct {
	Host string
	Port uint16
}

// ReadSOCKS5Request 读取 SOCKS5 CONNECT 请求。
//
// 第一版只支持：
// 1. SOCKS5
// 2. NO AUTHENTICATION REQUIRED
// 3. CONNECT
// 4. IPv4
// 5. Domain
// 6. IPv6
//
// reader 负责读取 SOCKS5 数据。
// writer 负责向 SOCKS5 客户端返回响应。
func ReadSOCKS5Request(
	reader io.Reader,
	writer io.Writer,
) (*SOCKS5Request, error) {
	// --------------------------------
	// 1. SOCKS5 握手
	// --------------------------------
	//
	// 客户端发送：
	//
	// +----+----------+----------+
	// |VER | NMETHODS | METHODS  |
	// +----+----------+----------+
	// | 1  |    1     | 1~255   |
	// +----+----------+----------+
	//
	// 我们只支持：
	//
	// 0x00 = NO AUTHENTICATION REQUIRED

	header := make([]byte, 2)

	if _, err := io.ReadFull(
		reader,
		header,
	); err != nil {
		return nil, fmt.Errorf(
			"read socks5 handshake failed: %w",
			err,
		)
	}

	if header[0] != socks5Version {
		return nil, fmt.Errorf(
			"unsupported socks5 version: %d",
			header[0],
		)
	}

	methodCount := int(header[1])

	if methodCount == 0 {
		return nil, fmt.Errorf(
			"socks5 method count cannot be zero",
		)
	}

	methods := make([]byte, methodCount)

	if _, err := io.ReadFull(
		reader,
		methods,
	); err != nil {
		return nil, fmt.Errorf(
			"read socks5 methods failed: %w",
			err,
		)
	}

	// 检查客户端是否支持 NO AUTH。
	supportNone := false

	for _, method := range methods {
		if method == socks5AuthNone {
			supportNone = true
			break
		}
	}

	if !supportNone {
		_ = writeSOCKS5AuthResponse(
			writer,
			socks5AuthNoAcceptable,
		)

		return nil, fmt.Errorf(
			"socks5 no-authentication method is not supported",
		)
	}

	// 告诉客户端使用 NO AUTH。
	if err := writeSOCKS5AuthResponse(
		writer,
		socks5AuthNone,
	); err != nil {
		return nil, err
	}

	// --------------------------------
	// 2. SOCKS5 CONNECT 请求
	// --------------------------------
	//
	// +----+-----+-------+------+----------+----------+
	// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  |   1   |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+

	requestHeader := make([]byte, 4)

	if _, err := io.ReadFull(
		reader,
		requestHeader,
	); err != nil {
		return nil, fmt.Errorf(
			"read socks5 request header failed: %w",
			err,
		)
	}

	if requestHeader[0] != socks5Version {
		return nil, fmt.Errorf(
			"unsupported socks5 request version: %d",
			requestHeader[0],
		)
	}

	// 第一版只支持 CONNECT。
	if requestHeader[1] != socks5CommandConnect {
		_ = writeSOCKS5Reply(
			writer,
			socks5ReplyCommandNotSupported,
			nil,
			0,
		)

		return nil, fmt.Errorf(
			"unsupported socks5 command: %d",
			requestHeader[1],
		)
	}

	addressType := requestHeader[3]

	host, err := readSOCKS5Address(
		reader,
		addressType,
	)
	if err != nil {
		_ = writeSOCKS5Reply(
			writer,
			socks5ReplyAddressTypeNotSupported,
			nil,
			0,
		)

		return nil, err
	}

	portBytes := make([]byte, 2)

	if _, err := io.ReadFull(
		reader,
		portBytes,
	); err != nil {
		return nil, fmt.Errorf(
			"read socks5 target port failed: %w",
			err,
		)
	}

	port := uint16(portBytes[0])<<8 |
		uint16(portBytes[1])

	if port == 0 {
		return nil, fmt.Errorf(
			"socks5 target port cannot be zero",
		)
	}

	return &SOCKS5Request{
		Host: host,
		Port: port,
	}, nil
}

// readSOCKS5Address 读取 SOCKS5 目标地址。
func readSOCKS5Address(
	reader io.Reader,
	addressType byte,
) (string, error) {
	switch addressType {

	case socks5AddressIPv4:
		data := make([]byte, 4)

		if _, err := io.ReadFull(
			reader,
			data,
		); err != nil {
			return "", fmt.Errorf(
				"read socks5 ipv4 address failed: %w",
				err,
			)
		}

		return net.IP(data).String(), nil

	case socks5AddressDomain:
		length := make([]byte, 1)

		if _, err := io.ReadFull(
			reader,
			length,
		); err != nil {
			return "", fmt.Errorf(
				"read socks5 domain length failed: %w",
				err,
			)
		}

		hostLength := int(length[0])

		if hostLength == 0 {
			return "", fmt.Errorf(
				"socks5 domain cannot be empty",
			)
		}

		host := make([]byte, hostLength)

		if _, err := io.ReadFull(
			reader,
			host,
		); err != nil {
			return "", fmt.Errorf(
				"read socks5 domain failed: %w",
				err,
			)
		}

		return string(host), nil

	case socks5AddressIPv6:
		data := make([]byte, 16)

		if _, err := io.ReadFull(
			reader,
			data,
		); err != nil {
			return "", fmt.Errorf(
				"read socks5 ipv6 address failed: %w",
				err,
			)
		}

		return net.IP(data).String(), nil

	default:
		return "", fmt.Errorf(
			"unsupported socks5 address type: %d",
			addressType,
		)
	}
}

// writeSOCKS5AuthResponse 写入 SOCKS5 认证响应。
func writeSOCKS5AuthResponse(
	writer io.Writer,
	method byte,
) error {
	data := []byte{
		socks5Version,
		method,
	}

	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf(
			"write socks5 auth response failed: %w",
			err,
		)
	}

	return nil
}

// WriteSOCKS5SuccessResponse 向 SOCKS5 客户端返回连接成功。
func WriteSOCKS5SuccessResponse(
	writer io.Writer,
) error {
	return writeSOCKS5Reply(
		writer,
		socks5ReplySuccess,
		net.IPv4zero,
		0,
	)
}

// WriteSOCKS5FailureResponse 向 SOCKS5 客户端返回连接失败。
func WriteSOCKS5FailureResponse(
	writer io.Writer,
) error {
	return writeSOCKS5Reply(
		writer,
		socks5ReplyGeneralFailure,
		net.IPv4zero,
		0,
	)
}

// writeSOCKS5Reply 写入 SOCKS5 CONNECT 响应。
//
// 第一版响应使用 IPv4 地址格式。
func writeSOCKS5Reply(
	writer io.Writer,
	reply byte,
	ip net.IP,
	port uint16,
) error {
	// 响应格式：
	//
	// +----+-----+-------+------+----------+----------+
	// |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  |   1   |  1   |    4     |    2     |
	// +----+-----+-------+------+----------+----------+

	data := make([]byte, 10)

	data[0] = socks5Version
	data[1] = reply
	data[2] = 0x00
	data[3] = socks5AddressIPv4

	ip = ip.To4()

	if ip == nil {
		ip = net.IPv4zero
	}

	copy(
		data[4:8],
		ip,
	)

	data[8] = byte(port >> 8)
	data[9] = byte(port)

	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf(
			"write socks5 reply failed: %w",
			err,
		)
	}

	return nil
}
