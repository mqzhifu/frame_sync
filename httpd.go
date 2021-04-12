package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
)

type MyServer struct {
	Host 		string
	Port 		string
	MapSize 	int
	RoomPeople 	int
	Uri			string
}

func  wwwHandler(w http.ResponseWriter, r *http.Request){
	//parameter := r.URL.Query()//GET 方式URL 中的参数 转 结构体
	uri := r.URL.RequestURI()
	mylog.Info("uri:",uri)
	if uri == "" || uri == "/" {
		ResponseStatusCode(w,500,"RequestURI is null or uir is :  '/'")
		return
	}

	if uri == "/www/getServer"{
		myServer := MyServer{
			Host: mynetWay.Option.Host,
			Port: mynetWay.Option.Port,
			MapSize: mynetWay.Option.MapSize,
			RoomPeople: mynetWay.Option.RoomPeople,
			Uri: mynetWay.Option.WsUri,
		}
		jsonStr,_ := json.Marshal(&myServer)

		w.Header().Set("Content-Length",strconv.Itoa( len(jsonStr) ) )
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(jsonStr)
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
	//uriSplit := strings.Split(uri,"?")
	//if uriSplit[0] == "/apireq.html" {
	//	uri = uriSplit[0]
	//}
	if uri == "/www/ws.html" ||  uri == "/www/jquery.min.js"||  uri == "/www/sync.js"{ //静态文件
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