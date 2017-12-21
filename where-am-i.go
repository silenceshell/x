package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	flag "github.com/spf13/pflag"
	"github.com/yinheli/qqwry"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
	//"os"
	"crypto/md5"
	"encoding/hex"
	"k8s.io/kubernetes/pkg/util/rand"

	"bytes"
)

var q *qqwry.QQwry
var indexT, guaT, tinyUrlT *template.Template
var urlInsert, urlUpdate, stmtIns, stmtCount *sql.Stmt
var ALPHABET string = "23456789bcdfghjkmnpqrstvwxyzBCDFGHJKLMNPQRSTVWXYZ-_";
var BASE int64 = int64(len(ALPHABET))

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

	urlInsert, err = db.Prepare(`INSERT INTO shorturl ( long_url, short_url, create_time ) VALUES ( ?, ?, ? )`)
	//urlInsert, err = db.Prepare(`INSERT INTO shorturl VALUES ( ?, ? )`)
	if err != nil {
		panic(err.Error())
	}
	defer urlInsert.Close()


	urlUpdate, err = db.Prepare("UPDATE shorturl SET short_url = ? WHERE id = ?")
	if err != nil {
		panic(err.Error())
	}
	defer urlUpdate.Close()

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

	indexT, _ = template.ParseFiles("tmpl/head.html", "tmpl/header.html", "tmpl/index.html", "tmpl/footer.html")
	guaT, _ = template.ParseFiles("tmpl/head.html", "tmpl/header.html", "tmpl/gua.html", "tmpl/footer.html")
	tinyUrlT, _ = template.ParseFiles("tmpl/head.html", "tmpl/header.html", "tmpl/tinyurl.html", "tmpl/footer.html")
	//t, _ = template.ParseFiles("tmpl/index.html")

	http.HandleFunc("/index", indexHandler)
	http.HandleFunc("/gua", guaHandler)
	http.HandleFunc("/tinyurl", tinyurlHandler)
	http.HandleFunc("/", defaultHandler)
	var address string = fmt.Sprintf(":%d", port)
	err = http.ListenAndServe(address, nil)
	if err != nil {
		log.Fatal("server start failed.", err)
	}
}

func encode(num int64) string {
	var b bytes.Buffer
	//var str []string = make([]string, 0, 6)
	for num > 0 {
		b.WriteByte(ALPHABET[num % BASE])
		//str = append(str, string())
		num = num / BASE;
	}
	return b.String()
		//StringBuilder str = new StringBuilder();
		//while (num > 0) {
		//	str.insert(0, ALPHABET.charAt(num % BASE));
		//	num = num / BASE;
		//}
		//return str.toString();
}

func getTinyUrl(url string) (string, error) {
	hasher := md5.New()
	hasher.Write([]byte(url))

	hex.EncodeToString(hasher.Sum(nil))

	now := time.Now()
	if res, err := urlInsert.Exec(url, "", now.Format("2006-01-02 15:04:05")); err != nil {
        id, err := res.LastInsertId()
        if err != nil {
            println("Error:", err.Error())
        } else {
            println("LastInsertId:", id)
        }

		var idStr string = encode(id)
		if _, err := urlUpdate.Exec(idStr, id); err != nil {
			log.Fatal("update tiny url table failed.", err)
		}

		return idStr, nil
	}
	return "", fmt.Errorf("get failed")
}

func tinyurlHandler(w http.ResponseWriter, r *http.Request) {
	var count int
	var err error
	var ip string
	var tmplHash map[string]string = make(map[string]string)

	if ip, _, err = net.SplitHostPort(r.RemoteAddr); err != nil {
		io.WriteString(w, "internal error: invalid parameter.")
		return
	}
	count, err = insertAndGetVisitorCount(ip)
	if err != nil {
		io.WriteString(w, err.Error())
		return
	}
	tmplHash["VisitorCount"] = strconv.Itoa(count)

	if r.Method == "GET" {
		indexT.ExecuteTemplate(w, "tinyurl", tmplHash)
	} else if r.Method == "POST" {
		r.ParseForm()
		url := r.Form["url"][0]

		newUrl, _ := getTinyUrl(url)

		var message string = "短地址：" + newUrl
		tmplHash["Message"] = message

		if nil != indexT.ExecuteTemplate(w, "tinyurl", tmplHash) {
			io.WriteString(w, "internal error: 502")
			return
		}
	}
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	//indexHandler(w, r)
	path := r.URL.Path
	fmt.Println(path)
	if path == "/" {
		io.WriteString(w, "hello boy.")
		return
	}

	var newPath string = "www.baidu.com"
	if path == "/abcde" {
		newPath = "https://www.baidu.com/s?wd=%E5%93%88%E5%93%88%E5%93%88%E5%93%88&rsv_spt=1&rsv_iqid=0x8d3ff876000032ae&issp=1&f=3&rsv_bp=1&rsv_idx=2&ie=utf-8&rqlang=cn&tn=baiduhome_pg&rsv_enter=1&oq=hhhh&inputT=2997&rsv_t=3dd7RQr4AlMzBIPfHVuNzVIaE8Cc0WWg81enKr2u0sKzRc2DFt%2BoUoOZVKsVOY%2FY4i8d&rsv_sug3=5&rsv_sug1=5&rsv_sug7=100&rsv_pq=b3f2ba48000046e6&rsv_sug2=1&prefixsug=hhhh&rsp=3&rsv_sug4=3384&rsv_sug=1"
	}

	http.Redirect(w, r, newPath, http.StatusFound)

	//w.WriteHeader(302)
	//w.Header().Set("Location", newPath)

	return
}

var yiJing []string = []string{"或跃在渊．无咎",
	"见龙在田．利见大人",
	"君子终日乾乾．夕惕若厉．无咎",
	"亢龙有悔．用九．见群龙．无首．吉",
	"地势坤．君子以厚德载物"}

func guaHandler(w http.ResponseWriter, r *http.Request) {
	var count int
	var err error
	var ip string
	var tmplHash map[string]string = make(map[string]string)

	if ip, _, err = net.SplitHostPort(r.RemoteAddr); err != nil {
		io.WriteString(w, "internal error: invalid parameter.")
		return
	}
	count, err = insertAndGetVisitorCount(ip)
	if err != nil {
		io.WriteString(w, err.Error())
		return
	}
	tmplHash["VisitorCount"] = strconv.Itoa(count)
	tmplHash["GuaGua"] = yiJing[rand.Intn(len(yiJing))]

	if r.Method == "GET" {
		guaT.ExecuteTemplate(w, "gua", tmplHash)
	}
}

func insertAndGetVisitorCount(ip string) (int, error) {
	var count int
	now := time.Now()
	if _, err := stmtIns.Exec(ip, now.Format("2006-01-02 15:04:05")); err != nil {
		return -1, fmt.Errorf("internal error x")
	}
	if nil != stmtCount.QueryRow().Scan(&count) {
		return -1, fmt.Errorf("internal error y")
	}

	return count, nil
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

	count, err = insertAndGetVisitorCount(ip)
	if err != nil {
		io.WriteString(w, err.Error())
		return
	}

	tmplHash["Country"] = clientCountry
	tmplHash["City"] = clientCity
	tmplHash["Ip"] = ip
	tmplHash["VisitorCount"] = strconv.Itoa(count)

	if r.Method == "GET" {
		indexT.ExecuteTemplate(w, "index", tmplHash)
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

		if nil != indexT.ExecuteTemplate(w, "index", tmplHash) {
			//if nil != indexT.Execute(w, tmplHash) {
			io.WriteString(w, "internal error: 502")
			return
		}
	}

	return
}
