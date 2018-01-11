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
	"math"
	"strings"
	"io/ioutil"
	"encoding/json"
	"encoding/base64"
	"os"
	"encoding/csv"
)

var q *qqwry.QQwry
var indexT, guaT, tinyUrlT, macT, picT *template.Template
var urlInsert, urlUpdate, urlSelect, stmtIns, stmtCount *sql.Stmt
var ALPHABET string = "23456789bcdfghjkmnpqrstvwxyzBCDFGHJKLMNPQRSTVWXYZ-_"
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
	if err != nil {
		panic(err.Error())
	}
	defer urlInsert.Close()

	urlUpdate, err = db.Prepare("UPDATE shorturl SET short_url = ? WHERE id = ?")
	if err != nil {
		panic(err.Error())
	}
	defer urlUpdate.Close()

	urlSelect, err = db.Prepare("SELECT long_url FROM shorturl WHERE id = ?")
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

	macDBInit()

	indexT, _ = template.ParseFiles("tmpl/head.html", "tmpl/header.html", "tmpl/index.html", "tmpl/footer.html")
	guaT, _ = template.ParseFiles("tmpl/head.html", "tmpl/header.html", "tmpl/gua.html", "tmpl/footer.html")
	tinyUrlT, _ = template.ParseFiles("tmpl/head.html", "tmpl/header.html", "tmpl/tinyurl.html", "tmpl/footer.html")
	macT, _ = template.ParseFiles("tmpl/head.html", "tmpl/header.html", "tmpl/mac.html", "tmpl/footer.html")
	picT, _ = template.ParseFiles("tmpl/head.html", "tmpl/header.html", "tmpl/pic.html", "tmpl/footer.html")

	http.HandleFunc("/index", indexHandler)
	http.HandleFunc("/gua", guaHandler)
	http.HandleFunc("/tinyurl", tinyurlHandler)
	http.HandleFunc("/mac", macAddrHandler)
	http.HandleFunc("/pic", picHandler)
	//http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/", defaultHandler)

	var address string = fmt.Sprintf(":%d", port)

	var stopCh chan int = make(chan int)
	go func() {
		err = http.ListenAndServe(address, nil)
		if err != nil {
			log.Fatal("server start failed.", err)
		}
		stopCh <- 1
	}()

	fmt.Printf("start http server at %s\n", address)
	<- stopCh

}

func picHandler(w http.ResponseWriter, r *http.Request) {
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
		picT.ExecuteTemplate(w, "pic", tmplHash)
	} else if r.Method == "POST" {
		r.ParseMultipartForm(32 << 20)
		file, handler, err := r.FormFile("pic")
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()

		buf := make([]byte, handler.Size)
		file.Read(buf)

		imgBase64Str := base64.StdEncoding.EncodeToString(buf)

		// Embed into an html without PNG file
		//img2html := "<html><body><img src=\"data:image/png;base64," + imgBase64Str + "\" /></body></html>"
		//io.WriteString(w, img2html)

		message := "data:image/png;base64," + imgBase64Str
		tmplHash["Message"] = message
		tmplHash["Image"] = message

		if nil != picT.ExecuteTemplate(w, "pic", tmplHash) {
			io.WriteString(w, "internal error: 502")
			return
		}

	}
}

type org struct {
	name string
	address string
}
type macInfo struct {
	mac 	string
	org
}
var macVendorHashMap map[string]org

func macDBInit() error {
	f, err:= os.Open("oui.csv")
	if err != nil {
		panic("file open failed")
	}
	defer f.Close()

	r := csv.NewReader(f)
	macVendorHashMap = make(map[string]org);
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		var o org = org{name:record[2], address:record[3]}
		macVendorHashMap[record[1]] = o
	}

	return nil
}

// get vendor info from local database
func getMacInfoLocal(macAddress string) (*macInfo, error) {
	newMac := strings.Replace(strings.ToUpper(macAddress), ":", "", -1)
	begin := time.Now()
	v, ok := macVendorHashMap[newMac[:6]]
	if !ok {
		return nil, fmt.Errorf("%s not found.", macAddress)
	}
	fmt.Println(time.Now().Sub(begin))

	var info *macInfo = &macInfo{mac:macAddress, org:org{name:v.name, address:v.address}}

	return info, nil
}

