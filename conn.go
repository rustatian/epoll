package epoll

import (
	"golang.org/x/sys/unix"
)

// conn represents low-level connection (fd)
type conn struct {
	fd int
	sa unix.Sockaddr
}
