package epoll

import (
	"golang.org/x/sys/unix"
)

type conn struct {
	fd int
	sa unix.Sockaddr
}
