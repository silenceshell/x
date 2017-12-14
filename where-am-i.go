package main

import (
	"fmt"
	"github.com/yinheli/qqwry"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	flag "github.com/spf13/pflag"
)

var q *qqwry.QQwry
var t *template.Template

func main() {
	var port int
	flag.IntVar(&port, "port", 8888, "specify server port, default 8888")
	flag.Parse()

	q = qqwry.NewQQwry("qqwry.dat")
	t, _ = template.ParseFiles("tmpl/index.html")

	http.HandleFunc("/", indexHandler)
	var address string = fmt.Sprintf(":%d", port)
	err := http.ListenAndServe(address, nil)
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
		err = t.Execute(w, map[string]string{"Country": q.Country, "City": q.City, "Ip": ip})
		if err != nil {
			io.WriteString(w, "internal error")
			return
		}
	} else if r.Method == "POST" {
		r.ParseForm()

		ipx := r.Form["ipx"][0]
		var message string
		if net.ParseIP(ipx) == nil {
			message = ipx + "：格式不合法。"
		} else {
			q.Find(ipx)
			message = ipx + "的地址是： " + q.Country + q.City
		}

		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			io.WriteString(w, "internal error")
			return
		}
		q.Find(ip)

		t.Execute(w, map[string]string{"Country": q.Country, "City": q.City, "Ip": ip, "Message": message})
	}
	return
}

func getCountryAndCity(ip string) (country, city string, err error) {
	ip, _, e := net.SplitHostPort(ip)
	if e != nil {
		return "", "", fmt.Errorf("internal error")
	}
	q.Find(ip)
	return q.City, q.City, nil
}
