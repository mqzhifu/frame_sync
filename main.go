package main

import (
	"fmt"
	"golang.org/x/net/websocket"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"zlib"
)

var mylog *zlib.Log
func routeHandler(ws *websocket.Conn) {
	mylog.Debug("11111")
	msg := make([]byte, 512)
	n, err := ws.Read(msg)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Receive: %s\n", msg[:n])

	send_msg := "[" + string(msg[:n]) + "]"
	m, err := ws.Write([]byte(send_msg))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Send: %s\n", msg[:m])
}

func  wwwHandler(w http.ResponseWriter, r *http.Request){
	//parameter := r.URL.Query()//GET 方式URL 中的参数 转 结构体
	uri := r.URL.RequestURI()
	mylog.Info("uri:",uri)
	if uri == "" || uri == "/" {
		ResponseStatusCode(w,500,"RequestURI is null or uir is :  '/'")
		return
	}

	////去掉 URI 中最后一个 /
	//uriLen := len(uri)
	//if string([]byte(uri)[uriLen-1:uriLen]) == "/"{
	//	uri = string([]byte(uri)[0:uriLen - 1])
	//}
	//mylog.Info("final uri : ",uri , " start routing ...")
	////*********: 还没有加  v1  v2 版本号
	//code := 200
	//var msg interface{}
	//先匹配下静态资源
	routeStatic(w,r,uri)
}

func  getStaticFileContent(fileSuffix string)(content string ,err error){
	//code,msg = httpd.redisMetrics()
	baseDir ,_ := os.Getwd()
	//path := baseDir + "/../gamematch/www"
	path := baseDir
	mylog.Debug("final path:",path)
	filePath := path+fileSuffix
	mylog.Info("getStaticFileContent File path:",filePath)
	b, err := ioutil.ReadFile(filePath) // just pass the file name
	return string(b),err
}

func  routeStatic(w http.ResponseWriter,r *http.Request,uri string){
	uriSplit := strings.Split(uri,"?")
	if uriSplit[0] == "/apireq.html" {
		uri = uriSplit[0]
	}
	if uri == "/demo.html" { //静态文件
		fileContent, err := getStaticFileContent(uri)
		if err != nil {
			ResponseStatusCode(w, 404, err.Error())
			return
		}
		//踦域处理
		w.Header().Set("Access-Control-Allow-Origin","*")
		w.Header().Add("Access-Control-Allow-Headers","Content-Type")
		//w.Header().Set("content-type","text/plain")

		w.Header().Set("Content-Length", strconv.Itoa(len(fileContent)))
		w.Write([]byte(fileContent))
	}
}

//http 响应状态码
func  ResponseStatusCode(w http.ResponseWriter,code int ,responseInfo string){
	mylog.Info("ResponseStatusCode",code,responseInfo)

	w.Header().Set("Content-Length",strconv.Itoa( len(responseInfo) ) )
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(403)
	w.Write([]byte(responseInfo))
}

func main(){
	//http://code.google.com/p/go.net/websocket
	//http://code.google.com/p/go.net/websocket
	//http://golang.org/x/net/websocket
	//github.com/gorilla/websocket
	//http://github.com/golang/net

	zlib.LogLevelFlag = zlib.LOG_LEVEL_DEBUG
	logOption := zlib.LogOption{
		OutFilePath : "/data/www/golang/src/logs",
		OutFileName: "frame_sync.log",
		Level : 511,
		Target : 7,
	}
	newlog,errs  := zlib.NewLog(logOption)
	if errs != nil{
		zlib.ExitPrint("new log err",errs.Error())
	}

	mylog = newlog


	http.Handle("/ws", websocket.Handler(routeHandler))
	http.HandleFunc("/",wwwHandler)
	mylog.Info("ok")
	if err := http.ListenAndServe("127.0.0.1:1111", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
