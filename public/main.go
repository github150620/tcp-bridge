package main

import (
	"flag"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"common/tcpjoin"
)

var (
	addr1    string
	addr2    string
	password string
)

var (
	wg  sync.WaitGroup
	ch1 chan net.Conn
	ch2 chan net.Conn
)

func main() {
	log.Println("public v0.1.0")

	flag.StringVar(&addr1, "addr1", "", "addr1")
	flag.StringVar(&addr2, "addr2", "", "addr2")
	flag.StringVar(&password, "password", "", "password")
	flag.Parse()

	if addr1 == "" {
		return
	}

	if addr2 == "" {
		return
	}

	if password == "" {
		return
	}

	ch1 = make(chan net.Conn, 10)
	ch2 = make(chan net.Conn, 10)

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := listenAndServe1(addr1)
		if err != nil {
			os.Exit(1)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := listenAndServe2(addr2)
		if err != nil {
			os.Exit(1)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		pair()
	}()

	wg.Wait()
}

func listenAndServe1(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Println("[ERROR]", "Listen failed:", err)
		return err
	}
	defer listener.Close()

	for {
		rw, err := listener.Accept()
		if err != nil {
			log.Println("[WARN] Accept failed:", err)
			time.Sleep(time.Second * 5)
			continue
		}
		log.Printf("[INFO] %s", rw.RemoteAddr().String())
		serve1(rw)
	}
}

func serve1(rw net.Conn) {
	ch1 <- rw
}

func listenAndServe2(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Println("[ERROR]", "Listen failed:", err)
		return err
	}
	defer listener.Close()

	for {
		rw, err := listener.Accept()
		if err != nil {
			log.Println("[WARN]", "Accept failed:", err)
			time.Sleep(time.Second * 5)
			continue
		}
		log.Printf("[INFO] %s", rw.RemoteAddr().String())
		serve2(rw)
	}
}

func serve2(rw net.Conn) {
	buf := make([]byte, 8)
	_, err := rw.Read(buf)
	if err != nil {
		rw.Close()
		return
	}
	if string(buf) != password {
		rw.Close()
		log.Printf("[WARN] password invalid: %s %s\n", rw.RemoteAddr().String(), string(buf))
		return
	}
	ch2 <- rw
}

func pair() {
	select {
	case rw1 := <-ch1:
		select {
		case rw2 := <-ch2:
			join := tcpjoin.New(rw1, rw2)
			go join.Run()
		case <-time.After(30 * time.Second):
			rw1.Close()
		}
	case rw1 := <-ch2:
		select {
		case rw2 := <-ch1:
			join := tcpjoin.New(rw1, rw2)
			go join.Run()
		case <-time.After(30 * time.Second):
			rw1.Close()
		}
	}
}
