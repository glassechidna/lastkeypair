package netcat
// adapted from https://github.com/vfedoroff/go-netcat/blob/master/main.go

import (
	"log"
	"io"
	"net"
)

func TcpToPipes(conn net.Conn, src io.Reader, dst io.Writer) {
	chanToStdout := streamCopy(conn, dst)
	chanToRemote := streamCopy(src, conn)
	select {
	case <-chanToStdout:
		log.Println("Remote connection is closed")
	case <-chanToRemote:
		log.Println("Local program is terminated")
	}
}

// Performs copy operation between streams: os and tcp streams
func streamCopy(src io.Reader, dst io.Writer) <-chan int {
	buf := make([]byte, 1024)
	syncChannel := make(chan int)
	go func() {
		defer func() {
			if con, ok := dst.(net.Conn); ok {
				con.Close()
				log.Printf("Connection from %v is closed\n", con.RemoteAddr())
			}
			syncChannel <- 0 // Notify that processing is finished
		}()
		for {
			var nBytes int
			var err error
			nBytes, err = src.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Read error: %s\n", err)
				}
				break
			}
			_, err = dst.Write(buf[0:nBytes])
			if err != nil {
				log.Fatalf("Write error: %s\n", err)
			}
		}
	}()
	return syncChannel
}
