package bak


/*
	createRoom
	createTeamRoom
	joinRoom
	joinTeamRoom
	leaveRoom
	dismissRoom
	changeRoom
	removeRoom
	changeCustomPlayerStatus
	getRoomById
	getRoomDetail


	startFrameSync
	stopFrameSync
	sendFrame
	requestFrame

*/
/*
//登陆
type RequestLogin struct {
	Token string `json:"token"`
}
//S端PING C端
type RequestPingRTT struct {
	AddTime 			int64	`json:"addTime"`
	ClientReceiveTime 	int64	`json:"clientReceiveTime"`
	ServerResponseTime	int64	`json:"serverResponseTime"`
}
//获取玩家状态信息
type RequestPlayerStatus struct {
	PlayerId	int `json:"playerId"`
}
//玩家掉线后，重连，游戏未结束，开始恢复
type RequestPlayerResumeGame struct {
	PlayerId	int `json:"playerId"`
	RoomId	 	string `json:"roomId"`
	sequenceNumber int 	`json:"sequenceNumber"`
}
//玩家报名进入匹配
type RequestPlayerMatchSign struct {
	PlayerId	int `json:"playerId"`
}
//玩家进入准备
type RequestPlayerReady struct {
	PlayerId	int `json:"playerId"`
	RoomId	 	string `json:"roomId"`
}
//获取一个房间的信息
type RequestGetRoom struct {
	PlayerId	int `json:"playerId"`
	RoomId	 	string `json:"roomId"`
}
//获取一个房间的所有历史记录
type RequestRoomHistory struct {
	PlayerId	int `json:"playerId"`
	RoomId	 	string `json:"roomId"`
	SequenceNumberStart int `json:"sequenceNumberStart"`
	SequenceNumberEnd int `json:"sequenceNumberEnd"`
}
//取消准备
type RequestCancelSign struct {
	PlayerId	int `json:"playerId"`
}
//客户端心跳
type RequestClientHeartbeat struct {
	Time	int64 `json:"time"`
}
//C端主动触发 一局游戏 结束
type RequestGameOver struct {
	PlayerId	int `json:"playerId"`
	RoomId	 	string `json:"roomId"`
	SequenceNumber int `json:"sequenceNumber"`
	Result 		string `json:"result"`
}




//================================================
type ResponseLoginRes struct {
	Code int				`json:"code"`
	ErrMsg string 			`json:"errMsg"`
	PlayerConnInfo	Player 	`json:"playerConnInfo"`
}

type ResponseGameOver struct {
	PlayerId	int `json:"playerId"`
	RoomId	 	string `json:"roomId"`
	SequenceNumber int `json:"sequenceNumber"`
	Result 		string `json:"result"`
}

type ResponsePlayerStatus struct {
	Id 			int		`json:"id"`
	Nickname	string	`json:"nickname"`
	Status 		int		`json:"status"`
	AddTime		int		`json:"addTime"`
	UpTime 		int		`json:"upTime"`
	RoomId 		string	`json:"roomId"`
}

//游戏开始后，第一次的初始化
type ResponseClientInitRoomData struct {
	RandSeek		int			`json:"randSeek"`
	RoomId			string		`json:"roomId"`
	SequenceNumber	int			`json:"sequenceNumber"`
	PlayerList		[]*Player	`json:"playerList"`
	Time 			int64 		`json:"time"`
	AddTime 		int 		`json:"addTime"`
	Status 			int 		`json:"status"`
}

type ResponseOtherPlayerOffline struct {
	PlayerId	int `json:"playerId"`
}

type ResponseRoomHistory struct {
	RoomId	 	string `json:"roomId"`
	SequenceNumber int `json:"sequenceNumber"`
	Command 	LogicFrameHistory
}

type ResponseKickOff struct {
	Time 			int64 		`json:"time"`
}

type ResponseStartBattle struct {
	SequenceNumber 			int64 		`json:"sequenceNumber"`
}

*/

