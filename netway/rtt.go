package netway

import (
	"frame_sync/myproto"
	"time"
	"zlib"
)
type RttCallbackCancel func()
type RttCallbackTimerTimeout func()
var rttMinTimeSecond = 1
//==================================
func  (netWay *NetWay)serverPingRtt(second time.Duration ,conn *Conn,times int ){
	//ping 一下，测试下RTT
	millisecond  := zlib.GetNowMillisecond()
	//超时时间 = rtt最小时间
	rttTimeout := millisecond + int64( int(second) * 1000);
	responseServerPing := myproto.ResponseServerPing{
		AddTime:millisecond,
		RttTimeout: rttTimeout,
		RttTimes: int32(times),
	}
	//正常收到了C端回调，且一切正常
	RttCallbackCancelFunc := func (){
		mylog.Warning("RttCallbackCancelFunc...")
	}
	//触发了超时回调
	RttCallbackTimerTimeoutFunc := func(){
		mylog.Error("RttCallbackTimerTimeoutFunc")
		if times >= 2{
			netWay.CloseOneConn(conn,CLOSE_SOURCE_RTT_TIMER_OUT)
			return
		}
	}
	//创建一个定时器
	getOnetimerCtrl(second,conn.RTTCancelChan,RttCallbackCancelFunc,RttCallbackTimerTimeoutFunc)
	netWay.SendMsgCompressByUid(conn.PlayerId,"serverPing",&responseServerPing)
}

func(netWay *NetWay) ClientPong(requestClientPong myproto.RequestClientPong,conn *Conn){
	RTT := requestClientPong.ClientReceiveTime -  requestClientPong.AddTime
	now := zlib.GetNowMillisecond()
	mylog.Info("client RTT:",RTT," ms")
	if now >= requestClientPong.RttTimeout{//证明已超过规定的返回时间了
		//既然超时了，应该再放大系数，再来一次，但timer callback 已经做了处理，这里不用再处理了
		//还有一种可能：第一次发的RTT测试，迟迟得不到响应，反而第二次的RTT测试先返回了
		//整个机制已经更新到第二次了，这个时候第一次的消息终于返回了，但已然没了意义，直接丢弃
		mylog.Error("ClientPong rtt > timeout, need drop this msg",now ,requestClientPong.RttTimeout)
		return
	}
	//未超时的话，此RTT时间才是可用的时间
	conn.RTT = RTT
	//既然RTT未超时且无异常，即取消到定时器
	conn.RTTCancelChan <- 1
	if RTT > int64(rttMinTimeSecond * 1000){
		//虽然没有触发超时机制，但是大于1秒，囚徒 模式的同步，游戏已毫无体验感了
		mylog.Error("RTT >= 1000ms")
		//如果重试2次以上的RTT ，还是超时，直接放弃了...关闭FD
		if requestClientPong.RttTimes >= 2{
			netWay.CloseOneConn(conn,CLOSE_SOURCE_RTT_TIMEOUT)
			return
		}
		timeSecond := rttMinTimeSecond * 2 //放大两倍再试一次
		netWay.serverPingRtt(time.Duration(timeSecond),conn,2)
		return
	}
	//小于500ms,证明网络还可以，没必要太频繁RTT测试 ，会造成大批量的PING/PONG，占用带宽、影响正常消息发送
	if RTT < 500 {
		mylog.Info("rtt sleep 500ms")
		//睡眠500毫秒
		time.Sleep(500 * time.Millisecond)
	}
	//再开启一个新的TIMER
	netWay.serverPingRtt(time.Duration(rttMinTimeSecond),conn,1)
}

func(netWay *NetWay)clientPing(pingRTT myproto.RequestClientPing,conn *Conn){
	responseServerPong := myproto.ResponseServerPong{
		AddTime: pingRTT.AddTime,
		ClientReceiveTime :pingRTT.ClientReceiveTime,
		ServerResponseTime:zlib.GetNowMillisecond(),
		RttTimes: pingRTT.RttTimes,
		RttTimeout: pingRTT.RttTimeout,
	}
	netWay.SendMsgCompressByUid(conn.PlayerId,"serverPong",&responseServerPong)
}

func getOnetimerCtrl(timeoutSecond time.Duration,cancel chan int,cancelCallback RttCallbackCancel ,timeoutCallback RttCallbackTimerTimeout){
	mylog.Debug("create a new timer : ",timeoutSecond * 1000 ,"ms" )
	timer := time.NewTicker(time.Second *  timeoutSecond)
	go func(timer *time.Ticker){
		isBreak := 0
		for{
			select {
			case <- cancel:
				zlib.MyPrint("select chan read : cancelTime")
				timer.Stop()
				cancelCallback()
				isBreak = 1
			case <- timer.C:
				zlib.MyPrint("select chan read : timeout")
				timeoutCallback()
				isBreak = 1
			}

			if isBreak == 1{
				break
			}
		}
		zlib.MyPrint("OnetimerCtrl end.")
	}(timer)

}