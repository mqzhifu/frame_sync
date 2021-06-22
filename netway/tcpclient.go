package netway

import (
	"frame_sync/myproto"
	"github.com/golang/protobuf/proto"
	"net"
	"strconv"
	"zlib"
)
func StartTcpClient(netWayOption NetWayOption,log *zlib.Log){
	ipPort := netWayOption.ListenIp + ":" +netWayOption.TcpPort
	//ipPort := netWayOption.ListenIp + ":11111"
	mylog := log
	var buf [512]byte
	// 绑定
	tcpAddr, err := net.ResolveTCPAddr("tcp", ipPort)
	if err !=nil{
		mylog.Error("net.Listen tcp err:",err.Error())
		return
	}
	// 连接
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err !=nil{
		mylog.Error("DialTCP err:",err.Error())
		return
	}
	//rAddr := conn.RemoteAddr()
	requestLogin := myproto.RequestLogin{
		Token: "aaaa",
	}
	binary ,_:= proto.Marshal(&requestLogin)
	strId := strconv.Itoa(1000)
	//合并 协议号 + 消息内容体
	content := zlib.BytesCombine([]byte(strId),binary)
	n, err := conn.Write(content)
	if err != nil {
		zlib.ExitPrint("Write err:",err.Error(),n)
	}

	n, err = conn.Read(buf[0:])
	if n == 0{
		zlib.ExitPrint("Read empty ",err,n)
	}

	if err != nil {
		zlib.ExitPrint("Read err:",err.Error(),n)
	}

	//for {
		// 发送
		//n, err := conn.Write([]byte("Hello server"))
		//if err !=nil{
		//	mylog.Error("conn.Write err:",err.Error())
		//	return
		//}
		//n, err = conn.Read(buf[0:])
		//if err !=nil{
		//	mylog.Error("conn.Read err:",err.Error())
		//	return
		//}
		//fmt.Println("Reply form server", rAddr.String(), string(buf[0:n]))
		//time.Sleep(time.Second * 2)
	//}

	//for{
	//
	//}
	//conn.Close()
	//os.Exit(0)
}
