package main

import (
	"container/list"
	"strconv"
	"time"
	"zlib"
)

type Room struct {
	Id			string
	AddTime 	int
	Status 		int
	PlayerList	[]*Player
	Timeout 	int
	SequenceNumber		int
	PlayersAckList		map[int]int
	PlayersAckStatus	int
	RandSeek			int
	LogicFrameHistory 	[]LogicFrameHistory
	CommandQueue 		*list.List
	CloseChan 	chan int
}

func NewRoom()*Room{
	room := new(Room)
	room.Id = CreateRoomId()
	room.Status = ROOM_STATUS_INIT
	room.AddTime = zlib.GetNowTimeSecondToInt()
	room.SequenceNumber = 0
	room.PlayersAckList =  make(map[int]int)
	room.PlayersAckStatus = PLAYERS_ACK_STATUS_INIT
	room.RandSeek = zlib.GetRandIntNum(100)
	room.CommandQueue = list.New()
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

func (room *Room)upStatus(status int){
	mylog.Info("room upStatus ,old :",room.Status, " new :" ,status)
	room.Status = status
}
