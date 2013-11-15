package main

import (
	"net"
	"net/http"

	"io"
	"log"
)

func websocketProxy(target string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d, err := net.Dial("tcp", target)
		if err != nil {
			http.Error(w, "Error contacting backend server.", 500)
			log.Printf("Error dialing websocket backend %s: %v", target, err)
			return
		}
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "Not a hijacker?", 500)
			return
		}
		nc, _, err := hj.Hijack()
		if err != nil {
			log.Printf("Hijack error: %v", err)
			return
		}
		defer nc.Close()
		defer d.Close()

		// Write headers by hand, because libwebsocket on the iPhone wants it that way ;)
		s := "GET /ws HTTP/1.1\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"Host: 192.168.10.231\r\n" +
			"Origin: http://192.168.10.231\r\n" +
			"Pragma: no-cache\r\n" +
			"Cache-Control: no-cache\r\n" +
			"Sec-WebSocket-Key: " + r.Header.Get("Sec-WebSocket-Key") + "\r\n" +
			"Sec-WebSocket-Version: " + r.Header.Get("Sec-WebSocket-Version") + "\r\n" +
			"Sec-WebSocket-Extensions: " + r.Header.Get("Sec-WebSocket-Extensions") + "\r\n\r\n"

		_, err = io.WriteString(d, s)
		if err != nil {
			log.Printf("Error copying request to target: %v", err)
			return
		}

		errc := make(chan error, 2)
		cp := func(dst io.Writer, src io.Reader) {
			_, err := io.Copy(dst, src)
			errc <- err
		}
		go cp(d, nc)
		go cp(nc, d)
		<-errc
	})
}

func main() {
	http.Handle("/ws", websocketProxy("192.168.10.231:80"))
	http.Handle("/", http.FileServer(http.Dir("./resources")))
	http.ListenAndServe(":8080", nil)
}
