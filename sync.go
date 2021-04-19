package main

import (
	"encoding/json"
	"strconv"
	"zlib"
)

type Sync struct {

}
//集合中的：每个副本元素
type SyncRoomPoolElement struct {
	Status 				int
	Room 				*Room
	SequenceNumber		int
	PlayersAckList		map[int]int
	PlayersAckStatus	int
	AddTime 			int
	RandSeek			int
	LogicFrameHistory 	[]LogicFrameHistory
}
type LogicFrameHistory struct {
	Action string
	Content string
}
//逻辑帧
type LogicFrame struct {
	RoomId 			string		`json:"roomId"`
	SequenceNumber 	int			`json:"sequenceNumber"`
	Commands 		[]Command	`json:"commands"`
}
//玩家操作指令集
type Command struct {
	Action 		string	`json:"action"`
	Value 		string	`json:"value"`
	PlayerId 	int		`json:"playerId"`
}
//集合 ：每个副本
var mySyncRoomPool map[string]*SyncRoomPoolElement
var mySyncPlayerRoom map[int]string
func NewSync()*Sync{
	mySyncRoomPool = make(map[string]*SyncRoomPoolElement)
	mySyncPlayerRoom = make(map[int]string)
	sync := new(Sync)
	return sync
}
//给集合添加一个新的 游戏副本
func (sync *Sync) addPoolElement(room 	*Room){
	mynetWay.Option.Mylog.Info("addPoolElement")
	_ ,exist := mySyncRoomPool[room.Id]
	if exist{
		mynetWay.Option.Mylog.Error("mySyncRoomPool has exist : ",room.Id)
		return
	}
	syncRoomPoolElement := SyncRoomPoolElement{
		Status: SYNC_ELEMENT_STATUS_WAIT,
		Room: room,
		SequenceNumber: 0,
		PlayersAckList: make(map[int]int),
		PlayersAckStatus:PLAYERS_ACK_STATUS_INIT,
		AddTime: zlib.GetNowTimeSecondToInt(),
		RandSeek: zlib.GetRandIntNum(100),

	}
	mySyncRoomPool[room.Id] = &syncRoomPoolElement
}

func  (sync *Sync)start(roomId string){
	mynetWay.Option.Mylog.Warning("start a new game:")

	syncRoomPoolElement,_ := sync.getPoolElementById(roomId)
	syncRoomPoolElement.Status = SYNC_ELEMENT_STATUS_EXECING
	syncRoomPoolElement.Room.Status = ROOM_STATUS_WAIT_EXECING
	//sync.upRoomPoolElementStatus(roomId   ,ROOM_STATUS_WAIT_EXECING )
	clientInitData := ResponseClientInitData{
		RoomId:         roomId,
		SequenceNumber: 0,
		PlayerList:     syncRoomPoolElement.Room.PlayerList,
		RandSeek:       syncRoomPoolElement.RandSeek,
	}

	for _,v :=range syncRoomPoolElement.Room.PlayerList{
		mySyncPlayerRoom[v.Id] = roomId
	}

	jsonStrByte,_ := json.Marshal(clientInitData)
	sync.boardCastFrameInRoom(roomId,"startInit",string(jsonStrByte));

}
//有一个房间内，搜索一个用户
func (sync *Sync)getPlayerByIdInRoom(playerId int ,room *Room,)(myplayer *Player,empty bool){
	for _,player:= range room.PlayerList{
		if player.Id == playerId{
			return player,false
		}
	}
	return myplayer,true
}

func (sync *Sync) getPoolElementById(roomId string)(SyncRoomPoolElement *SyncRoomPoolElement,empty bool){
	v,exist := mySyncRoomPool[roomId]
	if !exist{
		mynetWay.Option.Mylog.Error("getPoolElementById is empty,",roomId)
		return SyncRoomPoolElement,true
	}
	return v,false
}

func  (sync *Sync)receiveCommand(logicFrame LogicFrame,wsConn *WsConn){
	//logicFrame := LogicFrame{}
	//err := json.Unmarshal([]byte(content),&logicFrame)
	//if err != nil{
	//	mynetWay.Option.Mylog.Error("receiveCommand Unmarshal ",err.Error())
	//	return
	//}

	_,empty := sync.getPoolElementById(logicFrame.RoomId)
	if empty{
		mynetWay.Option.Mylog.Error("getPoolElementById is empty",logicFrame.RoomId)
	}
	//element.LogicFramePlayerAck = make(map[int]int)
	//复用玩家发来的logicFrame结构内容即可
	//logicFrame.SequenceNumber = element.LogicFrameSequenceNumber
	//for _,player:= range element.Room.PlayerList{
		logicFrameMsgJson ,_ := json.Marshal(logicFrame)
		//msg := Msg{
		//	Action: "logicFrame",
		//	Content: string(logicFrameMsgJson),
		//}
		//element.LogicFramePlayerAck[player.Id] = 0
		//mynetWay.SendMsgByUid(player.Id,msg)
	//}

	sync.boardCastFrameInRoom(logicFrame.RoomId,"pushLogicFrame",string(logicFrameMsgJson))
}

