package main

import (
	"bufio"
	"log"
	"os"
	"time"
	"flag"
	"io"

	"net"
	"net/http"
)

type ConnServer struct {
	port string
	host string
	statusCode int
}

func NewServer(host string, port string, statusCode int) *ConnServer {
	s := ConnServer{host: host, port: port, statusCode: statusCode}
	return &s
}

// Attempt to start the server, and if we can't, exit because what else is there?
func (s *ConnServer) StartServer() {
	log.Println("Listening on "+s.host+":"+s.port)
	listen, err := net.Listen("tcp", s.host+":"+s.port)
	if err != nil {
		log.Fatal("Listening has failed: ", err)
	}

	// We can return immediately and do other stuff if we want.

	func() {
		for {
			conn, err := listen.Accept()
			log.Println("Accepting connection")

			if err != nil {
				log.Println("Failed to accept new connection: ", err)
				defer conn.Close()
				continue
			}
			go DumpRequest(conn, s.statusCode)
		}
	}()
}

func DumpRequest(conn net.Conn, statusCode int) {
	t := time.Now()
	tfmt := t.Format("20060102150405")
	thttpfmt := t.Format(http.TimeFormat)
	defer conn.Close()

	f, err := os.CreateTemp("", "gowebdump-"+tfmt+"-") // in Go version older than 1.17 you can use ioutil.TempFile
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		log.Println("setting read deadline:", err)
		return
	}

	reader := bufio.NewReader(io.TeeReader(conn, f))
	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Println("reading request:", err)
		return
	}
	defer req.Body.Close()

	if _, err := io.Copy(io.Discard, req.Body); err != nil {
		log.Println("reading request body:", err)
		return
	}

	log.Printf("Saved request to %s; request: %s %s", f.Name(), req.Method, req.URL)

	response := ""
	if statusCode == 200 {
		response = "HTTP/1.1 200 OK\nContent-Type: text/html; charset=UTF-8\nServer: gowebdump 0.2\nDate: " + thttpfmt + "\nContent-Length: 2\n\nOK"
	} else if statusCode == 400 {
		response = "HTTP/1.1 400 Bad Request\nContent-Type: text/html; charset=UTF-8\nServer: gowebdump 0.2\nDate: " + thttpfmt + "\nContent-Length: 0\n\n"
	} else if statusCode == 404 {
		response = "HTTP/1.1 404 Not Found\nContent-Type: text/html; charset=UTF-8\nServer: gowebdump 0.2\nDate: " + thttpfmt + "\nContent-Length: 0\n\n"
	}

	if _, err := io.WriteString(conn, response); err != nil {
		log.Println("Error writing response:", err)
	}
}

func main() {
	
	port := flag.String("port", "7780", "Port to bind to")
	bindaddr := flag.String("bindaddr", "0.0.0.0", "Listen address")
	statuscode := flag.Int("statuscode", 200, "Status Code to return")

	flag.Parse()

	server := NewServer(*bindaddr, *port, *statuscode)
	server.StartServer()
}
