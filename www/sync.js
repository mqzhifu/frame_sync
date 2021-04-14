var self = null;
function ws (playerId,token,host,uri,matchGroupPeople,tableMax,DomIdObj){
    var self = this;
    this.hostUri = host +uri;
    this.playerId = playerId;
    this.token = token;
    this.matchGroupPeople = matchGroupPeople;
    this.tableMax = tableMax;
    this.wsObj = null;
    this.roomId = "";
    this.sequenceNumber = 0;
    this.heartbeatLoopFunc = null;
    this.randSeek = 0;
    this.domIdObj =DomIdObj ;
    // this.playerLocation = {1:"empty",2:"empty"}
    this.playerLocation = new Object();
    this.getPlayerDescById = function (id){
        return "player_"+ id;
    };
    this.gameOverAndClear = function(){
        commands ={"RoomId":self.roomId,"SequenceNumber":self.sequenceNumber,"Commands": []};
        var msg = {"action":"gameOver","content":JSON.stringify(commands)}
        var jsonStr = JSON.stringify(msg)
        self.sendById(jsonStr);
        // $("#"+self.domIdObj.optBntId).html("游戏结束1");
        self.upOptBnt("游戏结束1",1)
        return alert("完犊子了，撞车了...这游戏得结束了....");
    };
    // this.sendById =  function (id ,msg){
    this.sendById =  function ( msg ){
        // var wsObj = getPlayerWsObj(id);
        // var pre = getPlayerDescById(id);
        console.log(self.descPre + " send:"+msg)
        // alert(this.wsObj);
        // wsObj.send(msg);
        self.wsObj.send(msg);
    };
    this.upOptBnt = function(content,clearClick){
        $("#"+self.domIdObj.optBntId).html(content);
        if(clearClick == 1){
            $("#"+self.domIdObj.optBntId).unbind("click");
        }
    };
    this.closeFD = function (){
        console.log("closeFD");
        this.wsObj.close();
    };
    this.wsOpen = function(){
        //建立连接成功
        this.wsObj.onopen = function(){
            console.log("onOpen")
            var data = '{"action":"login","content":"'+self.token+'"}';
            // console.log("send:"+data);
            self.sendById(data)
        };
    };
    this.wsError = function(){
        this.wsObj.onerror = function(){
            console.log("error:"+ev);
            alert(ev);
        }
    };
    this.heartbeat = function(){
        var msg = {"action":"heartbeat","content":""}
        self.sendById(JSON.stringify(msg));
    };
    this.create  = function(){
        self.wsObj = new WebSocket(self.hostUri);
        self.wsOpen();
        // this.wsOnMessage();
    // }
    // this.wsOnMessage = function(){
        var pre = self.descPre;
        self.wsObj.onclose = function(ev){
            alert("receive server close:" +ev.code);
            window.clearInterval(self.heartbeatLoopFunc);
            // $("#"+self.domIdObj.optBntId).html("服务端关闭，游戏结束，连接断开");
            self.upOptBnt("服务端关闭，游戏结束，连接断开",1)
        };
        self.wsObj.onmessage = function(ev){  //获取后端响应
            console.log("onmessage:"+ pre +" "+ev.data);
            var msg = eval("("+ev.data+")");
            console.log(pre +" action:"+msg.action);
            console.log(pre +" content:"+msg.content);

            var logicFrame =  eval("("+msg.content+")");
            if ( msg.action == 'loginRes' ){
                if (logicFrame.code != 200){
                    return alert("loginRes failed!!!");
                }
                var bntContent = "连接成功，等待中...<a href='javascript:void(0);' onclick=''>取消</a>";
                self.upOptBnt(bntContent,1);
                // $("#"+self.domIdObj.optBntId).html("连接成功，等待中...");
                self.heartbeatLoopFunc = setInterval(self.heartbeat,5000);

            }else if ( msg.action == 'start_init' ){
                for(var i=0;i<logicFrame.PlayerList.length;i++){
                    self.playerLocation[""+logicFrame.PlayerList[i].Id+""] = "empty"
                    // alert(logicFrame.PlayerList[i].Id);
                }

                // return 1;
                self.randSeek  = logicFrame.RandSeek;
                // $("#randseek").html(self.randSeek);
                $("#"+self.domIdObj.randSeekId).html(self.randSeek);


                self.sequenceNumber  = logicFrame.SequenceNumber;
                // $("#sid").html(self.sequenceNumber);
                $("#"+self.domIdObj.seqId).html(self.sequenceNumber);

                self.roomId = logicFrame.RoomId;
                // $("#rid").html(logicFrame.RoomId);
                $("#"+self.domIdObj.roomId).html(self.roomId);

                var str =  pre+", RandSeek:"+    self.randSeek +" , SequenceNumber"+    self.sequenceNumber ;
                console.log(str);

                self.upOptBnt("游戏中...<a href='javascript:void(0);' onclick=''>异常掉线</a>",1)
                // $("#"+self.domIdObj.optBntId).html("游戏中...");
                // $("#"+self.domIdObj.optBntId).html("游戏中...");

                self.sendPlayerLogicFrameAck( self.sequenceNumber)
            }else if( "pushLogicFrame" == msg.action){
                var Commands = logicFrame.Commands;
                self.sequenceNumber  = logicFrame.SequenceNumber;
                $("#"+self.domIdObj.seqId).html(self.sequenceNumber);

                console.log("sequenceNumber:"+self.sequenceNumber+ ", commandLen:" + Commands.length)
                for(var i=0;i<Commands.length;i++){
                    str = pre + " , "+Commands[i].Action + " , "+ Commands[i].Value + " , " + Commands[i].PlayerId;
                    console.log(str);
                    if (Commands[i].Action == "move"){
                        var LocationArr = Commands[i].Value.split(",");
                        var LocationStart = LocationArr[0];
                        var LocationEnd = LocationArr[1];

                        // var lightTd = "map"+id +"_"+LocationStart + "_"+LocationEnd;
                        var lightTd =self.getMapTdId(self.tableId,LocationStart,LocationEnd);
                        console.log(pre+"  "+lightTd);
                        var tdObj = $("#"+lightTd);
                        if(Commands[i].PlayerId == playerId){
                            tdObj.css("background", "green");
                        }else{
                            tdObj.css("background", "red");
                        }
                        var playerLocation = self.playerLocation;
                        if (playerLocation[Commands[i].PlayerId] == "empty"){
                            //证明是第一次移动，没有之前的数据
                        }else{
                            // playerLocation = getPlayerLocation(playerId);
                            var playerLocationArr = playerLocation[Commands[i].PlayerId].split("_");
                            //非首次移动，这次移动后，要把之前所有位置清掉
                            var lightTd = self.getMapTdId(self.tableId,playerLocationArr[0],playerLocationArr[1]);
                            var tdObj = $("#"+lightTd);
                            tdObj.css("background", "");
                        }
                        playerLocation[Commands[i].PlayerId] = LocationStart + "_"+LocationEnd;
                    }
                }
                self.sendPlayerLogicFrameAck( self.sequenceNumber)


            }else if( "over" == msg.action){
                self.upOptBnt("游戏结束2",1)
                // $("#"+self.domIdObj.optBntId).html("游戏结束2");
                // return alert("游戏结束");
            }else{
                return alert("action error.");
            }
        };
    };
    this.getMapTdId = function (tableId,i,j){
        return tableId + "_" + i +"_" + j;
    }
    this.sendPlayerLogicFrameAck = function ( SequenceNumber){
        // var wsObj = getPlayerWsObj(playerId);

        var logicFrame ={"RoomId":this.roomId,"SequenceNumber":SequenceNumber};
        // return alert(this.RoomId);
        var msg = {"action":"playerLogicFrameAck","content":JSON.stringify(logicFrame)}
        var jsonStr = JSON.stringify(msg)
        // return alert(jsonStr);
        // sendById(playerId,jsonStr);
        this.sendById(jsonStr);
    }
    this.tableId = "";
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

        commands ={"RoomId":self.roomId,"SequenceNumber":self.sequenceNumber,"Commands": [{"Action":"move","Value":newLocation,"PlayerId":self.playerId}]};
        var msg = {"action":"playerCommandPush","content":JSON.stringify(commands)}

        // var playerLocation = getPlayerLocation(id);
        var playerLocationArr = playerLocation[self.playerId].split("_");
        // var lightTd = "map"+id +"_"+playerLocation[id];
        var lightTd = self.getMapTdId(self.tableId,playerLocationArr[0],playerLocationArr[1]);
        var tdObj = $("#"+lightTd);
        tdObj.css("background", "");

        var jsonStr = JSON.stringify(msg)
        // wsObj = getPlayerWsObj(id);
        self.sendById(jsonStr);
    }

    this.descPre = this.getPlayerDescById(playerId);
};


if ("WebSocket" in window) {
    console.log("browser has websocket lib.");
}else {
    // 浏览器不支持 WebSocket
    alert("您的浏览器不支持 WebSocket!");
}

