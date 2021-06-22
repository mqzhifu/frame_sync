package netway

import (
	"context"
	"errors"
	"frame_sync/myproto"
	"math/rand"
	"strconv"
	"sync"
	"time"
	"zlib"
)

type PlayerManager struct {
	Pool  				map[int32]*myproto.Player //玩家 状态池
	PoolRDLock 			*sync.RWMutex
	SidMapPid 			map[string]int32	//sessionId 映射 playerId
	SidMapPidRWLock 	*sync.RWMutex
	Store 				int32
	DefaultContentType 	int32
	DefaultProtocol		int32
	CloseChan 			chan int
}
type PlayerManagerOption struct{
	store int32
	ContentType int32
	Protocol int32
	Log *zlib.Log
}

func NewPlayerManager(option PlayerManagerOption)*PlayerManager {
	mylog.Info("NewPlayerManager instance")
	playerManager := new(PlayerManager)
	playerManager.Pool = make(map[int32]*myproto.Player)
	playerManager.SidMapPid = make(map[string]int32)
	playerManager.initPool()
	playerManager.Store = option.store
	playerManager.DefaultContentType = option.ContentType
	playerManager.DefaultProtocol = option.Protocol
	playerManager.PoolRDLock = &sync.RWMutex{}
	playerManager.CloseChan = make(chan int)
	playerManager.SidMapPidRWLock = &sync.RWMutex{}
	return playerManager
}

func (playerManager *PlayerManager)initPool(){
	if playerManager.Store == 1{

	}
}
func (playerManager *PlayerManager)getAll()map[int32]*myproto.Player{
	playerManager.PoolRDLock.RLock()
	defer playerManager.PoolRDLock.RUnlock()
	playerPool := make(map[int32]*myproto.Player)
	for k,player:= range  playerManager.Pool{
		playerPool[k] = player
	}

	return playerPool

}
func (playerManager *PlayerManager) Start(ctx context.Context){
	playerManager.checkOfflineTimeout(ctx)
}
func (playerManager *PlayerManager)Shutdown(){
	mylog.Alert("Shutdown PlayerManager")
	playerManager.CloseChan <- 1
}
//注意，会有：concurrent map iteration and map write
func (playerManager *PlayerManager)checkOfflineTimeout(ctx context.Context){
	defer func(ctx context.Context ) {
		if err := recover(); err != nil {
			myNetWay.RecoverGoRoutine(playerManager.checkOfflineTimeout,ctx,err)
		}
	}(ctx)

	mylog.Info("checkOfflineTimeout start:")
	for{
		select {
			case   <-playerManager.CloseChan:
				goto end
			default:
				if len(playerManager.Pool) <= 0{
					time.Sleep(time.Second * 10)
				}else{
					players := playerManager.getAll()
					for _,player:= range players{
						if player.Status != PLAYER_STATUS_OFFLINE{
							continue
						}

						now := zlib.GetNowTimeSecondToInt()
						timeout :=  int(player.AddTime )  + 3600
						if now > timeout{
							playerManager.delById(player.Id)
						}
					}
				}
		}
	}

end:
	mylog.Warning("checkOfflineTimeout close")
}

