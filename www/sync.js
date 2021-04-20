var self = null;
function ws (playerId,token,host,uri,matchGroupPeople,tableMax,DomIdObj,offLineWaitTime,actionMap){
    var self = this;
    this.hostUri = host +uri;//ws 连接地址
    this.playerId = playerId;//玩家ID
    this.token = token;//玩家的凭证
    this.matchGroupPeople = matchGroupPeople;//一个副本的人数
    this.tableMax = tableMax;//地址的表格大小
    this.wsObj = null;//js内置ws 对象
    this.otherPlayerOffline = 0;//其它玩家调线
    this.heartbeatLoopFunc = null;//心跳回调函数
    this.myClose = 0;//C端 主动关闭标识
    this.domIdObj =DomIdObj ;
    this.offLineWaitTime = offLineWaitTime;
    this.playerLocation = new Object();
    this.tableId = "";
    //下面，是由S端供给
    this.roomId = "";
    this.actionMap = actionMap;
    this.sequenceNumber = 0;
    this.randSeek = 0;
    //入口函数，必须得建立连接后，都有后续的所有操作
    this.create  = function(){
        console.log("create new WebSocket"+self.hostUri)
        self.wsObj = new WebSocket(self.hostUri);
        // self.wsOpen();
        self.wsObj.onclose = function(ev){
            self.onclose(ev)
        };
        self.wsObj.onmessage = function(ev){  //获取后端响应
            self.onmessage(ev)
        };
        self.wsObj.onopen = function(){
            self.wsOpen();
        };
    };
    //连接成功后，会执行此函数
    this.wsOpen = function(){
        console.log("ws link success , onOpen")
        // var data = '{"action":"login","content":"'+self.token+'"}';
        // self.sendById(data)
        var data = {"token":self.token}
        self.sendMsg("login",data)
    };
    this.gameOverAndClear = function(){
        commands ={"roomId":self.roomId,"sequenceNumber":self.sequenceNumber,"result": "aaaa"};
        // var msg = {"action":"gameOver","content":JSON.stringify(commands)}
        // var jsonStr = JSON.stringify(msg)
        // self.sendById(jsonStr);
        self.sendMsg("gameOver",commands,1)

        self.upOptBnt("游戏结束1",1)
        return alert("完犊子了，撞车了...这游戏得结束了....");
    };
    // this.sendById =  function (id ,msg){
    // this.sendById =  function ( msg ){
    //     console.log(self.descPre + " sendById:"+msg)
    //     self.wsObj.send(msg);
    // };
    this.getActionId = function (action,category){
        var data = self.actionMap[category];
        for(let key  in data){
            // console.log(data[key].Action);
            if (data[key].Action == action){
                return data[key].Id;
            }
        }
        return "";
    };
    this.getActionName = function (actionId,category){
        var data = self.actionMap[category];
        // alert(data[actionId]);
        return data[actionId].Action;
    };

    this.sendMsg =  function ( action,content,jsonEncode  ){
        // if (jsonEncode == 1){
            content = JSON.stringify(content)
        // }
        // var msg = {"action":action,"content":content};
        // var jsonStr =  JSON.stringify(msg);
        var id = self.getActionId(action,"client");
        content = id+content;
        console.log(self.descPre + " sendMsg:",content)
        self.wsObj.send(content);
    };
    this.upOptBnt = function(content,clearClick){
        $("#"+self.domIdObj.optBntId).html(content);
        if(clearClick == 1){
            $("#"+self.domIdObj.optBntId).unbind("click");
        }
    };
    this.closeFD = function (){
        console.log("closeFD");
        window.clearInterval(self.heartbeatLoopFunc);
        self.myClose = 1;
        self.wsObj.close();
    };
    this.wsError = function(){
        this.wsObj.onerror = function(){
            console.log("error:"+ev);
            alert(ev);
        }
    };
    this.heartbeat = function(){
        // var msg = {"action":"heartbeat","content":""}
        // self.sendById(JSON.stringify(msg));
        var msg = {"time":Date.now()};
        self.sendMsg("clientHeartbeat",msg)
    };
    this.onclose = function(ev){
        alert("receive server close:" +ev.code);
        window.clearInterval(self.heartbeatLoopFunc);
        var reConnBntId = "reconn_"+self.playerId;
        if (self.myClose == 1){
            self.upOptBnt("C端主动关闭WS，<a href='javascript:void(0);' id='"+reConnBntId+"'>重连接</a>",1)
            $("#"+reConnBntId).click(self.create);
        }else{
            self.upOptBnt("服务端关闭，游戏结束，连接断开",1)
        }
    };

    this.sendPlayerLogicFrameAck = function ( SequenceNumber){
        var logicFrame ={"roomId":this.roomId,"sequenceNumber":SequenceNumber};
        // var msg = {"action":"playerLogicFrameAck","content":JSON.stringify(logicFrame)}
        // var jsonStr = JSON.stringify(msg)
        // this.sendById(jsonStr);
        self.sendMsg("playerLogicFrameAck",logicFrame,1);
    }
    this.onmessage = function(ev){
        var pre = self.descPre;
        console.log("onmessage:"+ pre + " " +ev.data);
        var actionId = ev.data.substr(0,4);
        var content = ev.data.substr(4);
        var action = self.getActionName(actionId,"server")
        console.log(pre +" actionId:"+actionId , " content:",content , " actionName:",action);
        // var msg = eval("("+ev.data+")");
        // console.log(pre +" action:"+msg.action);
        // console.log(pre +" content:"+msg.content);
        // var logicFrame =  eval("("+msg.content+")");
        // if ( msg.action == 'loginRes' ) {
        //     self.rLoginRes(logicFrame);
        // }else if( msg.action == 'pushPlayerStatus'){//获取一个当前玩家的状态，如：是否有历史未结束的游戏
        //     self.rPushPlayerStatus(logicFrame);
        // }else if( msg.action == 'ping'){//获取一个当前玩家的状态，如：是否有历史未结束的游戏
        //     self.rPing(logicFrame);
        // }else if ( msg.action == 'otherPlayerOffline' ){
        //     self.rOtherPlayerOffline(logicFrame);
        // }else if ( msg.action == 'startInit' ){
        //     self.rStartInit(logicFrame);
        // }else if( "over" == msg.action){
        //     self.rOver(logicFrame);
        // }else if( "pushLogicFrame" == msg.action){
        //     self.rPushLogicFrame(logicFrame)
        // }else{
        //     return alert("action error.");
        // }
        var logicFrame =  eval("("+content+")");
        if ( action == 'loginRes' ) {
            self.rLoginRes(logicFrame);
        }else if( action == 'pushPlayerStatus'){//获取一个当前玩家的状态，如：是否有历史未结束的游戏
            self.rPushPlayerStatus(logicFrame);
        }else if( action == 'serverPing'){//获取一个当前玩家的状态，如：是否有历史未结束的游戏
            self.rPing(logicFrame);
        }else if ( action == 'otherPlayerOffline' ){
            self.rOtherPlayerOffline(logicFrame);
        }else if ( action == 'startInit' ){
            self.rStartInit(logicFrame);
        }else if( "gameOver" == action){
            self.rOver(logicFrame);
        }else if( "pushLogicFrame" == action){
            self.rPushLogicFrame(logicFrame)
        }else if( "otherPlayerResumeGame" == action){
            alert("玩家断线恢复喽~");
        }else{
            return alert("action error.");
        }
    };
    //=================== 以下都是 接收S端的处理函数========================================
    this.rPushLogicFrame = function(logicFrame){
        var pre = self.descPre;

        var commands = logicFrame.commands;
        self.sequenceNumber  = logicFrame.SequenceNumber;
        $("#"+self.domIdObj.seqId).html(self.sequenceNumber);

        console.log("sequenceNumber:"+self.sequenceNumber+ ", commandLen:" +  commands.length)
        for(var i=0;i<commands.length;i++){
            str = pre + " , "+commands[i].action + " , "+ commands[i].value + " , " + commands[i].playerId;
            console.log(str);
            if (commands[i].action == "move"){
                var LocationArr = commands[i].value.split(",");
                var LocationStart = LocationArr[0];
                var LocationEnd = LocationArr[1];

                // var lightTd = "map"+id +"_"+LocationStart + "_"+LocationEnd;
                var lightTd =self.getMapTdId(self.tableId,LocationStart,LocationEnd);
                console.log(pre+"  "+lightTd);
                var tdObj = $("#"+lightTd);
                if(commands[i].playerId == playerId){
                    tdObj.css("background", "green");
                }else{
                    tdObj.css("background", "red");
                }
                var playerLocation = self.playerLocation;
                if (playerLocation[commands[i].playerId] == "empty"){
                    //证明是第一次移动，没有之前的数据
                }else{
                    // playerLocation = getPlayerLocation(playerId);
                    // alert(commands[i].playerId);
                    var playerLocationArr = playerLocation[commands[i].playerId].split("_");
                    //非首次移动，这次移动后，要把之前所有位置清掉
                    var lightTd = self.getMapTdId(self.tableId,playerLocationArr[0],playerLocationArr[1]);
                    var tdObj = $("#"+lightTd);
                    tdObj.css("background", "");
                }
                playerLocation[commands[i].playerId] = LocationStart + "_"+LocationEnd;
            }
        }
        self.sendPlayerLogicFrameAck( self.sequenceNumber)
    };
    this.rPing = function(logicFrame){
        var now = Date.now();
        // var msg = {"action": "pong", "content": now}
        // var jsonStr = JSON.stringify(msg)
        logicFrame.clientReceiveTime =  now
        self.sendMsg("clientPong",logicFrame,1)
    };
    this.rLoginRes = function(logicFrame){
        if (logicFrame.code != 200 && logicFrame.code != 201) {
            return alert("loginRes failed!!!"+logicFrame.code + " , "+logicFrame.content);
        }

        if (logicFrame.code == 201){
            alert("还有未结束的游戏，请查看，roomId:"+logicFrame.content);
        }

        var readyBntId = "ready_"+self.playerId;
        var bntContent = "连接成功，<a href='javascript:void(0);' onclick='' id='"+readyBntId+"'>准备/报名</a>";
        self.upOptBnt(bntContent, 1);
        $("#"+readyBntId).click(self.ready);


        self.heartbeatLoopFunc = setInterval(self.heartbeat, 5000);
        // var msg = {"action": "playerStatus", "content": ""}
        // var jsonStr = JSON.stringify(msg)
        // self.sendById(jsonStr)
        var msg = {"playerId": this.playerId}
        self.sendMsg("playerStatus",msg)
    };
    this.rPushPlayerStatus = function(logicFrame){
        console.log("pushPlayerStatus:"+logicFrame.status)
        if (logicFrame.roomId ){//有未完结的记录
            alert("检测出，有未结束的一局游戏，开始恢复中...");
            var commands ={"roomId":self.roomId,"sequenceNumber":self.sequenceNumber,"playerId":self.playerId };
            self.sendMsg("playerResumeGame",commands)
        }else{

        }
    };
    this.rOtherPlayerOffline = function(logicFrame){
        //房间内有其它玩家掉线了
        self.otherPlayerOffline = logicFrame;
        return alert("其它玩家掉线了："+logicFrame.playerId +"，略等："+self.offLineWaitTime +"秒");
    };
    this.rStartInit = function(logicFrame){
        var pre = self.descPre;
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

        var str =  pre+", RandSeek:"+    self.randSeek +" , SequenceNumber"+    self.sequenceNumber ;
        console.log(str);

        var exceptionOffLineId = "exceptionOffLineId"+self.playerId;
        self.upOptBnt("游戏中...<a href='javascript:void(0);'  id='"+exceptionOffLineId+"'>异常掉线</a>",1)
        $("#"+exceptionOffLineId).click(self.closeFD)
        // $("#"+self.domIdObj.optBntId).html("游戏中...");
        // $("#"+self.domIdObj.optBntId).html("游戏中...");

        self.sendPlayerLogicFrameAck( self.sequenceNumber)
    };
    this.rOver = function(ev){
        self.upOptBnt("游戏结束2",1)
        // $("#"+self.domIdObj.optBntId).html("游戏结束2");
        // return alert("游戏结束");
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
    this.cancelSign = function(){
        var msg = {"playerId" :self.playerId};
        self.sendMsg("playerCancelReady",msg)

        var readyBntId = "ready_"+self.playerId;
        var bntContent = "连接成功，<a href='javascript:void(0);' onclick='' id='"+readyBntId+"'>准备/报名</a>";
        self.upOptBnt(bntContent, 1);
        $("#"+readyBntId).click(self.ready);
    };
    this.ready = function(){
        var msg = {"playerId" :self.playerId};
        self.sendMsg("playerReady",msg)

        var cancelBntId = "cancelSign_"+self.playerId;
        var bntContent = "<a href='javascript:void(0);' onclick='' id='"+cancelBntId+"'>取消报名/准备</a>";
        self.upOptBnt(bntContent, 1);
        $("#"+cancelBntId).click(self.cancelSign);

    };
    this.move = function ( dirObj ){
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


        if (self.otherPlayerOffline){
            return alert("玩家掉线了，请等待一下...");
        }

        var localNewLocation = newLocation.replace(',','_');
        for(let key  in playerLocation){
            // alert(playerLocation[key]);
            if(playerLocation[key] == localNewLocation){
                 return self.gameOverAndClear()
            }
        }


        // var vsPlayerId = 0;
        // if (self.playerId == 1){
        //     vsPlayerId = 2;
        // }else if(self.playerId == 2){
        //     vsPlayerId = 1;
        // }else{
        //     return alert("id error.");
        // }
        //
        // if(playerLocation[vsPlayerId] == localNewLocation){
        //     commands ={"RoomId":self.roomId,"SequenceNumber":self.sequenceNumber,"Commands": []};
        //     var msg = {"action":"gameOver","content":JSON.stringify(commands)}
        //     // var msg = {"action":"gameOver","content":""}
        //     var jsonStr = JSON.stringify(msg)
        //     this.sendById(jsonStr);
        //     return alert("撞车了不能动了...");
        // }

        console.log("dir:"+dir+"oldLocation"+nowLocationStr+" , newLocation:"+newLocation);

        var commands ={"roomId":self.roomId,"sequenceNumber":self.sequenceNumber,"commands": [{"action":"move","value":newLocation,"playerId":self.playerId}]};
        // var msg = {"action":"playerCommandPush","content":JSON.stringify(commands)}
        // var jsonStr = JSON.stringify(msg)
        // self.sendById(jsonStr);
        self.sendMsg("playerCommandPush",commands)

        // var playerLocation = getPlayerLocation(id);
        var playerLocationArr = playerLocation[self.playerId].split("_");
        // var lightTd = "map"+id +"_"+playerLocation[id];
        var lightTd = self.getMapTdId(self.tableId,playerLocationArr[0],playerLocationArr[1]);
        var tdObj = $("#"+lightTd);
        tdObj.css("background", "");
    }

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

