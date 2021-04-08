package main

import (
	"encoding/json"
	"strconv"
	"zlib"
)

type Sync struct {

}

type SyncRoomPoolElement struct {
	Status int
	Room 	*Room
	LogicFrameSequenceNumber int
	LogicFramePlayerAck map[int]int
	AddTime 	int
	RandSeek	int
}

type LogicFrame struct {
	RoomId 			string
	SequenceNumber 	int
	Commands 		[]Command
	RandSeek		int
}

type Command struct {
	Action 		string
	Value 		string
	PlayerId 	int
}

//type PlayerAckMsg struct {
//	SequenceNumber int
//	Commands 		string
//}


var mySyncRoomPool map[string]*SyncRoomPoolElement
func NewSync()*Sync{
	mySyncRoomPool = make(map[string]*SyncRoomPoolElement)
	sync := new(Sync)
	return sync
}

func (sync Sync) addPoolElement(room 	*Room){
	mynetWay.Option.Mylog.Info("addPoolElement")
	_ ,exist := mySyncRoomPool[room.Id]
	if exist{
		mynetWay.Option.Mylog.Error("mySyncRoomPool has exist : ",room.Id)
		return
	}
	syncRoomPoolElement := SyncRoomPoolElement{
		Status: SYNC_ELEMENT_STATUS_WAIT,
		Room: room,
		LogicFrameSequenceNumber: 1,
		LogicFramePlayerAck: make(map[int]int),
		AddTime: zlib.GetNowTimeSecondToInt(),
		RandSeek: zlib.GetRandIntNum(100),

	}
	mySyncRoomPool[room.Id] = &syncRoomPoolElement
}

func (sync Sync)start(roomId string){
	mynetWay.Option.Mylog.Warning("start a new game:")

	syncRoomPoolElement,_ := sync.getPoolElementById(roomId)
	syncRoomPoolElement.Status = SYNC_ELEMENT_STATUS_EXECING
	syncRoomPoolElement.Room.Status = ROOM_STATUS_WAIT_EXECING
	//sync.upRoomPoolElementStatus(roomId   ,ROOM_STATUS_WAIT_EXECING )

	LogicFramePlayerAck := make(map[int]int)
	//S端，每一逻辑帖，建立一个集合，保存广播的消息，玩家返回的确认ACK
	commands := []Command{}
	for _,player:= range syncRoomPoolElement.Room.PlayerList{
		location := strconv.Itoa(zlib.GetRandIntNum(4)) + "," + strconv.Itoa(zlib.GetRandIntNum(4))
		command := Command{
			Action: "move",
			Value: location,
			PlayerId: player.Id,
		}
		commands = append(commands,command)
		LogicFramePlayerAck[player.Id] = 0
		//sync.upLogicFramePlayerAck()
	}
	syncRoomPoolElement.LogicFramePlayerAck = LogicFramePlayerAck

	for _,player:= range syncRoomPoolElement.Room.PlayerList{

		logicFrameMsg := LogicFrame{
			RoomId: roomId,
			SequenceNumber :syncRoomPoolElement.LogicFrameSequenceNumber,
			Commands 		:commands,
			RandSeek: syncRoomPoolElement.RandSeek,
		}
		logicFrameMsgJson ,_ := json.Marshal(logicFrameMsg)
		msg := Msg{
			Action: "start_init",
			Content: string(logicFrameMsgJson),
		}
		mynetWay.SendMsgByUid(player.Id,msg)
	}







}
//有一个房间内，搜索一个用户
func (sync Sync)getPlayerByIdInRoom(playerId int ,room *Room,)(myplayer Player,empty bool){
	for _,player:= range room.PlayerList{
		if player.Id == playerId{
			return player,false
		}
	}
	return myplayer,true
}
func (sync Sync) getPoolElementById(roomId string)(SyncRoomPoolElement *SyncRoomPoolElement,empty bool){
	v,exist := mySyncRoomPool[roomId]
	if !exist{
		mynetWay.Option.Mylog.Error("getPoolElementById is empty,",roomId)
		return SyncRoomPoolElement,true
	}
	return v,false
}

