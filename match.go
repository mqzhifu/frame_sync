package main

import (
	"context"
	"errors"
	"time"
	"zlib"
)

type Match struct {
	Option MatchOption
}

type MatchOption struct {
	RoomPeople	int
}

type PlayerSign struct {
	AddTime 	int
	PlayerId	int
}

var signPlayerPool []PlayerSign
func NewMatch(matchOption MatchOption)*Match{
	match  := new(Match)
	match.Option = matchOption
	return match
}

func (match *Match)getOneSignPlayerById(playerId int ) (playerSign PlayerSign,empty bool){
	for _,v := range signPlayerPool{
		if v.PlayerId == playerId {
			return v,false
		}
	}
	return playerSign,true
}

func (match *Match) addOnePlayer(playerId int)error{
	_,empty := match.getOneSignPlayerById(playerId)
	if !empty{
		return errors.New("match sign addOnePlayer : player has exist")
	}
	newPlayerSign := PlayerSign{PlayerId: playerId,AddTime: zlib.GetNowTimeSecondToInt()}
	signPlayerPool = append(signPlayerPool,newPlayerSign)
	return nil
}

func (match *Match) delOnePlayer(playerId int){
	mylog.Info("cancel : delOnePlayer ",playerId)
	for k,v:=range signPlayerPool{
		if v.PlayerId == playerId{
			if len(signPlayerPool ) == 1{
				signPlayerPool = []PlayerSign{}
			}else{
				signPlayerPool = append(signPlayerPool[:k], signPlayerPool[k+1:]...)
			}
			return
		}
	}
	mylog.Error("no match playerId",playerId)
}

func (match *Match) matchingPlayerCreateRoom(ctx context.Context){
	mylog.Info("matchingPlayerCreateRoom:start")
	for{
		select {
		case   <-ctx.Done():
			//netWay.Option.Mylog.Warning("matchingPlayerCreateRoom close")
			goto end
		default:
			//zlib.MyPrint(len(signPlayerPool))
			//mylog.Info("matching:",len(signPlayerPool),match.Option.RoomPeople)
			if len(signPlayerPool) >= match.Option.RoomPeople{
				newRoom := NewRoom()
				for i:=0;i < len(signPlayerPool);i++{
					player,empty := mynetWay.Players.getById(signPlayerPool[i].PlayerId)
					if empty{
						mylog.Error("match Players.getById empty , ",signPlayerPool[i].PlayerId)
					}
					player.RoomId = newRoom.Id
					newRoom.AddPlayer(player)
				}
				//删除上面匹配成功的玩家
				signPlayerPool = append(signPlayerPool[match.Option.RoomPeople:])
				mylog.Info("create a room :",newRoom)
				//将该房间添加到容器中
				mySync.addPoolElement(newRoom)
				mySync.start(newRoom.Id)
			}
			time.Sleep(time.Second * 1)
			//mySleepSecond(1,"matching player")
		}
	}
end:
	mylog.Info("matchingPlayerCreateRoom close")
}

