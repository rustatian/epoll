package epoll

import (
	"net"
	"reflect"
)

const (
	connStr  string = "conn"
	fdStr    string = "fd"
	pfdStr   string = "pfd"
	sysfdStr string = "Sysfd"
)

func findSysFD(conn net.Conn) int {

	// net.conn private structure
	/*
		type conn struct {
			fd *netFD
		}
	*/
	connStruct := reflect.Indirect(reflect.ValueOf(conn)).FieldByName(connStr)
	// net.conn contains fd of *netFD
	fdVal := connStruct.FieldByName(fdStr)
	/*
		// Network file descriptor.
		type netFD struct {
			pfd poll.FD <-------------- WE NEED THIS FD

			// immutable until Close
			family      int
			sotype      int
			isConnected bool // handshake completed or use of association with peer
			net         string
			laddr       Addr
			raddr       Addr
		}
	*/
	pfdVal := reflect.Indirect(fdVal).FieldByName(pfdStr)

	/*
		// FD is a file descriptor. The net and os packages use this type as a
		// field of a larger type representing a network connection or OS file.
		type FD struct {
			// Lock sysfd and serialize access to Read and Write methods.
			fdmu fdMutex

			// System file descriptor. Immutable until Close.
			Sysfd int <-------------------------------------------- WE NEED SYSTEM FILE DESCRIPTOR

			// I/O poller.
			pd pollDesc

			// Writev cache.
			iovecs *[]syscall.Iovec

			// Semaphore signaled when file is closed.
			csema uint32

			// Non-zero if this file has been set to blocking mode.
			isBlocking uint32

			// Whether this is a streaming descriptor, as opposed to a
			// packet-based descriptor like a UDP socket. Immutable.
			IsStream bool

			// Whether a zero byte read indicates EOF. This is false for a
			// message based socket connection.
			ZeroReadIsEOF bool

			// Whether this is a file rather than a network socket.
			isFile bool

	*/
	return int(pfdVal.FieldByName(sysfdStr).Int())
}
