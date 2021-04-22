package main

type RequestLogin struct {
	Token string `json:"token"`
}

type RequestPlayerResumeGame struct {
	PlayerId	int `json:"playerId"`
	RoomId	 	string `json:"roomId"`
	sequenceNumber int 	`json:"sequenceNumber"`
}

type RequestPlayerStatus struct {
	PlayerId	int `json:"playerId"`
}

type RequestPlayerReady struct {
	PlayerId	int `json:"playerId"`
}

type RequestCancelSign struct {
	PlayerId	int `json:"playerId"`
}

type RequestClientHeartbeat struct {
	Time	int64 `json:"time"`
}

type RequestPingRTT struct {
	AddTime 			int64	`json:"addTime"`
	ClientReceiveTime 	int64	`json:"clientReceiveTime"`
	ServerResponseTime	int64	`json:"serverResponseTime"`
}

type RequestGameOver struct {
	PlayerId	int `json:"playerId"`
	RoomId	 	string `json:"roomId"`
	SequenceNumber int `json:"sequenceNumber"`
	Result 		string `json:"result"`
}

//================================================
type ResponseLoginRes struct {
	Code int		`json:"code"`
	Content string `json:"content"`
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
type ResponseClientInitData struct {
	RandSeek		int			`json:"randSeek"`
	RoomId			string		`json:"roomId"`
	SequenceNumber	int			`json:"sequenceNumber"`
	PlayerList		[]*Player	`json:"playerList"`
	Time 			int64 		`json:"time"`
}

type ResponseOtherPlayerOffline struct {
	PlayerId	int `json:"playerId"`
}



