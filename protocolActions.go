package main

import (
	"strings"
	"zlib"
)

type ProtocolActions struct {
	ActionMaps map[string]map[int]ActionMap
}

type ActionMap struct {
	Id 		int
	Action	string
	Desc 	string
	Demo 	string
}

//var actionMap  	map[string]map[int]ActionMap
func ProtocolActionsNew()*ProtocolActions{
	mylog.Info("New ProtocolAction instance")
	protocolActions := new(ProtocolActions)
	protocolActions.initProtocolActionMap()
	return protocolActions
}

func (protocolActions *ProtocolActions)initProtocolActionMap(){
	mylog.Info("initActionMap")
	actionMap := make( 	map[string]map[int]ActionMap )

	actionMap["client"] = loadingActionMapConfigFile("clientActionMap.txt")
	actionMap["server"] = loadingActionMapConfigFile("serverActionMap.txt")

	protocolActions.ActionMaps = actionMap
}

func loadingActionMapConfigFile(fileName string)map[int]ActionMap{
	client,err := zlib.ReadLine(fileName)
	if err != nil{
		zlib.ExitPrint("initActionMap ReadLine err :",err.Error())
	}
	am := make(map[int]ActionMap)
	for _,v:= range client{
		contentArr := strings.Split(v,"|")
		id := zlib.Atoi(contentArr[1])
		//zlib.ExitPrint(id)
		actionMap := ActionMap{
			Id: id,
			Action: contentArr[2],
			Desc: contentArr[3],
			Demo: contentArr[4],
		}
		am[id] = actionMap
	}
	return am
}


func(protocolActions *ProtocolActions)getActionMap()map[string]map[int]ActionMap{
	return protocolActions.ActionMaps
}

func(protocolActions *ProtocolActions)GetActionName(id int,category string)(actionMapT ActionMap,empty bool){
	am := protocolActions.ActionMaps[category]
	for k,v:=range am{
		if k == id {
			return v,false
		}
	}
	return  actionMapT,true
}

func  (protocolActions *ProtocolActions)GetActionId(action string,category string)(actionMapT ActionMap,empty bool){
	mylog.Info("GetActionId ",action , " ",category)
	am := protocolActions.ActionMaps[category]
	for _,v:=range am{
		if v.Action == action {
			return v,false
		}
	}
	return  actionMapT,true
}