func (sync *Sync) playerLogicFrameAck(logicFrame LogicFrame,wsConn *WsConn ){
	//logicFrame := LogicFrame{}
	//err := json.Unmarshal([]byte(content),&logicFrame)
	//if err != nil{
	//	mynetWay.Option.Mylog.Error("receiveCommand Unmarshal ",err.Error())
	//	return
	//}

	syncRoomPoolElement,empty := sync.getPoolElementById(logicFrame.RoomId)
	if empty{
		mynetWay.Option.Mylog.Error("roomId is wrong",logicFrame.RoomId)
		return
	}
	//if syncRoomPoolElement.SequenceNumber != logicFrame.SequenceNumber{
	//	mynetWay.Option.Mylog.Error("SequenceNumber wrong",logicFrame.SequenceNumber,syncRoomPoolElement.SequenceNumber)
	//	return
	//}
	if syncRoomPoolElement.PlayersAckStatus != PLAYERS_ACK_STATUS_WAIT{
		mynetWay.Option.Mylog.Error("该帧号的状态!=待确认 状态,当前状态为：",syncRoomPoolElement.PlayersAckStatus)
		return
	}

	player ,empty := sync.getPlayerByIdInRoom(wsConn.PlayerId,syncRoomPoolElement.Room)
	if empty{
		mynetWay.Option.Mylog.Error("playerId wrong,not found in this room.",wsConn.PlayerId)
		return
	}
	if syncRoomPoolElement.PlayersAckList[player.Id] == 1{
		mynetWay.Option.Mylog.Error("该玩家已经确认过了，不要重复操作")
		return
	}else{
		syncRoomPoolElement.PlayersAckList[player.Id] = 1
	}

	ack := 0
	for _,v := range syncRoomPoolElement.PlayersAckList{
		if v == 1{
			ack++
		}
	}
	mynetWay.Option.Mylog.Info("this room LogicFrameSequenceNumber(",syncRoomPoolElement.SequenceNumber,") ack total:",ack)

	//该逻辑帧已全部确认
	if ack == len(syncRoomPoolElement.PlayersAckList){
		mynetWay.Option.Mylog.Info("one logic frame all ack.")
		//更新 帧号 变更状态
		syncRoomPoolElement.SequenceNumber++
		syncRoomPoolElement.PlayersAckList = make(map[int]int)
		sync.upSyncRoomPoolElementPlayersAckStatus(logicFrame.RoomId,PLAYERS_ACK_STATUS_INIT)
		//syncRoomPoolElement.PlayersAckStatus = PLAYERS_ACK_STATUS_INIT
		mylog.Notice("have a new SequenceNumber")
		//这里有个特殊处理，首帧其实是初始化的一些数据，当完成后，第一帧的动作按说应该是玩家操作
		//但这里，先模拟一下，将玩家随机散落到一地图上的一些点
		if syncRoomPoolElement.SequenceNumber == 1{
			mynetWay.Option.Mylog.Info("syncRoomPoolElement.SequenceNumber == 0")
			//S端，每一逻辑帖，建立一个集合，保存广播的消息，玩家返回的确认ACK
			commands := []Command{}
			for _,player:= range syncRoomPoolElement.Room.PlayerList{
				location := strconv.Itoa(zlib.GetRandIntNum(mynetWay.Option.MapSize)) + "," + strconv.Itoa(zlib.GetRandIntNum(mynetWay.Option.MapSize))
				command := Command{
					Action: "move",
					Value: location,
					PlayerId: player.Id,
				}
				commands = append(commands,command)
			}
			logicFrameMsg := LogicFrame{
				RoomId: logicFrame.RoomId,
				SequenceNumber :syncRoomPoolElement.SequenceNumber,
				Commands 		:commands,
			}
			logicFrameMsgJson ,_ := json.Marshal(logicFrameMsg)
			sync.boardCastFrameInRoom(logicFrame.RoomId,"pushLogicFrame",string(logicFrameMsgJson))
		}
	}

	mynetWay.Option.Mylog.Info("playerLogicFrameAck~ finish.")
	return

}

