package netway

import (
	"container/list"
	"frame_sync/myproto"
	"strconv"
	"sync"
	"time"
	"zlib"
)

type Room struct {
	Id					string                        	`json:"id"`					//房间ID
	AddTime 			int32                        	`json:"addTime"`			//创建时间
	//Timeout 			int32                           `json:"timeout"`			//超时时间
	StartTime 			int32                          	`json:"startTime"`			//开始游戏时间
	EndTime 			int32                           `json:"endTime"`			//游戏结束时间
	ReadyTimeout 		int32                           `json:"readyTimeout"`		//玩家准备超时时间
	Status 				int32                         	`json:"status"`				//状态
	PlayerList			[]*myproto.Player             	`json:"playerList"`			//玩家列表
	SequenceNumber		int                           	`json:"sequenceNumber"`		//当前帧频逻辑帧顺序号
	PlayersReadyList	map[int32]int32                 `json:"playersReadyList"`	//玩家准备列表
	RandSeek			int32                           `json:"randSeek"`			//随机数种子
	PlayersAckList		map[int32]int32               	`json:"playersAckList"`		//玩家确认列表
	PlayersAckStatus	int                             `json:"playersAckStatus"`	//玩家确认列表的状态
	PlayersAckListRWLock  	*sync.RWMutex						`json:"-"`					//玩家一帧内的确认操作，需要加锁
	PlayersReadyListRWLock  *sync.RWMutex						`json:"-"`
	//接收玩家操作指令-集合
	PlayersOperationQueue 		*list.List             	`json:"-"`//用于存储玩家一个逻辑帧内推送的：玩家操作指令
	CloseChan 			chan int                       	`json:"-"`//关闭信号管道
	ReadyCloseChan 		chan int                      	`json:"-"`//<玩家准备>协程的关闭信号管道
	WaitPlayerOfflineCloseChan	chan int				`json:"-"`//<一局游戏，某个玩家掉线，其它玩家等待它的时间>
	//本局游戏，历史记录，玩家的所有操作
	LogicFrameHistory 	[]*myproto.ResponseRoomHistory 	`json:"logicFrameHistory"`//玩家的历史所有记录
}

func NewRoom()*Room {
	room := new(Room)
	room.Id = CreateRoomId()
	room.Status = ROOM_STATUS_INIT
	room.AddTime = int32(zlib.GetNowTimeSecondToInt())
	room.StartTime = 0
	room.EndTime = 0
	room.ReadyTimeout = 0
	room.SequenceNumber = 0
	room.PlayersAckList =  make(map[int32]int32)
	room.PlayersAckListRWLock = &sync.RWMutex{}
	room.PlayersAckStatus = PLAYERS_ACK_STATUS_INIT
	room.RandSeek = int32(zlib.GetRandIntNum(100))
	room.PlayersOperationQueue = list.New()
	room.PlayersReadyList =  make(map[int32]int32)
	room.PlayersReadyListRWLock = &sync.RWMutex{}

	myMetrics.fastLog("total.RoomNum",2,0)

	return room
}

func CreateRoomId()string{
	tt := time.Now().UnixNano() / 1e6
	string:=strconv.FormatInt(tt,10)
	return string
}

func(room *Room) AddPlayer(player *myproto.Player){
	room.PlayerList = append(room.PlayerList,player)
}

func (room *Room)UpStatus(status int32){
	mylog.Info("room upStatus ,old :",room.Status, " new :" ,status)
	room.Status = status
}
