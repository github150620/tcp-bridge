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
	proxyToken string
)

var (
	ch chan struct{}
)

func main() {
	flag.Usage = func() {
		fmt.Println("Usage: agent -server <address> -proxy <address> -token <token>")
		flag.PrintDefaults()
	}

	flag.StringVar(&serverAddr, "server", "", "The server address like 192.168.1.100:3389.")
	flag.StringVar(&proxyAddr, "proxy", "", "The proxy address like x.x.x.x:x.")
	flag.StringVar(&proxyToken, "token", "", "The token proxy will check.")
	flag.Parse()

	if serverAddr == "" {
		flag.Usage()
		return
	}

	if proxyAddr == "" {
		flag.Usage()
		return
	}

	if proxyToken == "" {
		flag.Usage()
		return
	}

	log.Println("agent v0.1.0")

	ch = make(chan struct{}, 1)
	ch <- struct{}{}

	for {
		select {
		case <-ch:
			log.Printf("[INFO] Start a new connection")
			go connectAndServe()
		}
	}
}

func connectAndServe() {
	defer func() {
		ch <- struct{}{}
	}()

	buf := make([]byte, len(proxyToken))

	log.Printf("[INFO] Connecting to proxy(%s)...", proxyAddr)
	rw1, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		log.Printf("[WARN] Failed! Error: %v", err)
		time.Sleep(time.Second * 5)
		return
	}
	log.Printf("[INFO] Connected. Local address is %s.", rw1.LocalAddr().String())

	log.Printf("[INFO] Sending token(%s)...", []byte(proxyToken))
	n, err := rw1.Write([]byte(proxyToken))
	if err != nil {
		log.Printf("[WARN] Failed! Error: %v", err)
		rw1.Close()
		time.Sleep(time.Second * 5)
		return
	}
	log.Printf("[INFO] Sent.")

	log.Printf("[INFO] Waiting for proxy token...")
	n, err = rw1.Read(buf)
	if err != nil {
		log.Printf("[WARN] Receive failed. %v", err)
		rw1.Close()
		time.Sleep(time.Second * 5)
		return
	}
	log.Printf("[INFO] Received token(%s)", buf[:n])

	if string(buf[:n]) != proxyToken {
		log.Printf("[INFO] Invalid token. Try again after 5 seconds.")
		rw1.Close()
		time.Sleep(time.Second * 5)
		return
	}

	log.Printf("[INFO] Connecting to server(%s)...", serverAddr)
	rw2, err := net.Dial("tcp", serverAddr)
	if err != nil {
		log.Printf("[WARN] Failed! Error: %v", err)
		rw1.Close()
		time.Sleep(time.Second * 5)
		return
	}
	log.Printf("[INFO] Connected. Local address is %s", rw2.LocalAddr().String())

	log.Printf("[INFO] Join(%s,%s)...", rw1.LocalAddr().String(), rw2.LocalAddr().String())
	join := tcpjoin.New(rw1, rw2)
	go func() {
		join.Run()
		log.Printf("[INFO] Join(%s,%s) over.", rw1.LocalAddr().String(), rw2.LocalAddr().String())
	}()
}
