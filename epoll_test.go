package epoll

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"

	"github.com/roadrunner-server/tcplisten"
	"golang.org/x/sys/unix"
)

func TestName(t *testing.T) {
	ep, err := Create()
	if err != nil {
		panic(err)
	}

	cfg := tcplisten.Config{
		ReusePort:   false,
		DeferAccept: false,
		FastOpen:    false,
		Backlog:     0,
	}

	_, l, err := cfg.NewListenerWithFD("tcp4", "127.0.0.1:8080")
	if err != nil {
		panic(err)
	}

	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)

	//err = ep.AddFD(fd)
	//if err != nil {
	//	panic(err)
	//}

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			select {
			case <-s:
				return
			default:
				conn, err2 := l.Accept()
				if err2 != nil {
					panic(err2)
				}

				err = ep.AddConn(conn)
				if err != nil {
					panic(err)
				}
			}
		}
	}()

	go func() {
		for {
			ev := make([]unix.EpollEvent, 100)
			n, err2 := ep.Wait(ev)
			if err2 != nil {
				panic(err2)
			}

			go func() {
				for i := 0; i < n; i++ {
					fd := int(ev[i].Fd)
					if fd == 0 {
						break
					}

					fmt.Printf("fd is: %d\n", fd)
					fmt.Printf("event no: %d\n", ev[i].Events)

					// len of the IP packet
					var packet [65535]byte
					n, err22 := syscall.Read(fd, packet[:])
					if err22 != nil {
						if errors.Is(err22, unix.EWOULDBLOCK) {
							_ = ep.DeleteFD(fd)
							return
						}
						panic(err22)
					}

					if n == 0 {
						ep.ModWrite(fd)
						nw, err := syscall.Write(fd, []byte("echo"))
						if err != nil {
							panic(err)
						}
						fmt.Printf("data written: %d\n", nw)
						return
					}

					fmt.Printf("read data: %d\n", n)
				}
			}()

			//for i := 0; i < len(conns); i++ {
			//	data, err3 := io.ReadAll(conns[i])
			//	if err3 != nil {
			//		panic(err3)
			//	}
			//
			//	fmt.Println(string(data))
			//
			//	err := ep.Delete(conns[i])
			//	if err != nil {
			//		panic(err)
			//	}
			//	_ = conns[i].Close()
			//}
		}
	}()

	wg.Wait()
}
