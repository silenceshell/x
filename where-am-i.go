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
	"database/sql"
 	_ "github.com/go-sql-driver/mysql"
	"time"
	"strconv"
)

var q *qqwry.QQwry
var t *template.Template
var stmtIns *sql.Stmt
var stmtCount *sql.Stmt

func main() {
	var port int
	var mysqlHost, mysqlUser, mysqlPassword string

	flag.IntVar(&port, "port", 8888, "specify server port, default 8888")
	flag.StringVar(&mysqlHost, "host", "localhost", "specify mysql host, default localhost")
	flag.StringVar(&mysqlUser, "user", "root", "specify mysql user, default root")
	flag.StringVar(&mysqlPassword, "password", "", "specify mysql password, default empty")
	flag.Parse()

	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:3306)/x", mysqlUser, mysqlPassword, mysqlHost))
	if err != nil {
    	panic(err.Error())
	}
	defer db.Close()

	stmtIns, err = db.Prepare("INSERT INTO visitor VALUES( ?, ? )") // ? = ip, access time
	if err != nil {
		panic(err.Error())
	}
	defer stmtIns.Close()

	stmtCount, err = db.Prepare("SELECT count(*) FROM visitor")
	if err != nil {
		panic(err.Error())
	}
	defer stmtCount.Close()

	q = qqwry.NewQQwry("qqwry.dat")
	t, _ = template.ParseFiles("tmpl/index.html")

	http.HandleFunc("/", indexHandler)
	var address string = fmt.Sprintf(":%d", port)
	err = http.ListenAndServe(address, nil)
	if err != nil {
		log.Fatal("server start failed.", err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	var clientCountry, clientCity, ip string
	var count int
	var err error
	var tmplHash map[string]string = make(map[string]string)

	if ip, _, err = net.SplitHostPort(r.RemoteAddr); err != nil {
		io.WriteString(w, "internal error: invalid parameter.")
		return
	}
	q.Find(ip)
	clientCountry = q.Country
	clientCity = q.City

	fmt.Println(r.Method)

	now := time.Now()
	if _, err := stmtIns.Exec(ip, now.Format("2006-01-02 15:04:05")); err != nil {
		io.WriteString(w, "internal error x")
		return
	}
	if nil != stmtCount.QueryRow().Scan(&count) {
		io.WriteString(w, "internal error y")
		return
	}

	tmplHash["Country"] = clientCountry
	tmplHash["City"] = clientCity
	tmplHash["Ip"] = ip
	tmplHash["VisitorCount"] = strconv.Itoa(count)

	if r.Method == "GET" {
		t, _ = template.ParseFiles("tmpl/index.html")
		if err = t.Execute(w, tmplHash); err!=nil {
			io.WriteString(w, "internal error: 501"+err.Error())
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
		tmplHash["Message"] = message

		if nil != t.Execute(w, tmplHash) {
			io.WriteString(w, "internal error: 502")
			return
		}
	}

	return
}
