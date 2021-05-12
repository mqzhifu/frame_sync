package main
//
//import (
//	"frame_sync/netway"
//	"github.com/golang/protobuf/proto"
//	"zlib"
//)
//
////http://code.google.com/p/go.net/websocket
////http://code.google.com/p/go.net/websocket
////http://golang.org/x/net/websocket
////github.com/gorilla/websocket
////http://github.com/golang/net
//
//type MyTokenTest struct {
//	AppId 	int
//	Uid 	int
//	ATime	int
//	Expire  int
//	Token 	string
//}
//func (netWay *netway.NetWay)testCreateJwtToken()[]MyTokenTest {
//
//	var tokenList   []MyTokenTest
//	appId := 2
//	for i:=1000;i<1010;i++{
//		addTime := zlib.GetNowTimeSecondToInt()
//		Expire := zlib.GetNowTimeSecondToInt() +  24 * 60 * 60
//		jwtDataPayload := zlib.JwtDataPayload{
//			AppId: int32(appId),
//			Uid : int32(i),
//			ATime: int32(addTime),
//			Expire:int32(Expire),
//		}
//
//		myTokenTest := MyTokenTest{
//			AppId: appId,
//			Uid :i,
//			ATime: addTime,
//			Expire:Expire,
//		}
//
//		jwtToken := zlib.CreateJwtToken(netWay.Option.LoginAuthSecretKey,jwtDataPayload)
//		myTokenTest.Token = jwtToken
//
//		tokenList = append(tokenList,myTokenTest)
//	}
//
//
//	return tokenList
//}
//
//func testProtobuf2(content []byte , cls interface{}){
//	//aaa := cls.(*RequestLogin)
//
//	//name := "main.RequestLogin"
//	//pt := proto.MessageType(name)
//	//aaa := reflect.New(pt.Elem()).Interface().(proto.Message)
//
//
//	aaa := cls.(proto.Message)
//
//	//x2 := reflect.New(pt.Elem()).Interface().(proto.Message)
//	//var xa2 = &any.Any{TypeUrl:"./"+name,Value: dst}
//
//
//	proto.Unmarshal(content,aaa)
//	//zlib.MyPrint(aaa.Token)
//	//cls = aaa
//}
//
//func testProtobuf(){
//	//aaa := ResponseStartBattle{SequenceNumberStart: 0}
//	//bbb,err:=json.Marshal(aaa)
//	//zlib.ExitPrint(string(bbb),err)
//
//
//	//requestLogin := RequestLogin{
//	//	Token: "aaaa",
//	//}
//	//data, err := proto.Marshal(&requestLogin)
//	//zlib.MyPrint(data,err)
//	//if err != nil {
//	//	log.Fatal("marshaling error: ", err)
//	//}
//
//	//rr := RequestLogin{}
//	//zlib.MyPrint(rr)
//	//
//	//aa,err := proto.Marshal(&rr)
//	//zlib.ExitPrint(aa,err)
//
//	//proto.Unmarshal(data,&rr)
//	//testProtobuf2(data,&rr)
//	//zlib.ExitPrint(rr)
//
//	//requestLogin := main.RequestLoginNew{
//	//	Token: "aaaa",
//	//}
//	//data, err := proto.Marshal(requestLogin)
//	//if err != nil {
//	//	log.Fatal("marshaling error: ", err)
//	//}
//	//zlib.ExitPrint(data)
//}
//func testVar(){
//
//}
//func test(){
//	//testProtobuf()
//
//	testVar()
//
//	//queue := list.New()
//	//queue.PushBack("aaaa")
//	//element := queue.Front()
//	//info := element.Value.(string)
//	//zlib.ExitPrint(info)
//
//	//routerMap := make(map[string]interface{})
//	//routerMap["ping"] = Ping{a:1,b:"im,bbb"}
//	//
//	////rto := reflect.TypeOf(routerMap["ping"])
//	//rvo := reflect.ValueOf(routerMap["ping"])
//	//rvoKind := rvo.Kind()
//	//if rvoKind != reflect.Struct{
//	//	zlib.ExitPrint("not struct")
//	//}
//	//
//	//var newStruct interface{}
//	//structType:= rvo.Type()
//	//switch structType.String() {
//	//	case "main.Ping":
//	//		zlib.MyPrint("this struce type:","main.Ping")
//	//		newStruct := newStruct.(Ping)
//	//}
//	//test2(newStruct)
//	//zlib.MyPrint(structType)
//	//
//	//for k := 0; k < rto.NumField(); k++ {
//	//	aaa:= rvo.Type().Field(k)
//	//	zlib.MyPrint(aaa)
//	//}
//	//zlib.ExitPrint(2333)
//}
