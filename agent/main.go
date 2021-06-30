package main

import (
	"common/tcpjoin"
	"flag"
	"fmt"
	"log"
	"net"
	"time"
)

var (
	serverAddr    string
	proxyAddr     string
	proxyPassword string
)

var (
	ch chan struct{}
)

func main() {
	flag.Usage = func() {
		fmt.Println("Usage: public")
		flag.PrintDefaults()
	}

	flag.StringVar(&serverAddr, "s", "", "server address for connecting")
	flag.StringVar(&proxyAddr, "x", "", "proxy address for connecting")
	flag.StringVar(&proxyPassword, "p", "", "password for proxy")
	flag.Parse()

	if serverAddr == "" {
		flag.Usage()
		return
	}

	if proxyAddr == "" {
		flag.Usage()
		return
	}

	log.Println("private v0.1.0")

	ch = make(chan struct{}, 5)
	ch <- struct{}{}
	ch <- struct{}{}
	ch <- struct{}{}
	ch <- struct{}{}
	ch <- struct{}{}

	for {
		select {
		case <-ch:
			go connectAndServe()
		}
	}
}

func connectAndServe() {
	defer func() {
		ch <- struct{}{}
	}()

	buf := make([]byte, len(proxyPassword))

	rw1, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		time.Sleep(time.Second * 5)
		return
	}
	log.Printf("[INFO] %s connected to proxy", rw1.LocalAddr().String())

	_, err = rw1.Write([]byte(proxyPassword))
	if err != nil {
		rw1.Close()
		time.Sleep(time.Second * 5)
		return
	}
	log.Printf("[INFO] %s sent password", rw1.LocalAddr().String())

	n, err := rw1.Read(buf)
	if err != nil {
		rw1.Close()
		time.Sleep(time.Second * 5)
		return
	}
	log.Printf("[INFO] %s received password %s", rw1.LocalAddr().String(), buf[:n])

	if string(buf[:n]) != proxyPassword {
		rw1.Close()
		time.Sleep(time.Second * 5)
		return
	}

	rw2, err := net.Dial("tcp", serverAddr)
	if err != nil {
		rw1.Close()
		time.Sleep(time.Second * 5)
		return
	}
	log.Printf("[INFO] %s connected to server", rw1.LocalAddr().String())

	log.Printf("[INFO] %s <-> %s", rw1.LocalAddr().String(), rw2.LocalAddr().String())
	join := tcpjoin.New(rw1, rw2)
	go join.Run()
}
