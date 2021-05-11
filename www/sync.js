var self = null;
function ws (playerId,token,host,uri,matchGroupPeople,tableMax,DomIdObj,offLineWaitTime,actionMap,FPS){
    var self = this;

    this.status = "init";//1初始化 2等待准备 3运行中  4结束
    this.wsObj = null;//js内置ws 对象

    this.hostUri = host +uri;//ws 连接地址
    this.playerId = playerId;//玩家ID
    this.matchGroupPeople = matchGroupPeople;//一个副本的人数
    this.heartbeatLoopFunc = null;//心跳回调函数
    this.offLineWaitTime = offLineWaitTime;//lockStep 玩家掉线后，其它玩家等待最长时间
    //以上都是S端返回的一些配置值

    this.token = token;//玩家的凭证
    this.tableMax = tableMax;//地址的表格大小
    this.otherPlayerOffline = 0;//其它玩家调线
    this.pushLogicFrameLoopFunc = null;//定时循环 - 推送玩家操作函数
    this.playerOperationsQueue = [];//一个帧时间内，收集玩家的操作指令 队列
    this.closeFlag = 0;//关闭标识，0正常1手动关闭2后端关闭

    this.tableId = "";
    this.domIdObj = DomIdObj ;
    this.playerLocation = new Object();//每个玩家的位置信息
    this.operationsInc = 0;//玩家操作指令自增ID
    this.logicFrameLoopTimeMs = 0;//毫秒，多少时间内收集一次玩家操作，推送到S端
    this.FPS = FPS;//每秒多少逻辑帧
    this.playerCommandPushLock = 0;
    //下面是帧同步初始化信息，是由S端供给
    this.roomId = "";
    this.actionMap = actionMap;
    this.sequenceNumber = 0;
    this.randSeek = 0;
    this.communicationContentType = "protobuf";
    // this.communicationContentType = "json";
    //入口函数，必须得先建立连接后，都有后续的所有操作
    this.create  = function(){
        console.log("this status :",self.status);
        if (self.status != "init" && self.status != "close"){
            return alert(" status !=  init or close");
        }
        self.closeFlag = 0;
        self.logicFrameLoopTimeMs = parseInt( 1000 / this.FPS);
        console.log("create new WebSocket"+self.hostUri)
        self.wsObj = new WebSocket(self.hostUri);

        self.wsObj.onclose = function(ev){
            self.onclose(ev)
        };
        self.wsObj.onmessage = function(ev){  //获取后端响应

            self.onmessage(ev)
        };
        self.wsObj.onopen = function(){
            self.wsOpen();
        };
        self.wsObj.onerror = function(ev){
            // self.wsError(ev.);
            for(let key  in ev){
                console.log(key + '---' + ev[key])
            }
        };
    };
    //连接成功后，会执行此函数
    this.wsOpen = function(){
        console.log("onOpen : ws link success  ")
        this.upStatus("wsLInkSuccess");

        var requestLoginObj = new proto.main.RequestLogin();
        requestLoginObj.setToken(self.token) ;
        this.sendNewMsg("login",requestLoginObj);
    };
    this.upStatus = function(status){
        this.status = status;
    };
    //两个玩家，位移碰撞了，触发了游戏结束
    this.gameOverAndClear = function(){
        this.upStatus("end")

        var requestGameOver = new proto.main.RequestGameOver()
        requestGameOver.setRoomId(self.roomId);
        requestGameOver.setSequenceNumber(self.sequenceNumber);
        requestGameOver.setResult("ccccccWin");
        this.sendNewMsg("gameOver", requestGameOver);

        window.clearInterval(self.pushLogicFrameLoopFunc);
        self.upOptBnt("游戏结束1",1)

        return alert("完犊子了，撞车了...这游戏得结束了....");
    };
    this.getActionId = function (action,category){
        // return alert(self.actionMap);
        var data = self.actionMap[category];
        // console.log(data);
        for(let key  in data){
            // console.log(data[key].Action);
            if (data[key].Action == action){
                return data[key].Id;
            }
        }
        alert(action + ": no match");
        return "";
    };
    this.getActionName = function (actionId,category){
        var data = self.actionMap[category];
        // alert(data[actionId]);
        return data[actionId].Action;
    };
    // this.sendMsgJson =  function ( action,content,jsonEncode  ){
    //     // if (jsonEncode == 1){
    //         content = JSON.stringify(content)
    //     // }
    //     var msg = {"action":action,"content":content};
    //     // var jsonStr =  JSON.stringify(msg);
    //     // var tmpcontent = content
    //     self.wsObj.send(content);
    //     return 111;
    // }
    this.sendNewMsg =  function ( action,contentObj  ){
        console.log( " sendMsg:" + self.descPre ,contentObj.toObject());
        var id = self.getActionId(action,"client");
        var content = null;
        if (self.communicationContentType == "json"){
            content = contentObj.toObject();
            content = JSON.stringify(content);
            if(action == "playerOperations"){
                console.log(content);
                content = content.replace("operationsList","operations");
                console.log(content);
            }
            content = id + content;
            self.wsObj.send(content);
        }else if (self.communicationContentType == "protobuf"){
            content = contentObj.serializeBinary();
            // console.log(content,content.length);
            id = id + "";
            var idBinary = stringToUint8Array(id);
            var mergedArray = new Uint8Array(idBinary.length + content.length);
            mergedArray.set(idBinary);
            mergedArray.set(content, idBinary.length);

            self.wsObj.send(mergedArray);
        }
    };
    this.upOptBnt = function(content,clearClick){
        $("#"+self.domIdObj.optBntId).html(content);
        if(clearClick == 1){
            $("#"+self.domIdObj.optBntId).unbind("click");
        }
    };
    this.closeFD = function (){//主动关闭
        console.log("closeFD");
        // window.clearInterval(self.heartbeatLoopFunc);
        // window.clearInterval(self.pushLogicFrameLoopFunc);
        self.myClose = 1;
        self.wsObj.close();
    };
    this.onclose = function(ev){//接收到服务端关闭
        alert("receive server close:" +ev.code);
        window.clearInterval(self.pushLogicFrameLoopFunc);
        self.upStatus("close")
        // window.clearInterval(self.heartbeatLoopFunc);
        if (self.myClose == 1){
            var reConnBntId = "reconn_"+self.playerId;
            var msg = "重连接";
            self.upOptBntHref(reConnBntId,msg,self.create)
            // self.upOptBnt("C端主动关闭WS，<a href='javascript:void(0);' id='"+reConnBntId+"'>重连接</a>",1)
            // $("#"+reConnBntId).click(self.create);
        }else{
            self.closeFlag = 2;
            self.upOptBnt("服务端关闭，游戏结束，连接断开",1)
        }
    };

    this.wsError = function(ev){
        console.log("error:"+ev);
        alert(ev);
    };
    //心跳执行函数
    this.heartbeat = function(){
        var now = Date.now();
        var requestClientHeartbeat = new proto.main.RequestClientHeartbeat();
        requestClientHeartbeat.setTime(now);
        this.sendNewMsg("clientHeartbeat",requestClientHeartbeat);
    };
    //接收S端WS消息
    this.onmessage = function(ev){
        var pre = self.descPre;
        console.log("onmessage:"+ pre + " " +ev.data);

        var actionId = "";
        var content = "";
        var action = "";

        if (self.communicationContentType == 'protobuf'){
            var reader = new FileReader();
            reader.readAsArrayBuffer(ev.data);
            reader.onloadend = function(e) {
                // console.info(reader.result);
                var buf = new Uint8Array(reader.result);
                // console.info(buf);
                var bbb = new ArrayBuffer((buf.length - 4 ) );
                var ccc = new Uint8Array(bbb);
                var j = 0;
                for (var i = 0; i < buf.length; i++) {
                    if (i < 4){
                        actionId += String.fromCharCode(buf[i]);
                    }else{
                        ccc[j] = buf[i];
                        j++;
                    }
                }
                // console.log(ccc);
                content = ccc;
                action = self.getActionName(actionId,"server")
                var actionLow = action.substring(0, 1).toUpperCase() + action.substring(1)
                var className =  "proto.main.Response" + actionLow;
                var responseProtoClass = eval(className);
                content =  responseProtoClass.deserializeBinary(content).toObject();
                // var responseLoginRes = proto.main.ResponseLoginRes.deserializeBinary(content).toObject();
                console.log(pre +" actionId:"+actionId , " content:",content , " actionName:",action, " protobufClassName:",className);
                self.router(action,content);
                // var rs = proto.main.ResponseLoginRes.deserializeBinary(reader.result);
                // console.log(rs)
            };
        }else if(self.communicationContentType == "json"){
            actionId = ev.data.substr(0,4);
            content = ev.data.substr(4);
            action = self.getActionName(actionId,"server")
            console.log(pre +" actionId:"+actionId , " content:",content , " actionName:",action);
            var content =  eval("("+content+")");
            self.router(action,content);
        }else{
            return alert("communicationContentType err");
        }



        // return alert(11);
    };
    this.router = function(action,content){
        console.log("router:",action,content)
        if ( action == 'loginRes' ) {
            self.rLoginRes(content);
        }else if( action == 'pushPlayerStatus'){//获取一个当前玩家的状态，如：是否有历史未结束的游戏
            self.rPushPlayerStatus(content);
        }else if( action == 'serverPing'){//获取一个当前玩家的状态，如：是否有历史未结束的游戏
            self.rPing(content);
        }else if ( action == 'startBattle' ){
            self.rStartBattle(content);
        }else if ( action == 'pushRoomInfo' ){
            self.rPushRoomInfo(content);
        }else if ( action == 'otherPlayerOffline' ){
            self.rOtherPlayerOffline(content);
        }else if ( action == 'enterBattle' ){
            self.rEnterBattle(content);
        }else if( "gameOver" == action){
            self.rOver(logicFrame);
        }else if( "pushLogicFrame" == action){
            self.rPushLogicFrame(content)
        }else if( "readyTimeout" == action){
            self.rReadyTimeout(content)
        }else if( "serverPong" == action){
            self.rServerPong(content)
        }else if( "otherPlayerResumeGame" == action){
            if(content.playerId != self.playerId){
                var tdId = self.tableId + "_" + self.playerLocation[content.playerId];
                var tdObj = $("#"+tdId)
                tdObj.css("background", "red");
            }
        }else if( "pushRoomHistory" == action){
            self.rPushRoomHistory(content);
            alert("接收到，玩家-房间-历史操作记录~");
        }else{
            return alert("action error.");
        }
    };
    this.rReadyTimeout= function(logicFrame){
        console.log("rReadyTimeout:",logicFrame);

        this.upStatus("loginSuccess");

        var matchSignBntId = "matchSign_"+self.playerId;
        var hrefBody = "连接成功，匹配报名";

        self.upOptBntHref(matchSignBntId,hrefBody,self.matchSign);

        return alert("抱歉，准确时间已超时");
    };
    this.rPushRoomHistory = function(logicFrame){
        console.log("rPushRoomHistory:");
        if(self.communicationContentType =="protobuf"){
            logicFrame.list = logicFrame.listList;
        }
        var list = logicFrame.list;
        for(var i=0;i<list.length;i++){
            // console.log( "rPushRoomHistory:" + logicFrame[i].Action);
            if (  list[i].action == "pushLogicFrame"){
                var data = eval( "(" + list[i].content + ")" )
                self.rPushLogicFrame(data);
            }
        }
        var requestPlayerResumeGame = new proto.main.RequestPlayerResumeGame();
        requestPlayerResumeGame.setRoomId(self.roomId);
        requestPlayerResumeGame.setSequenceNumber(self.sequenceNumber);
        requestPlayerResumeGame.setPlayerId(self.playerId);
        self.sendNewMsg("playerResumeGame",requestPlayerResumeGame);
    };
    this.upOptBntHref = function(domId,value,clickCallback){
        var bntContent = "<a href='javascript:void(0);' onclick='' id='"+domId+"'>"+value+"</a>";
        self.upOptBnt(bntContent, 1);
        $("#"+domId).click(clickCallback);
    };
    //=================== 以下都是 接收S端的处理函数========================================
    this.rLoginRes = function(logicFrame){
        if (logicFrame.code != 200) {
            this.status = "loginFailed";
            return alert("loginRes failed!!!"+logicFrame.code + " , "+logicFrame.errMsg);
        }
        var now = Date.now();
        var requestClientPing = new proto.main.RequestClientPing();
        requestClientPing.setAddTime(now);
        this.sendNewMsg("clientPing",requestClientPing);
        this.upStatus("loginSuccess");

        var playerConnInfo = logicFrame.player;
        if (playerConnInfo.roomId){
            alert("检测出，有未结束的一局游戏，开始恢复中...,先获取房间信息:rooId:"+playerConnInfo.roomId);
            var requestGetRoom = new proto.main.RequestGetRoom();
            requestGetRoom.setRoomId(playerConnInfo.roomId);
            requestGetRoom.setPlayerId(playerId);
            self.sendNewMsg("getRoom",requestGetRoom);
            //     var msg = {"roomId":playerConnInfo.roomId,"playerId":playerId};
        }else{
            var matchSignBntId = "matchSign_"+self.playerId;
            var hrefBody = "连接成功，匹配报名";

            self.upOptBntHref(matchSignBntId,hrefBody,self.matchSign);
        }
        // self.heartbeatLoopFunc = setInterval(self.heartbeat, 5000);
    };
    this.rServerPong = function(logicFrame){
        console.log("rServerPong:",logicFrame)
    };
    this.rPing = function(logicFrame){
        var now = Date.now();

        var requestClientPong = new proto.main.RequestClientPong();
        requestClientPong.setClientReceiveTime(now);
        requestClientPong.setAddTime(logicFrame.addTime);
        this.sendNewMsg("clientPong",requestClientPong);
        //     logicFrame.clientReceiveTime =  now

    };
    this.rStartBattle = function(logicFrame){
        // logicFrame.playerList = logicFrame.

        self.upStatus("startBattle");
        self.pushLogicFrameLoopFunc = setInterval(self.playerCommandPush,self.logicFrameLoopTimeMs);

        var exceptionOffLineId = "exceptionOffLineId"+self.playerId;
        // self.upOptBnt("游戏中...<a href='javascript:void(0);'  id='"+exceptionOffLineId+"'>异常掉线</a>",1)
        // $("#"+exceptionOffLineId).click(self.closeFD);

        var msg = "异常掉线";
        self.upOptBntHref(exceptionOffLineId,msg,self.closeFD);
    };
    this.rPushRoomInfo = function(logicFrame){
        if(this.communicationContentType =="protobuf"){
            logicFrame.playerList = logicFrame.playerListList;
        }
        self.initLocalGlobalVar(logicFrame);
        var requestRoomHistory = new proto.main.RequestRoomHistory();
        requestRoomHistory.setRoomId(self.roomId);
        requestRoomHistory.setSequenceNumberstart(0);
        requestRoomHistory.setSequenceNumberend(-1);
        requestRoomHistory.setPlayerId(self.playerId);
        // var history ={"roomId":self.roomId,"sequenceNumber":0,"playerId":self.playerId };
        // self.sendMsg("getRoomHistory",history);
        self.sendNewMsg("roomHistory",requestRoomHistory);

        var readySignBntId = "ready_"+self.playerId;
        var hrefBody = "匹配成功，准备";

        self.upOptBntHref(readySignBntId,hrefBody,self.ready);
    };
    this.rPushLogicFrame = function(logicFrame){//接收S端逻辑帧
        var pre = self.descPre;
        if(this.communicationContentType =="protobuf"){
            logicFrame.operations = logicFrame.operationsList;
        }
        var operations = logicFrame.operations;
        self.sequenceNumber  = logicFrame.sequenceNumber;
        $("#"+self.domIdObj.seqId).html(self.sequenceNumber);

        self.playerCommandPushLock = 0;

        console.log("rPushLogicFrame ,sequenceNumber:"+self.sequenceNumber+ ", operationsLen:" +  operations.length)
        for(var i=0;i<operations.length;i++){
            playerId= operations[i].playerId;
            var str = pre + " i=i , id: "+operations[i].id + " , event:"+operations[i].event + " , value:"+ operations[i].value + " , playerId:" + playerId;
            console.log(str);
            if (operations[i].event == "move"){
                var LocationArr = operations[i].value.split(",");
                var LocationStart = LocationArr[0];
                var LocationEnd = LocationArr[1];

                // var lightTd = "map"+id +"_"+LocationStart + "_"+LocationEnd;
                var lightTd =self.getMapTdId(self.tableId,LocationStart,LocationEnd);
                console.log(pre+"  "+lightTd);
                var tdObj = $("#"+lightTd);
                if(playerId == self.playerId){
                    tdObj.css("background", "green");
                }else{
                    tdObj.css("background", "red");
                }
                var playerLocation = self.playerLocation;
                if (playerLocation[playerId] == "empty"){
                    //证明是第一次移动，没有之前的数据
                }else{
                    // playerLocation = getPlayerLocation(playerId);
                    // alert(commands[i].playerId);
                    var playerLocationArr = playerLocation[playerId].split("_");
                    //非首次移动，这次移动后，要把之前所有位置清掉
                    var lightTd = self.getMapTdId(self.tableId,playerLocationArr[0],playerLocationArr[1]);
                    var tdObj = $("#"+lightTd);
                    tdObj.css("background", "");
                }
                playerLocation[playerId] = LocationStart + "_"+LocationEnd;
            }else if(operations[i].event == "empty"){

            }
        }
        // self.sendPlayerLogicFrameAck( self.sequenceNumber)
    };

    // this.rPushPlayerStatus = function(logicFrame){
    //     console.log("pushPlayerStatus:"+logicFrame.status)
    //     if (logicFrame.roomId ){//有未完结的记录
    //     }else{
    //
    //     }
    // };
    this.rOtherPlayerOffline = function(logicFrame){
        //房间内有其它玩家掉线了
        console.log("in test:",logicFrame.playerId,logicFrame)
        self.otherPlayerOffline = logicFrame.playerId;
        alert("其它玩家掉线了："+logicFrame.playerId +"，略等："+self.offLineWaitTime +"秒");

        var tdId = self.tableId + "_" + self.playerLocation[logicFrame.playerId];
        var tdObj = $("#"+tdId)
        tdObj.css("background", "#A9A9A9");
        // var lightTd =self.getMapTdId(self.tableId,LocationStart,LocationEnd);
        // console.log(pre+"  "+lightTd);
        // var tdObj = $("#"+lightTd);
        // if(commands[i].playerId == playerId){
        //     tdObj.css("background", "green");
        // }else{
        //     tdObj.css("background", "red");
        // }
    };
    this.initLocalGlobalVar = function(logicFrame){
        var pre = self.descPre;
        console.log("initLocalGlobalVar:",logicFrame)
        for(var i=0;i<logicFrame.playerList.length;i++){
            self.playerLocation[""+logicFrame.playerList[i].id+""] = "empty"
        }
        // return 1;
        self.randSeek  = logicFrame.randSeek;
        // $("#randseek").html(self.randSeek);
        $("#"+self.domIdObj.randSeekId).html(self.randSeek);


        self.sequenceNumber  = logicFrame.sequenceNumber;
        // $("#sid").html(self.sequenceNumber);
        $("#"+self.domIdObj.seqId).html(self.sequenceNumber);

        self.roomId = logicFrame.roomId;
        // $("#rid").html(logicFrame.RoomId);
        $("#"+self.domIdObj.roomId).html(self.roomId);

        var str =  pre+", roomId:" +logicFrame.roomId+ ", RandSeek:"+    self.randSeek +" , SequenceNumber"+    self.sequenceNumber ;
        console.log(str);
    };
    this.rEnterBattle = function(logicFrame){
        if(this.communicationContentType =="protobuf"){
            logicFrame.playerList = logicFrame.playerListList;
        }

        self.initLocalGlobalVar(logicFrame);

        var readySignBntId = "ready_"+self.playerId;
        var hrefBody = "匹配成功，准备";

        self.upOptBntHref(readySignBntId,hrefBody,self.ready);

    };
    this.rOver = function(ev){
        window.clearInterval(self.pushLogicFrameLoopFunc);
        self.upOptBnt("游戏结束2",1)
    }
    //=================== 以上都是 接收S端的处理函数========================================

    this.getMap = function (tableId) {
        // var tableDivPre = "map";
        this.tableId = tableId;
        var tableObj = $("#" + tableId);
        var matrix = new Array();
        var matrixSize = this.tableMax;
        var inc = 0;
        for (var i = 0; i < matrixSize; i++) {
            matrix[i] = new Array();
            var trTemp = $("<tr></tr>");
            for (var j = 0; j < matrixSize; j++) {
                // var tdId = tableId + "_" + i +"_" + j;
                var tdId = this.getMapTdId(tableId,i,j);
                matrix[i][j] = inc;
                trTemp.append("<td id='"+tdId+"'>"+ i +","+j +"</td>");
                inc++;
            }
            // alert(trTemp);
            trTemp.appendTo(tableObj);
        }
    };
    this.ready = function(){
        self.upStatus("ready");
        var requestPlayerReady = new proto.main.RequestPlayerReady();
        requestPlayerReady.setPlayerId(self.playerId);
        requestPlayerReady.setRoomId(self.roomId);
        self.sendNewMsg("playerReady",requestPlayerReady);
        // var msg = {"playerId" :self.playerId};
        // self.sendMsg("playerReady",msg)

        // var cancelBntId = "cancelSign_"+self.playerId;
        // var bntContent = "<a href='javascript:void(0);' onclick='' id='"+cancelBntId+"'>取消报名/准备</a>";
        // self.upOptBnt(bntContent, 1);
        // $("#"+cancelBntId).click(self.cancelSign);
        self.upOptBntHref("","等待其它玩家准备",this.ready);
    };
    this.cancelSign = function(){
        self.upStatus("cancelSign");

        var requestPlayerMatchSignCancel = new proto.main.RequestPlayerMatchSignCancel();
        requestPlayerMatchSignCancel.setPlayerId(self.playerId);
        self.sendNewMsg("playerMatchSignCancel",requestPlayerMatchSignCancel);
        // var msg = {"playerId" :self.playerId};
        // self.sendMsg("playerMatchSignCancel",msg)

        var matchSignBntId = "matchSign_"+self.playerId;
        var hrefBody = "连接成功，匹配报名";

        self.upOptBntHref(matchSignBntId,hrefBody,self.matchSign);

        // var readyBntId = "ready_"+self.playerId;
        // var bntContent = "连接成功，<a href='javascript:void(0);' onclick='' id='"+readyBntId+"'>准备/报名</a>";
        // self.upOptBnt(bntContent, 1);
        // $("#"+readyBntId).click(self.ready);
    };
    this.matchSign = function(){
        self.upStatus("matchSign")

        var requestPlayerMatchSign = new proto.main.RequestPlayerMatchSign();
        requestPlayerMatchSign.setPlayerId(self.playerId);
        self.sendNewMsg("playerMatchSign",requestPlayerMatchSign);

        // var msg = {"playerId" :self.playerId};
        // self.sendMsg("playerMatchSign",msg);

        var cancelBntId = "cancelSign_"+self.playerId;
        var hrefBody = "取消匹配报名";

        self.upOptBntHref(cancelBntId,hrefBody,self.cancelSign);

        // var cancelBntId = "cancelSign_"+self.playerId;
        // var bntContent = "<a href='javascript:void(0);' onclick='' id='"+cancelBntId+"'>取消报名/准备</a>";
        // self.upOptBnt(bntContent, 1);
        // $("#"+cancelBntId).click(self.cancelSign);
    };
    this.move = function ( dirObj ){

        if (self.otherPlayerOffline){
            return alert("其它玩家掉线了，请等待一下...");
        }

        if (self.closeFlag > 0 ){
            return alert("WS FD 已关闭...");
        }

        if (self.status != "startBattle"){
            return alert("status err , != startBattle ， 游戏还未开始，请等待一下...");
        }

        var dir = dirObj.data;
        var playerLocation = self.playerLocation;
        var nowLocationStr = playerLocation[self.playerId]
        var nowLocationArr = nowLocationStr.split("_");
        var nowLocationLine =  nowLocationArr[0];
        var nowLocationColumn = nowLocationArr[1];

        nowLocationLine = Number(nowLocationLine)
        nowLocationColumn = Number(nowLocationColumn)
        var newLocation = "";
        if(dir == "up"){
            if(nowLocationLine == 0 ){
                return alert("nowLocationLine == 0");
            }
            var newLocationLine =  nowLocationLine - 1;
            newLocation = newLocationLine + "," + nowLocationColumn;
        }else if(dir == "down"){
            if(nowLocationLine == self.tableMax - 1 ){
                return alert("nowLocationLine == "+ self.tableMax);
            }
            var newLocationLine =  nowLocationLine + 1;
            newLocation = newLocationLine + "," + nowLocationColumn;
        }else if(dir == "left"){
            if(nowLocationColumn == 0 ){
                return alert("nowLocationColumn == 0");
            }
            var newLocationColumn =  nowLocationColumn - 1;
            newLocation = nowLocationLine + "," + newLocationColumn;
        }else if(dir == "right"){
            if(nowLocationColumn ==  self.tableMax - 1 ){
                return alert("nowLocationColumn == "+ self.tableMax);
            }
            var newLocationColumn =  nowLocationColumn + 1;
            newLocation = nowLocationLine + "," + newLocationColumn;
        }else {
            return alert("move dir error."+dir);
        }

        var localNewLocation = newLocation.replace(',','_');
        for(let key  in playerLocation){
            // alert(playerLocation[key]);
            if(playerLocation[key] == localNewLocation){
                 return self.gameOverAndClear()
            }
        }

        console.log("dir:"+dir+"oldLocation"+nowLocationStr+" , newLocation:"+newLocation);
        // var commands ={"id":3,"roomId":self.roomId,"sequenceNumber":self.sequenceNumber,"commands": [{"id":1,"action":"move","value":newLocation,"playerId":self.playerId}]};
        // self.sendMsg("playerCommandPush",commands)
        self.playerOperationsQueue.push({"id":self.operationsInc,"event":"move","value":newLocation,"playerId":self.playerId});
        self.operationsInc++;
        var playerLocationArr = playerLocation[self.playerId].split("_");
        var lightTd = self.getMapTdId(self.tableId,playerLocationArr[0],playerLocationArr[1]);
        var tdObj = $("#"+lightTd);
        tdObj.css("background", "");
    }
    this.playerCommandPush = function (){
        var requestPlayerOperations = new proto.main.RequestPlayerOperations();
        requestPlayerOperations.setId(3);
        requestPlayerOperations.setRoomId(self.roomId);
        requestPlayerOperations.setSequenceNumber(self.sequenceNumber);

        // var loginFrame ={"id":3,"roomId":self.roomId,"sequenceNumber":self.sequenceNumber,"operations": []};
        if(self.playerCommandPushLock == 1){
            console.log("send msg lock...please wait server back login frame");
            return
        }
        if (self.playerOperationsQueue.length > 0){
            // {"id":self.operationsInc,"event":"move","value":newLocation,"playerId":self.playerId}
            var operations = new Array(self.playerOperationsQueue.length)
            for(var i=0;i<self.playerOperationsQueue.length;i++){
                var operation = new proto.main.Operation();
                operation.setId(self.playerOperationsQueue[i].id);
                operation.setEvent(self.playerOperationsQueue[i].event);
                operation.setValue(self.playerOperationsQueue[i].value);
                operation.setPlayerId(self.playerId),
                // operations.push(operation);
                operations[i] = operation;
            }
            requestPlayerOperations.setOperationsList(operations);
            // loginFrame.operations = self.playerOperationsQueue;
            self.playerOperationsQueue = [];//将当前队列里的，当前帧的数据，清空
        }else{
            // window.clearInterval(self.pushLogicFrameLoopFunc);

            var operations = new Array(1);
            var operation = new proto.main.Operation();
            operation.setId(1);
            operation.setEvent("empty");
            operation.setValue("-1");
            operation.setPlayerId(self.playerId),
            // operations.push(operation);
            operations[0] = operation;
            // console.log(operations)
            requestPlayerOperations.setOperationsList(operations);
            // var emptyCommand = [{"id":1,"event":"empty","value":"-1","playerId":self.playerId}];
            // loginFrame.operations = emptyCommand;
        }
        self.sendNewMsg("playerOperations",requestPlayerOperations);
        self.playerCommandPushLock = 1;
        // self.sendMsg("playerOperations",loginFrame);
    };


    this.getPlayerDescById = function (id){
        return "player_"+ id;
    };
    this.getMapTdId = function (tableId,i,j){
        return tableId + "_" + i +"_" + j;
    }

    this.descPre = this.getPlayerDescById(playerId);
};


if ("WebSocket" in window) {
    console.log("browser has websocket lib.");
}else {
    // 浏览器不支持 WebSocket
    alert("您的浏览器不支持 WebSocket!");
}

function stringToUint8Array(str){
    var arr = [];
    for (var i = 0, j = str.length; i < j; ++i) {
        arr.push(str.charCodeAt(i));
    }

    var tmpUint8Array = new Uint8Array(arr);
    return tmpUint8Array
}