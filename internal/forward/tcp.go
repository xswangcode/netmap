package forward

import (
	"io"
	"net"

	"netmap/internal/logger"
)

// TCP 在两个 TCP 连接之间进行双向数据转发。
//
// 一个方向结束时，只关闭对端的写方向，
// 不立即关闭整个 TCP 连接，避免影响另一个方向的数据传输。
func TCP(a net.Conn, b net.Conn) {
	done := make(chan struct{}, 2)

	go func() {
		n, err := io.Copy(a, b)

		logger.Debug(
			"tcp forward closed direction=b->a bytes=%d error=%v",
			n,
			err,
		)

		// b -> a 方向结束。
		// 关闭 a 的写方向，通知 a 对端不会再发送数据。
		closeWrite(a)

		done <- struct{}{}
	}()

	go func() {
		n, err := io.Copy(b, a)

		logger.Debug(
			"tcp forward closed direction=a->b bytes=%d error=%v",
			n,
			err,
		)

		// a -> b 方向结束。
		// 关闭 b 的写方向，通知 b 对端不会再发送数据。
		closeWrite(b)

		done <- struct{}{}
	}()

	// 必须等待两个方向都结束。
	<-done
	<-done
}

// closeWrite 关闭 TCP 连接的写方向。
//
// 普通 net.Conn 不一定支持半关闭，因此这里只对 TCPConn 进行处理。
func closeWrite(conn net.Conn) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return
	}

	if err := tcpConn.CloseWrite(); err != nil {
		logger.Debug(
			"tcp half-close write failed local=%s remote=%s error=%v",
			tcpConn.LocalAddr(),
			tcpConn.RemoteAddr(),
			err,
		)
	}
}
