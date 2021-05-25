package netway

import (
	"encoding/json"
	"errors"
	"frame_sync/myproto"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"regexp"
	"strconv"
	"strings"
	"zlib"
)

func(netWay *NetWay) Router(msg Message,wsConn *WsConn)(data interface{},err error){

	requestLogin := myproto.RequestLogin{}
	requestClientPong := myproto.RequestClientPong{}
	requestClientPing := myproto.RequestClientPing{}
	requestPlayerResumeGame := myproto.RequestPlayerResumeGame{}
	//requestPlayerStatus := RequestPlayerStatus{}
	requestPlayerOperations := myproto.RequestPlayerOperations{}
	requestPlayerMatchSign := myproto.RequestPlayerMatchSign{}
	requestPlayerMatchSignCancel := myproto.RequestPlayerMatchSignCancel{}
	requestGameOver := myproto.RequestGameOver{}
	requestClientHeartbeat := myproto.RequestClientHeartbeat{}
	requestPlayerReady := myproto.RequestPlayerReady{}
	requestRoomHistory := myproto.RequestRoomHistory{}
	requestGetRoom := myproto.RequestGetRoom{}
	requestPlayerOver := myproto.RequestPlayerOver{}

	//这里有个BUG，LOGIN 函数只能在第一次调用，回头加个限定


	switch msg.Action {
		case "login"://
			err = netWay.parserContentMsg(msg.Content,&requestLogin)
		case "clientPong"://
			err = netWay.parserContentMsg(msg.Content,&requestClientPong)
		case "playerResumeGame"://恢复未结束的游戏
			err = netWay.parserContentMsg(msg.Content,&requestPlayerResumeGame)
		//case "playerStatus"://玩家检测是否有未结束的游戏
		//	err = parserMsgContent(msg.Content,&requestPlayerStatus)
		//netWay.Players.getPlayerStatus(requestPlayerStatus,wsConn)
		case "playerOperations"://玩家推送操作指令
			err = netWay.parserContentMsg(msg.Content,&requestPlayerOperations)
		case "playerLogicFrameAck":
			//err = parserMsgContent(msg.Content,&logicFrame)
			//mySync.playerLogicFrameAck(logicFrame,wsConn)
		case "playerMatchSignCancel"://玩家取消报名等待
			err = netWay.parserContentMsg(msg.Content,&requestPlayerMatchSignCancel)
		case "gameOver"://游戏结束
			err = netWay.parserContentMsg(msg.Content,&requestGameOver)
		case "clientHeartbeat"://心跳
			err = netWay.parserContentMsg(msg.Content,&requestClientHeartbeat)
		case "playerMatchSign"://
			err = netWay.parserContentMsg(msg.Content,&requestPlayerMatchSign)
		case "playerReady"://玩家进入状态状态
			err = netWay.parserContentMsg(msg.Content,&requestPlayerReady)
		case "roomHistory"://一局副本的，所有历史操作记录
			err = netWay.parserContentMsg(msg.Content,&requestRoomHistory)
		case "getRoom"://
			err = netWay.parserContentMsg(msg.Content,&requestGetRoom)
		case "clientPing":
			err = netWay.parserContentMsg(msg.Content,&requestClientPing)
		case "playerOver":
			err = netWay.parserContentMsg(msg.Content,&requestPlayerOver)
		default:
			mylog.Error("Router err:",msg)
			return data,nil
	}
	if err != nil{
		return data,err
	}
	mylog.Info("Router ",msg.Action)
	switch msg.Action {
		case "login"://
			jwtData ,err := netWay.login(requestLogin,wsConn)
			return jwtData,err
		case "clientPong"://
			netWay.ClientPong(requestClientPong,wsConn)
		case "clientHeartbeat"://心跳
			netWay.heartbeat(requestClientHeartbeat,wsConn)
		case "playerMatchSign"://
			myMatch.addOnePlayer(requestPlayerMatchSign,wsConn)
		case "clientPing"://
			netWay.clientPing(requestClientPing,wsConn)
		case "playerResumeGame"://恢复未结束的游戏
			netWay.mySync.PlayerResumeGame(requestPlayerResumeGame,wsConn )
		case "playerOperations"://玩家推送操作指令
			netWay.mySync.ReceivePlayerOperation(requestPlayerOperations,wsConn,msg.Content)
		case "playerMatchSignCancel"://玩家取消报名等待
			myMatch.delOnePlayer(requestPlayerMatchSignCancel,wsConn)
		case "gameOver"://游戏结束
			netWay.mySync.GameOver(requestGameOver,wsConn)
		case "playerReady"://玩家进入状态状态
			netWay.mySync.PlayerReady(requestPlayerReady,wsConn)
		case "roomHistory"://一局副本的，所有历史操作记录
			netWay.mySync.RoomHistory(requestRoomHistory,wsConn)
		case "getRoom":
			netWay.mySync.GetRoom(requestGetRoom,wsConn)
		case "playerOver":
			netWay.mySync.PlayerOver(requestPlayerOver,wsConn)
		default:
			mylog.Error("Router err:",msg)

		//case "playerAddRoom"://玩家进入房间
	}

	return data,nil
}
//发送一条消息给一个玩家FD，同时将消息内容进行编码与压缩
func(netWay *NetWay)SendMsgCompressByConn(wsConn *WsConn,action string , contentStruct interface{}){
	mylog.Info("SendMsgCompressByConn ", "" , " action:",action)
	contentByte ,_ := netWay.CompressContent(contentStruct)
	netWay.SendMsg(wsConn,action,contentByte)
}
//发送一条消息给一个玩家FD，同时将消息内容进行编码与压缩
func(netWay *NetWay)SendMsgCompressByUid(uid int32,action string , contentStruct interface{}){
	mylog.Info("SendMsgCompressByUid uid:",uid , " action:",action)
	contentByte ,_ := netWay.CompressContent(contentStruct)
	netWay.SendMsgByUid(uid,action,contentByte)
}
func(netWay *NetWay)SendMsg(wsConn *WsConn,action string,content []byte){
	//获取协议号结构体
	actionMapT,empty := netWay.ProtocolActions.GetActionId(action,"server")
	mylog.Info("SendMsg",actionMapT.Id,wsConn.PlayerId,action)
	if empty{
		mylog.Error("GetActionId empty",action)
		return
	}
	//协议号
	strId := strconv.Itoa(actionMapT.Id)
	//合并 协议号 + 消息内容体
	content = zlib.BytesCombine([]byte(strId),content)
	if wsConn.Status == CONN_STATUS_CLOSE {
		mylog.Error("wsConn status =CONN_STATUS_CLOSE.")
		return
	}
	//myMetrics.IncNode("output_num")
	//myMetrics.PlusNode("output_size",len(content))
	//房间做统计处理
	if action =="pushLogicFrame"{
		roomId := netWay.PlayerManager.GetRoomIdByPlayerId(wsConn.PlayerId)
		roomSyncMetrics := RoomSyncMetricsPool[roomId]
		roomSyncMetrics.OutputNum++
		roomSyncMetrics.OutputSize = roomSyncMetrics.OutputSize + len(content)
	}
	if mynetWay.Option.ContentType == CONTENT_TYPE_PROTOBUF {
		wsConn.Write(content,websocket.BinaryMessage)
		//netWay.myWriteMessage(wsConn,websocket.BinaryMessage,content)
	}else{
		wsConn.Write(content,websocket.TextMessage)
		//netWay.myWriteMessage(wsConn,websocket.TextMessage,content)
	}
}
//发送一条消息给一个玩家FD，
func(netWay *NetWay)SendMsgByUid(playerId int32,action string , content []byte){
	wsConn,ok := wsConnManager.getConnPoolById(playerId)
	if !ok {
		mylog.Error("wsConn not in pool,maybe del.")
		return
	}
	netWay.SendMsg(wsConn,action,content)
}
//真的发送一条消息给sock FD
//func(netWay *NetWay) myWriteMessage(wsConn *WsConn ,msgCate int ,content []byte){
//	defer func() {
//		if err := recover(); err != nil {
//			mylog.Error("wsConn.Conn.WriteMessage failed:",err)
//			netWay.CloseOneConn(wsConn,CLOSE_SOURCE_SEND_MESSAGE)
//		}
//	}()
//	wsConn.Write(content,msgCate)
//}
//解析C端发送的数据，这一层只解析前4个字节，找到对应的action，对于content不做处理
func  (netWay *NetWay)parserContentProtocol(content string)(message Message,err error){
	if len(content)<4{
		return message,errors.New("content < 4")
	}

	actionIdStr := content[0:4]
	actionId,_ := strconv.Atoi(actionIdStr)
	actionName,empty := netWay.ProtocolActions.GetActionName(actionId,"client")
	if empty{
		errMsg := "actionId ProtocolActions.GetActionName empty!!!"
		mylog.Error(errMsg,actionId)
		return message,errors.New("actionId ProtocolActions.GetActionName empty!!!")
	}
	if len(content)==4{
		errMsg := "content = 4 ,body is empty"
		return message,errors.New(errMsg)
	}
	mylog.Info("parserContent",actionName.Action)

	msg := Message{
		Action: actionName.Action,
		Content: content[4:],
	}
	return msg,nil
}
//协议层的解包已经结束，这个时候需要将content内容进行转换成MSG结构
func (netWay *NetWay)parserContentMsg(content string ,out interface{})error{
	var err error
	if mynetWay.Option.ContentType == CONTENT_TYPE_JSON {
		unTrunVarJsonContent := zlib.CamelToSnake([]byte(content))
		err = json.Unmarshal(unTrunVarJsonContent,out)
	}else if  mynetWay.Option.ContentType == CONTENT_TYPE_PROTOBUF {
		aaa := out.(proto.Message)
		err = proto.Unmarshal([]byte(content),aaa)
	}else{
		mylog.Error("parserContent err")
	}

	if err != nil{
		mylog.Error("parserMsgContent:",err.Error())
		return err
	}

	mylog.Debug("netWay parserMsgContent:",out)

	return nil
}
//将 结构体 压缩成 字符串
func (netWay *NetWay)CompressContent(contentStruct interface{})(content []byte  ,err error){
	if mynetWay.Option.ContentType == CONTENT_TYPE_JSON {
		//这里有个问题：纯JSON格式与PROTOBUF格式在PB文件上 不兼容
		//严格来说是GO语言与protobuf不兼容，即：PB文件的  结构体中的 JSON-TAG
		//PROTOBUF如果想使用驼峰式变量名，即：成员变量名区分出大小写，那必须得用<下划线>分隔，编译后，下划线转换成大写字母
		//编译完成后，虽然支持了驼峰变量名，但json-tag 并不是驼峰式，却是<下划线>式
		//那么，在不想改PB文件的前提下，就得在程序中做兼容

		//所以，先将content 字符串 由下划线转成 驼峰式
		content, err = json.Marshal(JsonCamelCase{contentStruct})
		mylog.Info("CompressContent json:",string(content),err )
	}else if  mynetWay.Option.ContentType == CONTENT_TYPE_PROTOBUF {
		contentStruct := contentStruct.(proto.Message)
		content, err = proto.Marshal(contentStruct)
	}else{
		err = errors.New(" switch err")
	}
	if err != nil{
		mylog.Error("CompressContent err :",err.Error())
	}
	return content,err
}

type JsonCamelCase struct {
	Value interface{}
}
//下划线 转 驼峰命
func Case2Camel(name string) string {
	//将 下划线 转 空格
	name = strings.Replace(name, "_", " ", -1)
	//将 字符串的 每个 单词 的首字母转大写
	name = strings.Title(name)
	//最后再将空格删掉
	return strings.Replace(name, " ", "", -1)
}

func (c JsonCamelCase) MarshalJSON() ([]byte, error) {
	var keyMatchRegex = regexp.MustCompile(`\"(\w+)\":`)
	marshalled, err := json.Marshal(c.Value)
	converted := keyMatchRegex.ReplaceAllFunc(
		marshalled,
		func(match []byte) []byte {
			matchStr := string(match)
			key := matchStr[1 : len(matchStr)-2]
			resKey := zlib.Lcfirst(Case2Camel(key))
			return []byte(`"` + resKey + `":`)
		},
	)
	return converted, err
}
