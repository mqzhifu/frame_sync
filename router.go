package main

import (
	"encoding/json"
	"errors"
	"github.com/golang/protobuf/proto"
	"strconv"
)
//解析C端发送的数据，这一层只解析前4个字节，找到对应的action
func  (netWay *NetWay)parserContent(content string)(message Message,err error){
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


func (netWay *NetWay)parserMsgContent(content string ,out interface{})error{
	//mylog.Debug("netWay parserMsgContent start :",content," out ",out)
	//zlib.MyPrint("out:",out)
	var err error
	if mynetWay.Option.ContentType == CONTENT_TYPE_JSON{
		err = json.Unmarshal([]byte(content),out)
	}else if  mynetWay.Option.ContentType == CONTENT_TYPE_PROTOBUF{
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

func(netWay *NetWay) Router(msg Message,wsConn *WsConn)(data interface{},err error){

	requestLogin := RequestLogin{}
	requestClientPong :=RequestClientPong{}
	requestClientPing := RequestClientPing{}
	requestPlayerResumeGame := RequestPlayerResumeGame{}
	//requestPlayerStatus := RequestPlayerStatus{}
	requestPlayerOperations := RequestPlayerOperations{}
	requestPlayerMatchSign := RequestPlayerMatchSign{}
	requestPlayerMatchSignCancel := RequestPlayerMatchSignCancel{}
	requestGameOver := RequestGameOver{}
	requestClientHeartbeat := RequestClientHeartbeat{}
	requestPlayerReady := RequestPlayerReady{}
	requestRoomHistory := RequestRoomHistory{}
	requestGetRoom := RequestGetRoom{}
	requestPlayerOver := RequestPlayerOver{}

	//这里有个BUG，LOGIN 函数只能在第一次调用，回头加个限定


	switch msg.Action {
		case "login"://
			err = netWay.parserMsgContent(msg.Content,&requestLogin)
		case "clientPong"://
			err = netWay.parserMsgContent(msg.Content,&requestClientPong)
		case "playerResumeGame"://恢复未结束的游戏
			err = netWay.parserMsgContent(msg.Content,&requestPlayerResumeGame)
		//case "playerStatus"://玩家检测是否有未结束的游戏
		//	err = parserMsgContent(msg.Content,&requestPlayerStatus)
			//netWay.Players.getPlayerStatus(requestPlayerStatus,wsConn)
		case "playerOperations"://玩家推送操作指令
			err = netWay.parserMsgContent(msg.Content,&requestPlayerOperations)
		case "playerLogicFrameAck":
			//err = parserMsgContent(msg.Content,&logicFrame)
			//mySync.playerLogicFrameAck(logicFrame,wsConn)
		case "playerCancelReady"://玩家取消报名等待
			err = netWay.parserMsgContent(msg.Content,&requestPlayerMatchSignCancel)
		case "gameOver"://游戏结束
			err = netWay.parserMsgContent(msg.Content,&requestGameOver)
		case "clientHeartbeat"://心跳
			err = netWay.parserMsgContent(msg.Content,&requestClientHeartbeat)
		case "playerMatchSign"://
			err = netWay.parserMsgContent(msg.Content,&requestPlayerMatchSign)
		case "playerReady"://玩家进入状态状态
			err = netWay.parserMsgContent(msg.Content,&requestPlayerReady)
		case "roomHistory"://一局副本的，所有历史操作记录
			err = netWay.parserMsgContent(msg.Content,&requestRoomHistory)
		case "getRoom"://
			err = netWay.parserMsgContent(msg.Content,&requestGetRoom)
		case "clientPing":
			err = netWay.parserMsgContent(msg.Content,&requestClientPing)
		case "playerOver":
			err = netWay.parserMsgContent(msg.Content,&requestPlayerOver)
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
			netWay.playerMatchSign(requestPlayerMatchSign,wsConn)
		case "clientPing"://
			netWay.clientPing(requestClientPing,wsConn)
		case "playerResumeGame"://恢复未结束的游戏
			mySync.playerResumeGame(requestPlayerResumeGame,wsConn )
		case "playerOperations"://玩家推送操作指令
		//zlib.MyPrint(requestPlayerOperations.Operations)
		//zlib.ExitPrint(requestPlayerOperations)
			mySync.receivePlayerOperation(requestPlayerOperations,wsConn)
		case "playerCancelReady"://玩家取消报名等待
			mySync.cancelSign(requestPlayerMatchSignCancel,wsConn)
		case "gameOver"://游戏结束
			mySync.gameOver(requestGameOver,wsConn)
		case "playerReady"://玩家进入状态状态
			mySync.playerReady(requestPlayerReady,wsConn)
		case "roomHistory"://一局副本的，所有历史操作记录
			mySync.RoomHistory(requestRoomHistory,wsConn)
		case "getRoom"://
			mySync.GetRoom(requestGetRoom,wsConn)
		case "playerOver":
			mySync.playerOver(requestPlayerOver,wsConn)
		default:
			mylog.Error("Router err:",msg)

		//case "playerStatus"://玩家检测是否有未结束的游戏
		//	netWay.Players.getPlayerStatus(requestPlayerStatus,wsConn)
		//case "playerLogicFrameAck":
		//mySync.playerLogicFrameAck(logicFrame,wsConn)
		//case "netClose"://网络异常断开，也可能是主动断开
		//case "clientPreClose"://C端主动断开连接前，提前通知
		//	netWay.CloseOneConn(wsConn,CLOSE_SOURCE_CLIENT_PRE)
		//case "playerGameOver"://玩家的某些操作，触发了该玩家挂了
		//case "playerAddRoom"://玩家进入房间
		//case "gameStart"://所有-玩家均进入准备状态，点击'开始按钮'，触发游戏开始事件
	}

	return data,nil
}




//func RouterJsonUnmarshal(content string ,out interface{})error{
//	err := json.Unmarshal([]byte(content),out)
//	if err != nil{
//		mylog.Error("RouterJsonUnmarshal err ",err.Error())
//		return err
//	}
//	return nil
//}