// get vendor info from macvendors.co
func getMacInfo(macAddress string) (*macInfo, error) {
	var macInfo *macInfo = &macInfo{}
	macInfo.mac = macAddress

	resp, err := http.Get(fmt.Sprintf("https://macvendors.co/api/%s/", macAddress))
	if err!= nil {
		return nil, err
	}
	defer resp.Body.Close()

	//result := []byte(`{"result":{"company":"Apple, Inc.","mac_prefix":"08:74:02","address":"1 Infinite Loop,Cupertino  CA  95014,US","start_hex":"087402000000","end_hex":"087402FFFFFF","country":"US","type":"MA-L"}}`)
	result, err := ioutil.ReadAll(resp.Body)
	if err!= nil {
		return nil, err
	}
	var rat map[string]map[string]string
	if err := json.Unmarshal(result, &rat); err != nil {
        return nil, err
    }

	macInfo.org.name = rat["result"]["company"]
	macInfo.org.address = rat["result"]["address"]

	return macInfo, nil
}

func macAddrHandler(w http.ResponseWriter, r *http.Request) {
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
		macT.ExecuteTemplate(w, "mac", tmplHash)
	} else if r.Method == "POST" {
		r.ParseForm()
		macAddress := r.Form["mac"][0]

		//macInfo, err := getMacInfo(macAddress)
		macInfo, err := getMacInfoLocal(strings.TrimSpace(macAddress))
		if err != nil {
			tmplHash["Message"] = err.Error()
			//io.WriteString(w, "internal error: 502")
			//return
		} else {
			message := fmt.Sprintf("%s 的厂家是：%s  厂家地址是：%s", macAddress, macInfo.org.name, macInfo.org.address)
			tmplHash["Message"] = message
		}

		if nil != macT.ExecuteTemplate(w, "mac", tmplHash) {
			io.WriteString(w, "internal error: 502")
			return
		}
	}
}

func encode(num int64) string {
	var b bytes.Buffer
	for num > 0 {
		b.WriteByte(ALPHABET[num%BASE])
		num = num / BASE
	}
	return b.String()
}

func decode(str string) int64 {
	var id int64 = 0
	var size int64 = int64(len(str))
	var i int64
	for i = 0; i < size; i++ {
		var value int64 = int64(strings.IndexByte(ALPHABET, str[i]))
		id += value * int64(math.Pow(float64(BASE), float64(size-i-1)))
	}

	return id
}

func getTinyUrl(url string) (string, error) {
	hasher := md5.New()
	hasher.Write([]byte(url))

	hex.EncodeToString(hasher.Sum(nil))

	now := time.Now()
	if res, err := urlInsert.Exec(url, "", now.Format("2006-01-02 15:04:05")); err == nil {
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
		tinyUrlT.ExecuteTemplate(w, "tinyurl", tmplHash)
	} else if r.Method == "POST" {
		r.ParseForm()
		url := r.Form["url"][0]

		newUrl, _ := getTinyUrl(url)
		message := url + " 生成的短链接为：" + r.Host + "/" + newUrl
		//message := fmt.Sprintf(`<a href="%s" target="_blank">%s</a>`, newUrl, newUrl)

		tmplHash["Message"] = message

		if nil != tinyUrlT.ExecuteTemplate(w, "tinyurl", tmplHash) {
			io.WriteString(w, "internal error: 502")
			return
		}
	}
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if path == "/" {
		http.Redirect(w, r, "/index", http.StatusFound)
		return
	} else {
		path = path[1:]
		longUrl := decode(path)
		var newPath string
		if nil != urlSelect.QueryRow(longUrl).Scan(&newPath) {
			return
		}
		http.Redirect(w, r, newPath, http.StatusFound)
	}

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
	fmt.Println(ip)
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
			io.WriteString(w, "internal error: 502")
			return
		}
	}

	return
}
