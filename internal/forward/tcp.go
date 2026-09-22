package forward

import (
	"io"
	"net"
)

// TCP 在两个 TCP 连接之间进行双向数据转发。
func TCP(a net.Conn, b net.Conn) {
	done := make(chan struct{}, 2)

	go func() {
		_, _ = io.Copy(a, b)
		done <- struct{}{}
	}()

	go func() {
		_, _ = io.Copy(b, a)
		done <- struct{}{}
	}()

	<-done
}
