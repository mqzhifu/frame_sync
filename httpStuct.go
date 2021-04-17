package main

type RequestLogin struct {
	Token string `json:"token"`
}

type ResponseGameOver struct {
	Result string `json:"result"`
}

type PingRTT struct {
	AddTime 			int64
	ClientReceiveTime 	int64
	ServerResponseTime	int64
}