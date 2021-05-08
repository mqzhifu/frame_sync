package main

import (
	"container/list"
	"strconv"
	"time"
	"zlib"
)

type Room struct {
	Id					string				`json:"id"`
	AddTime 			int32				`json:"addTime"`
	Timeout 			int32				`json:"timeout"`
	StartTime 			int32 				`json:"startTime"`
	EndTime 			int32				`json:"endTime"`
	ReadyTimeout 		int32				`json:"readyTimeout"`
	Status 				int32				`json:"status"`
	PlayerList			[]*Player			`json:"playerList"`
	SequenceNumber		int					`json:"sequenceNumber"`
	PlayersAckList		map[int32]int32		`json:"playersAckList"`
	PlayersAckStatus	int					`json:"playersAckStatus"`
	PlayersReadyList	map[int32]int32		`json:"playersReadyList"`
	RandSeek			int32				`json:"randSeek"`
	PlayersOperationQueue 		*list.List	`json:"-"`
	CloseChan 			chan int			`json:"-"`
	//ReadyCloseChan 		chan int			`json:"-"`
	LogicFrameHistory 	[]*ResponseRoomHistory	`json:"logicFrameHistory"`
}

func NewRoom()*Room{
	room := new(Room)
	room.Id = CreateRoomId()
	room.Status = ROOM_STATUS_INIT
	room.AddTime = int32(zlib.GetNowTimeSecondToInt())
	room.SequenceNumber = 0
	room.PlayersAckList =  make(map[int32]int32)
	room.PlayersAckStatus = PLAYERS_ACK_STATUS_INIT
	room.RandSeek = int32(zlib.GetRandIntNum(100))
	room.PlayersOperationQueue = list.New()
	room.PlayersReadyList =  make(map[int32]int32)
	//mynetWay.Option.Mylog.Info("addPoolElement")
	//_ ,exist := mySyncRoomPool[room.Id]
	//if exist{
	//	mynetWay.Option.Mylog.Error("mySyncRoomPool has exist : ",room.Id)
	//	return
	//}
	//syncRoomPoolElement := SyncRoomPoolElement{
	//	Status: SYNC_ELEMENT_STATUS_WAIT,
	//	Room: room,
	//}

	return room
}

func CreateRoomId()string{
	tt := time.Now().UnixNano() / 1e6
	string:=strconv.FormatInt(tt,10)
	return string
}

func(room *Room) AddPlayer(player *Player){
	room.PlayerList = append(room.PlayerList,player)
}

func (room *Room)upStatus(status int32){
	mylog.Info("room upStatus ,old :",room.Status, " new :" ,status)
	room.Status = status
}
