package myprotocol

import (
	"fmt"
	"path"
	"runtime"
	"strings"
	"zlib"
)

type ProtocolActions struct {
	//并发安全的，因为无写操作
	ActionMaps map[string]map[int32]ActionMap
	Log *zlib.Log
}

type ActionMap struct {
	Id 		int32		`json:"id"`
	Action	string		`json:"action"`
	Desc 	string		`json:"desc"`
	Demo 	string		`json:"demo"`
}

//var actionMap  	map[string]map[int]ActionMap
func NewProtocolActions(log *zlib.Log)*ProtocolActions {
	log.Info("NewProtocolActions instance:")
	protocolActions := new(ProtocolActions)
	protocolActions.Log = log
	protocolActions.initProtocolActionMap()
	return protocolActions
}

func (protocolActions *ProtocolActions)initProtocolActionMap(){
	//netway.mylog.Info("initActionMap")
	actionMap := make( 	map[string]map[int32]ActionMap)

	actionMap["client"] = protocolActions.loadingActionMapConfigFile("clientActionMap.txt")
	actionMap["server"] = protocolActions.loadingActionMapConfigFile("serverActionMap.txt")

	protocolActions.ActionMaps = actionMap
}
func  (protocolActions *ProtocolActions)loadingActionMapConfigFile(fileName string)map[int32]ActionMap {
	_, _,_,dir  := getInfo(1)
	client,err := zlib.ReadLine(dir +"/"+fileName)
	if err != nil{
		protocolActions.Log.Error("initActionMap ReadLine err :",err.Error())
		zlib.PanicPrint("initActionMap ReadLine err :",err.Error())
	}
	am := make(map[int32]ActionMap)
	for _,v:= range client{
		contentArr := strings.Split(v,"|")
		if len(contentArr)  <  4{
			protocolActions.Log.Error("read line len < 4:",contentArr)
			continue
		}
		id := int32(zlib.Atoi(contentArr[1]))
		actionMap := ActionMap{
			Id: id,
			Action: contentArr[2],
			Desc: contentArr[3],
			Demo: contentArr[4],
		}
		am[id] = actionMap
	}
	if len(am) <= 0{
		protocolActions.Log.Error("protocolActions len(am) <= 0")
		zlib.PanicPrint("protocolActions len(am) <= 0")
	}
	return am
}
//获取上层调用者的文件位置
func getInfo(depth int) (funcName, fileName string, lineNo int ,dir string) {
	pc, file, lineNo, ok := runtime.Caller(depth)
	if !ok {
		fmt.Println("runtime.Caller() failed")
		return
	}
	funcName = runtime.FuncForPC(pc).Name()
	fileName = path.Base(file) // Base函数返回路径的最后一个元素

	i := strings.LastIndex(file, "/")
	//if i < 0 {
	//	i = strings.LastIndex(path, "\\")
	//}
	//if i < 0 {
	//	return "", errors.New(`error: Can't find "/" or "\".`)
	//}
	dir = string(file[0 : i+1])
	return
}

func(protocolActions *ProtocolActions)GetActionMap()map[string]map[int32]ActionMap {
	return protocolActions.ActionMaps
}

func(protocolActions *ProtocolActions)GetActionName(id int32,category string)(actionMapT ActionMap,empty bool){
	am := protocolActions.ActionMaps[category]
	for k,v:=range am{
		if k == id {
			return v,false
		}
	}
	return  actionMapT,true
}

func  (protocolActions *ProtocolActions)GetActionId(action string,category string)(actionMapT ActionMap,empty bool){
	//netway.mylog.Info("GetActionId ",action , " ",category)
	am := protocolActions.ActionMaps[category]
	for _,v:=range am{
		if v.Action == action {
			return v,false
		}
	}
	return  actionMapT,true
}
