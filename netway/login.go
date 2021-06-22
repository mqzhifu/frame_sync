package netway

import (
	"frame_sync/myproto"
	"zlib"
)

func  (netWay *NetWay)loginPre(conn *Conn)(jwt zlib.JwtData,err error,firstMsg myproto.Msg){
	//这里有个问题，连接成功后，C端立刻就得发消息，不然就异常~bug
	var loginRes myproto.ResponseLoginRes

	content,err := conn.Read()
	if err != nil{
		loginRes = myproto.ResponseLoginRes{
			Code : 500,
			ErrMsg:err.Error(),
		}
		netWay.SendMsgCompressByUid(conn.PlayerId,"loginRes",loginRes)
		netWay.CloseOneConn(conn, CLOSE_SOURCE_FD_READ_EMPTY)
		return
	}
	msg,err := myProtocolManager.parserContentProtocol(content)
	if err != nil{
		netWay.CloseOneConn(conn, CLOSE_SOURCE_FD_PARSE_CONTENT)
		return
	}
	if msg.Action != "login"{
		netWay.CloseOneConn(conn, CLOSE_SOURCE_FIRST_NO_LOGIN)
		mylog.Error("first msg must login api!!!")
		return
	}
	//开始：登陆/验证 过程
	jwtDataInterface,err := netWay.Router(msg,conn)
	jwt = jwtDataInterface.(zlib.JwtData)
	netWay.Option.Mylog.Debug("login rs :",jwt,err)
	if err != nil{
		netWay.Option.Mylog.Error(err.Error())
		loginRes = myproto.ResponseLoginRes{
			Code : 500,
			ErrMsg:err.Error(),
		}
		//NewWsConn.SendMsg(loginRes)
		netWay.SendMsgCompressByUid(conn.PlayerId,"loginRes",&loginRes)
		netWay.CloseOneConn(conn, CLOSE_SOURCE_AUTH_FAILED)
		return jwt ,err,firstMsg
	}
	netWay.Option.Mylog.Info("login jwt auth ok~~")
	return jwt,nil,msg
}

func(netWay *NetWay)login(requestLogin myproto.RequestLogin,conn *Conn)(JwtData zlib.JwtData,err error){
	token := ""
	if netWay.Option.LoginAuthType == "jwt"{
		token = requestLogin.Token
		jwtData,err := zlib.ParseJwtToken(netWay.Option.LoginAuthSecretKey,token)
		return jwtData,err
	}else{
		netWay.Option.Mylog.Error("LoginAuthType err")
	}

	return JwtData,err
}