func  (sync *Sync)gameOver(requestGameOver RequestGameOver,wsConn *WsConn){
	//logicFrame := LogicFrame{}
	//err := json.Unmarshal([]byte(content),&logicFrame)
	//if err != nil{
	//	mynetWay.Option.Mylog.Error("receiveCommand Unmarshal ",err.Error())
	//	return
	//}

	syncRoomPoolElement,empty := sync.getPoolElementById(requestGameOver.RoomId)
	if empty{
		mynetWay.Option.Mylog.Error("getPoolElementById is empty",requestGameOver.RoomId)
	}

	//responseGameOver := ResponseGameOver{}
	jsonStr ,_ := json.Marshal(requestGameOver)
	sync.boardCastFrameInRoom(requestGameOver.RoomId,"gameOver",string(jsonStr))

	syncRoomPoolElement.Status = SYNC_ELEMENT_STATUS_END
	for _,v:= range syncRoomPoolElement.Room.PlayerList{
		delete(mySyncPlayerRoom,v.Id)
	}
	//syncRoomPoolElement.Room.Status = ROOM_STATUS_WAIT_END
}
func (sync *Sync)upSyncRoomPoolElementPlayersAckStatus(roomId string,status int){
	syncRoomPoolElement,_  := sync.getPoolElementById(roomId)
	mylog.Notice("upSyncRoomPoolElementPlayersAckStatus ,old :",syncRoomPoolElement.PlayersAckStatus,"new",status)
	syncRoomPoolElement.PlayersAckStatus = status
}

func (sync *Sync)cancelSign(requestCancelSign RequestCancelSign,wsConn *WsConn){
	myMatch.delOnePlayer(wsConn.PlayerId)
}
func (sync *Sync)close(wsConn *WsConn){
	mylog.Warning("sync.close")
	roomId,ok := mySyncPlayerRoom[wsConn.PlayerId]
	if !ok {
		return
	}

	mynetWay.Players.upPlayerPool(wsConn.PlayerId,PLAYER_STATUS_OFFLINE)
	responseOtherPlayerOffline := ResponseOtherPlayerOffline{
		PlayerId: wsConn.PlayerId,
	}
	jsonStr,_ := json.Marshal(responseOtherPlayerOffline)

	sync.boardCastFrameInRoom(roomId,"otherPlayerOffline",string(jsonStr))
}
//给一个副本里的所有玩家广播数据，且该数据必须得有C端ACK
func  (sync *Sync)boardCastFrameInRoom(roomId string,action string ,content string){
	mylog.Notice("boardCastFrameInRoom:",roomId,action)
	syncRoomPoolElement,_  := sync.getPoolElementById(roomId)
	if syncRoomPoolElement.PlayersAckStatus == PLAYERS_ACK_STATUS_WAIT{
		mylog.Error("syncRoomPoolElement PlayersAckStatus = ",PLAYERS_ACK_STATUS_WAIT,syncRoomPoolElement.PlayersAckList)
		zlib.ExitPrint(11111)
	}
	PlayersAckList := make(map[int]int)
	for _,player:= range syncRoomPoolElement.Room.PlayerList {
		if player.Status == PLAYER_STATUS_OFFLINE{
			mylog.Error("player offline")
			continue
		}
		mynetWay.SendMsgByUid(player.Id,action,content)
		PlayersAckList[player.Id] = 0
	}
	syncRoomPoolElement.PlayersAckList = PlayersAckList
	sync.upSyncRoomPoolElementPlayersAckStatus(roomId,PLAYERS_ACK_STATUS_WAIT)
	//syncRoomPoolElement.PlayersAckStatus = PLAYERS_ACK_STATUS_WAIT

	logicFrameHistory := LogicFrameHistory{
		Action: action,
		Content: content,
	}
	//该局副本的所有玩家操作日志，用于断线重连-补放/重播
	syncRoomPoolElement.LogicFrameHistory = append(syncRoomPoolElement.LogicFrameHistory,logicFrameHistory)
}

func  (sync *Sync)playerResumeGame(requestPlayerResumeGame RequestPlayerResumeGame,wsConn *WsConn){
	str,_ := json.Marshal(requestPlayerResumeGame)
	mynetWay.SendMsgByUid(wsConn.PlayerId,"otherPlayerResumeGame",string(str))
}