func  (sync Sync)receiveCommand(content string,wsConn *WsConn){
	logicFrame := LogicFrame{}
	err := json.Unmarshal([]byte(content),&logicFrame)
	if err != nil{
		mynetWay.Option.Mylog.Error("receiveCommand Unmarshal ",err.Error())
		return
	}

	element,empty := sync.getPoolElementById(logicFrame.RoomId)
	if empty{
		mynetWay.Option.Mylog.Error("getPoolElementById is empty",logicFrame.RoomId)
	}
	element.LogicFramePlayerAck = make(map[int]int)
	//复用玩家发来的logicFrame结构内容即可
	logicFrame.SequenceNumber = element.LogicFrameSequenceNumber

	for _,player:= range element.Room.PlayerList{
		logicFrameMsgJson ,_ := json.Marshal(logicFrame)
		msg := Msg{
			Action: "logicFrame",
			Content: string(logicFrameMsgJson),
		}
		element.LogicFramePlayerAck[player.Id] = 0
		mynetWay.SendMsgByUid(player.Id,msg)
	}

	//sync.boardCastFrameForElement(roomId,msg)
}

func (sync Sync) playerLogicFrameAck(content string,wsConn *WsConn ){
	logicFrame := LogicFrame{}
	err := json.Unmarshal([]byte(content),&logicFrame)
	if err != nil{
		mynetWay.Option.Mylog.Error("receiveCommand Unmarshal ",err.Error())
		return
	}

	syncRoomPoolElement,empty := sync.getPoolElementById(logicFrame.RoomId)
	if empty{
		mynetWay.Option.Mylog.Error("roomId is wrong",logicFrame.RoomId)
		return
	}
	if syncRoomPoolElement.LogicFrameSequenceNumber != logicFrame.SequenceNumber{
		mynetWay.Option.Mylog.Error("SequenceNumber wrong",logicFrame.SequenceNumber)
		return
	}

	player ,empty := sync.getPlayerByIdInRoom(wsConn.PlayerId,syncRoomPoolElement.Room)
	if empty{
		mynetWay.Option.Mylog.Error("playerId wrong,not found in this room.",wsConn.PlayerId)
		return
	}

	syncRoomPoolElement.LogicFramePlayerAck[player.Id] = 1
	ack := 0
	for _,v := range syncRoomPoolElement.LogicFramePlayerAck{
		if v == 1{
			ack++
		}
	}
	mynetWay.Option.Mylog.Info("this room LogicFrameSequenceNumber(",syncRoomPoolElement.LogicFrameSequenceNumber,") ack total:",ack)

	if ack == len(syncRoomPoolElement.LogicFramePlayerAck){
		mynetWay.Option.Mylog.Info("one logic frame all ack.")
		//该逻辑帧已全部确认
		syncRoomPoolElement.LogicFrameSequenceNumber++
		syncRoomPoolElement.LogicFramePlayerAck = make(map[int]int)
	}

	mynetWay.Option.Mylog.Info("playerLogicFrameAck~ finish.")
	return

}

func  (sync Sync)end(roomId string){
	syncRoomPoolElement ,_:= sync.getPoolElementById(roomId)
	syncRoomPoolElement.Status = SYNC_ELEMENT_STATUS_END
	syncRoomPoolElement.Room.Status = ROOM_STATUS_WAIT_END
}
func  (sync Sync)boardCastFrameForElement(roomId int64,msg Msg){
	//syncRoomPoolElement ,_:= sync.getPoolElementById(roomId)
	//LogicFramePlayerAck := make(map[int]int)
	//logicFrameMsg := LogicFrame{
	//	SequenceNumber :syncRoomPoolElement.LogicFrameSequenceNumber,
	//	Commands 		:msg.Content,
	//}
	//logicFrameMsgJson ,_ := json.Marshal(logicFrameMsg)
	////S端，每一逻辑帖，建立一个集合，保存广播的消息，玩家返回的确认ACK
	//for _,player:= range syncRoomPoolElement.Room.PlayerList{
	//	LogicFramePlayerAck[player.Uid] = 0
	//	mynetWay.SendMsgByUid(player.Uid,string(logicFrameMsgJson))
	//}
	//syncRoomPoolElement.LogicFramePlayerAck = LogicFramePlayerAck
}