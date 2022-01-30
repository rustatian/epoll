package epoll

import (
	"errors"
	"net"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

type Epoll struct {
	// epoll file descriptor
	fd int
	// https://man7.org/linux/man-pages/man2/eventfd.2.html
	efd   int
	mu    sync.RWMutex
	conns map[int]net.Conn // int - file descriptor, conn - associated connection
}

func Create() (*Epoll, error) {
	// https://man7.org/linux/man-pages/man2/epoll_create.2.html
	epFd, err := unix.EpollCreate1(0) // don't need a EPOLL_CLOEXEC
	if err != nil {
		return nil, err
	}

	// event FD needed to use wake
	r0, _, e0 := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if e0 != 0 {
		return nil, errors.New("error, e0 != 0")
	}

	return &Epoll{fd: epFd, efd: int(r0), conns: make(map[int]net.Conn)}, nil
}

func (ep *Epoll) AddFD(fd int) error {
	err := unix.EpollCtl(ep.fd, syscall.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: unix.EPOLLIN,
		Fd:     int32(fd),
		Pad:    0,
	})

	if err != nil {
		_ = syscall.Close(ep.fd)
		return err
	}

	return nil
}

func (ep *Epoll) AddConn(conn net.Conn) error {
	connFd := findSysFD(conn)

	/* EPOLL_CTL_ADD
	Add the file descriptor fd to the interest list for epfd. The set of events that
	we are interested in monitoring for fd is specified in the buffer pointed to
	by ev, as described below. If we attempt to add a file descriptor that is
	already in the interest list, epoll_ctl() fails with the error EEXIST.
	*/

	/*
		EPOLLIN Data other than high-priority data can be read
		EPOLLPRI High-priority data can be read
		EPOLLRDHUP Shutdown on peer socket (since Linux 2.6.17)
		EPOLLOUT Normal data can be written
		EPOLLET Employ edge-triggered event notification
		EPOLLONESHOT Disable monitoring after event notification
		EPOLLERR An error has occurred
		EPOLLHUP A hangup has occurred

	*/
	err := unix.EpollCtl(ep.fd, syscall.EPOLL_CTL_ADD, connFd, &unix.EpollEvent{
		Events: unix.EPOLLIN | unix.EPOLLHUP | unix.EPOLLET,
		Fd:     int32(connFd),
		Pad:    0,
	})
	if err != nil {
		return err
	}

	ep.mu.Lock()
	ep.conns[connFd] = conn
	ep.mu.Unlock()

	return nil
}

// https://man7.org/linux/man-pages/man2/epoll_wait.2.html
func (ep *Epoll) Wait(ev []unix.EpollEvent) (int, error) {

	/*
		If timeout equals –1, block until an event occurs for one of the file descriptors in
		the interest list for epfd or until a signal is caught.
	*/
	n, err := unix.EpollWait(ep.fd, ev[:], -1)
	if err != nil {
		if errors.Is(err, unix.EINTR) {
			return 0, nil
		}
		return 0, err
	}
	return n, nil
}

func (ep *Epoll) WaitBlocking() error {
	// todo(rustatian): to sync.pool
	//events := make([]unix.EpollEvent, 100)
	//for {
	//	n, err := unix.EpollWait(ep.fd, events, -1)
	//	if err != nil {
	//		if errors.Is(err, unix.EINTR) {
	//			return nil
	//		}
	//		return err
	//	}
	//
	//	for i := 0; i < n; i++ {
	//		ep.mu.RLock()
	//		if _, ok := ep.conns[int(events[i].Fd)]; !ok {
	//			fd, sa, err := syscall.Accept(int(events[i].Fd))
	//			if err != nil {
	//				return err
	//			}
	//
	//		}
	//	}
	//
	//	return nil
	//}
	return nil
}

func (ep *Epoll) ModWrite(fd int) {
	err := unix.EpollCtl(ep.fd, unix.EPOLL_CTL_MOD, fd, &unix.EpollEvent{
		Events: unix.EPOLLOUT,
		Fd:     int32(fd),
		Pad:    0,
	})
	if err != nil {
		panic(err)
	}
}

func (ep *Epoll) RawWait(ev []unix.EpollEvent) (int, error) {
	n, err := unix.EpollWait(ep.fd, ev, -1)
	if err != nil {
		if errors.Is(err, unix.EINTR) {
			return 0, nil
		}
		return 0, err
	}

	return n, nil
}

func (ep *Epoll) DeleteFD(fd int) error {
	err := unix.EpollCtl(ep.fd, unix.EPOLL_CTL_DEL, fd, &unix.EpollEvent{
		Events: unix.EPOLLIN | unix.EPOLLHUP,
		Fd:     int32(fd),
		Pad:    0,
	})
	if err != nil {
		return err
	}

	return nil
}

func (ep *Epoll) Delete(conn net.Conn) error {
	connFd := findSysFD(conn)
	err := unix.EpollCtl(ep.fd, unix.EPOLL_CTL_DEL, connFd, &unix.EpollEvent{
		Events: unix.EPOLLIN | unix.EPOLLHUP,
		Fd:     int32(connFd),
		Pad:    0,
	})
	if err != nil {
		return err
	}

	ep.mu.Lock()
	delete(ep.conns, connFd)
	ep.mu.Unlock()
	return nil
}
