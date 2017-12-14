package main

import (
	"github.com/yinheli/qqwry"
	"io"
	"log"
	"net"
	"net/http"
)

var q *qqwry.QQwry

func main() {
	q = qqwry.NewQQwry("qqwry.dat")

	http.HandleFunc("/", indexHandler)
	err := http.ListenAndServe(":88", nil)
	if err != nil {
		log.Fatal("server start failed.", err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			io.WriteString(w, "internal error")
			return
		}
		q.Find(ip)
		io.WriteString(w, "您好，来自"+q.Country+q.City+"的朋友！您的ip地址是："+ip)
	}
	return
}
