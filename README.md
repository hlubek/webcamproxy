webcamproxy
===========

A proxy for Instant Webcam (or any other MPEG stream) that broadcasts the websocket message from one source to multiple clients.

# Installation

Install a current version of Go: http://golang.org/doc/install

Get the webcamproxy package and install it:

    go get github.com/chlu/webcamproxy

# Running

Start the proxy by pointing it to the IP address of the webcam stream:

    webcamproxy -webcam="192.168.1.2"
