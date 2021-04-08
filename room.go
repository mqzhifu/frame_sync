package main

import (
	"strconv"
	"time"
	"zlib"
)

type Room struct {
	Id			string
	AddTime 	int
	Status 		int
	PlayerList	[]Player
}

type Player struct {
	Id int
}

func NewRoom()*Room{
	room := new(Room)
	room.Id = CreateRoomId()
	room.Status = ROOM_STATUS_WAIT
	room.AddTime = zlib.GetNowTimeSecondToInt()
	return room
}

func CreateRoomId()string{
	tt := time.Now().UnixNano() / 1e6
	string:=strconv.FormatInt(tt,10)
	return string
}

func(room *Room) AddPlayer(player Player){
	room.PlayerList = append(room.PlayerList,player)
}
