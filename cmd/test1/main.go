package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

func main() {
	// Define command-line flags
	address := flag.String("address", "", "The server address in the format <host>:<port>")
	hostname := flag.String("hostname", "", "The hostname for the request (optional)")
	token := flag.String("token", "", "The token for the request (optional)")
	dnslookup := flag.Bool("dnslookup", false, "Perform DNS lookup on sending the sendTransaction request (optional)")
	transaction := flag.String("transaction", "<your transaction data>", "The transaction data to send (base64 encoded)")
	flag.Parse()

	// Validate required flags
	if *address == "" {
		log.Fatal("Usage: " + os.Args[0] + " -address <host>:<port> [-hostname <hostname>] [-token <token>]")
	}

	host, port, err := net.SplitHostPort(*address)
	if err != nil {
		log.Fatalf("Invalid address format. Expected <host>:<port>, got: %v", *address)
	}

	pinger, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		log.Fatalf("Failed to resolve host: %v", err)
	}
	pingpong, err := probing.NewPinger(pinger.String())
	if err != nil {
		log.Fatalf("Failed to create pinger: %v", err)
	}
	pingpong.Count = 10
	pingpong.Timeout = time.Second
	err = pingpong.Run()
	if err != nil {
		log.Fatalf("Ping failed: %v", err)
	}
	stats := pingpong.Statistics()
	log.Printf("Ping to %s: %v ms", stats.Addr, stats.AvgRtt.Milliseconds())

	start := time.Now()

	var connect string
	if *dnslookup {
		connect = *address
	} else {
		connect = pinger.IP.String() + ":" + port

	}
	conn, err := net.Dial("tcp", connect)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Use hostname flag if provided, otherwise default to the address
	if *hostname == "" {
		*hostname = *address
	}

	log.Println("Dial:", time.Since(start))

	requestBody := `{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "sendTransaction",
		"params": ["` + *transaction + `", {"skipPreflight": true, "encoding": "base64"}]
	}`

	req := "POST /" + *token + " HTTP/1.1\r\n"
	req += "Host: " + *hostname + "\r\n"
	req += "Content-Type: application/json\r\n"
	req += "Content-Length: " + strconv.Itoa(len(requestBody)) + "\r\n"
	req += "Connection: close\r\n"
	req += "\r\n" + requestBody

	bytesWritten, err := conn.Write([]byte(req))
	if err != nil {
		panic(err)
	}

	log.Println("Bytes written:", bytesWritten)
	log.Println("Write request:", time.Since(start))

	oneByte := make([]byte, 1)
	_, err = conn.Read(oneByte)
	if err != nil {
		panic(err)
	}
	log.Println("First byte:", time.Since(start))

	responseBody, err := ioutil.ReadAll(conn)
	if err != nil {
		panic(err)
	}
	log.Println("Everything:", time.Since(start))
	log.Println("Data:", string(oneByte), string(responseBody))
}