func  (playerManager *PlayerManager)addPlayer(id int32,firstMsg myproto.Msg)(existPlayer myproto.Player,err error){
	mylog.Info("addPlayerPool :",id)
	hasPlayer,empty  := playerManager.GetById(id)
	if !empty{
		mylog.Notice("this player id has exist pool",id)
		if hasPlayer.Status == PLAYER_STATUS_ONLINE {
			errMsg := "hasPlayer.Status = PLAYER_STATUS_ONLINE "
			mylog.Error(errMsg)
			err = errors.New(errMsg)
			return *hasPlayer,err
		}else{
			playerManager.upPlayerStatus(id, PLAYER_STATUS_ONLINE)
			return *hasPlayer,nil
		}
	}else{
		mylog.Info("new player add pool...")
		timeout := int32(zlib.GetNowTimeSecondToInt() + 60 * 60)
		player := myproto.Player{
			Id:       id,
			AddTime:  int32(zlib.GetNowTimeSecondToInt()),
			Nickname: "",
			Status:   PLAYER_STATUS_ONLINE,
			Timeout: timeout,
			SessionId: playerManager.createNewSessionId(),
			ContentType: int32(firstMsg.ContentType),
			ProtocolType: int32(firstMsg.ProtocolType),
		}
		playerManager.PoolRDLock.Lock()
		defer playerManager.PoolRDLock.Unlock()

		playerManager.Pool[id] = &player
		playerManager.SidMapPidRWLock.Lock()
		playerManager.SidMapPid[player.SessionId] = id
		playerManager.SidMapPidRWLock.Unlock()
		return player,nil
	}
	return existPlayer,nil
}
func  (playerManager *PlayerManager)createNewSessionId()string{
	nano := time.Now().UnixNano()
	rand.Seed(nano)
	rndNum := rand.Int63()
	sessionId := zlib.Md5( zlib.Md5(strconv.FormatInt(nano, 10))+ zlib.Md5(strconv.FormatInt(rndNum, 10)))
	return sessionId
}
func (playerManager *PlayerManager)GetPlayerCtrlInfoById(playerId int32)ProtocolCtrlInfo{
	var contentType  int32
	var protocolType int32
	if playerId == 0{
		contentType = playerManager.DefaultContentType
		protocolType = playerManager.DefaultProtocol
	}else{
		player ,empty := playerManager.GetById(playerId)
		//mylog.Debug("GetContentTypeById player",player)
		if empty{
			contentType = playerManager.DefaultContentType
			protocolType = playerManager.DefaultProtocol
		}else{
			contentType = player.ContentType
			protocolType = player.ProtocolType
		}
	}

	protocolCtrlInfo := ProtocolCtrlInfo{
		ContentType: contentType,
		ProtocolType: protocolType,
	}
	return protocolCtrlInfo
}
func  (playerManager *PlayerManager)GetById(playerId int32)(player *myproto.Player,empty bool){
	playerManager.PoolRDLock.RLock()
	defer playerManager.PoolRDLock.RUnlock()
	myPlayer ,ok := playerManager.Pool[playerId]
	if ok {
		return myPlayer ,false
	}else{
		return player,true
	}
}

func  (playerManager *PlayerManager)GetRoomIdByPlayerId(playerId int32)string{
	player , empty := playerManager.GetById(playerId)
	if empty{
		mylog.Error("GetRoomIdByPlayerId GetById is empty!!! ,pid:",playerId)
		return ""
	}
	return player.RoomId
}

func  (playerManager *PlayerManager)delById(playerId int32){
	playerManager.PoolRDLock.Lock()
	defer playerManager.PoolRDLock.Unlock()
	mylog.Warning("playerManager delById :",playerId)
	delete(playerManager.Pool,playerId)
	if playerManager.Store == 1{

	}
}

func   (playerManager *PlayerManager)upPlayerStatus(id int32,status int32){
	player ,_:= playerManager.GetById(id)

	mylog.Info("upPlayerStatus" , " old : ",player.Status," new:",status)


	playerManager.PoolRDLock.Lock()
	defer playerManager.PoolRDLock.Unlock()

	player.Status = status
	player.UpTime = int32(zlib.GetNowTimeSecondToInt())
}

func   (playerManager *PlayerManager)UpPlayerRoomId(playerId int32,roomId string){
	player := playerManager.Pool[playerId]

	mylog.Info("upPlayerRoomId" , " old : ",player.RoomId," new:",roomId)

	playerManager.PoolRDLock.Lock()
	defer playerManager.PoolRDLock.Unlock()

	player.RoomId = roomId
	player.UpTime = int32 (zlib.GetNowTimeSecondToInt())

	if playerManager.Store == 1{

	}

}
