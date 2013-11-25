webcamproxy [![Build Status](https://travis-ci.org/chlu/webcamproxy.png?branch=master)](https://travis-ci.org/chlu/webcamproxy)
===========

A proxy for Instant Webcam (or any other MPEG stream) that broadcasts the websocket message from one source to multiple clients.

# Installation

Install a current version of Go: http://golang.org/doc/install

Get the webcamproxy package and install it:

    go get github.com/chlu/webcamproxy

# Running

Start the proxy by pointing it to the IP address of the webcam stream:

    webcamproxy -webcam="192.168.1.2"

Access the webcam interface on http://localhost:8080/ to see the output of the webcam.
