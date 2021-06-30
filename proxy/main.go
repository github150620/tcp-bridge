package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"common/tcpjoin"
)

var (
	clientListenAddr string
	agentListenAddr  string
	agentToken    string
)

var (
	ch1 chan net.Conn
	ch2 chan net.Conn
)

func main() {
	flag.Usage = func() {
		fmt.Println("Usage: proxy")
		flag.PrintDefaults()
	}

	flag.StringVar(&clientListenAddr, "client", "", "listen address for client")
	flag.StringVar(&agentListenAddr, "agent", "", "listen address for agent")
	flag.StringVar(&agentToken, "token", "", "token for agent")
	flag.Parse()

	if clientListenAddr == "" {
		flag.Usage()
		return
	}

	if agentListenAddr == "" {
		flag.Usage()
		return
	}

	if agentToken == "" {
		flag.Usage()
		return
	}

	log.Println("proxy v0.1.0")

	ch1 = make(chan net.Conn, 10)
	ch2 = make(chan net.Conn, 10)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := listenAndServe1(clientListenAddr)
		if err != nil {
			log.Println("[ERROR]", err)
			os.Exit(1)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := listenAndServe2(agentListenAddr)
		if err != nil {
			log.Println("[ERROR]", err)
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
		log.Printf("[INFO] client %s connected", rw.RemoteAddr().String())
		go serve1(rw)
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
		log.Printf("[INFO] agent %s connected", rw.RemoteAddr().String())
		go serve2(rw)
	}
}

func serve2(rw net.Conn) {
	buf := make([]byte, len(agentToken))
	_, err := rw.Read(buf)
	if err != nil {
		rw.Close()
		return
	}
	if string(buf) != agentToken {
		rw.Close()
		log.Printf("[WARN] %s token invalid (%s)\n", rw.RemoteAddr().String(), string(buf))
		return
	}
	log.Printf("[INFO] %s token ok", rw.RemoteAddr().String())

	ch2 <- rw
}

func pair() {
	for {
		select {
		case rw1 := <-ch1:
			select {
			case rw2 := <-ch2:
				log.Printf("[INFO] %s send token...", rw2.RemoteAddr().String())
				rw2.Write([]byte(agentToken))
				log.Printf("[INFO] %s <-> %s", rw2.RemoteAddr().String(), rw1.RemoteAddr().String())
				join := tcpjoin.New(rw1, rw2)
				go join.Run()
			case <-time.After(30 * time.Second):
				rw1.Close()
				log.Printf("[INFO] %s no pair", rw1.RemoteAddr().String())
			}
		case rw2 := <-ch2:
			select {
			case rw1 := <-ch1:
				log.Printf("[INFO] %s send token...", rw2.RemoteAddr().String())
				rw2.Write([]byte(agentToken))
				log.Printf("[INFO] %s <-> %s", rw2.RemoteAddr().String(), rw1.RemoteAddr().String())
				join := tcpjoin.New(rw2, rw1)
				go join.Run()
			case <-time.After(30 * time.Second):
				rw2.Close()
				log.Printf("[INFO] %s no pair", rw2.RemoteAddr().String())
			}
		}
	}
}
