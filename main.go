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
			go DumpRequest(conn)
		}
	}()
}

func DumpRequest(conn net.Conn) {
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

	// Copies the decoded body; chunked-transfer framing is not included.
	if _, err := io.Copy(io.Discard, req.Body); err != nil {
		log.Println("reading request body:", err)
		return
	}

	log.Printf("Saved request to %s; request: %s %s", f.Name(), req.Method, req.URL)

	response_OK := "HTTP/1.1 200 OK\nContent-Type: text/html; charset=UTF-8\nServer: gowebdump 0.1\nDate: " + thttpfmt + "Content-Length: 2\n\nOK"
	//response_BADRESPONSE := "HTTP/1.1 400 Bad Request\nContent-Type: text/html; charset=UTF-8\nServer: gowebdump 0.1\nDate: " + thttpfmt + "Content-Length: 0\n\n"
	//response_NOTFOUND := "HTTP/1.1 404 Not Found\nContent-Type: text/html; charset=UTF-8\nServer: gowebdump 0.1\nDate: " + thttpfmt + "Content-Length: 0\n\n"

	if _, err := io.WriteString(conn, response_OK); err != nil {
		log.Println("Error writing response:", err)
	}
}

func Serve() {
	mux := http.NewServeMux()

	err := http.ListenAndServe(":7780", mux)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	
	port := flag.String("port", "7780", "Port to bind to")
	bindaddr := flag.String("bindaddr", "0.0.0.0", "Listen address")
	statuscode := flag.Int("statuscode", 200, "Status Code to return")

	server := NewServer(*bindaddr, *port, *statuscode)
	server.StartServer()
}
