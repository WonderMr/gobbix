var o = new ActiveXObject("V83.COMConnector")
try{
    o.Connect("Srvr=\"localhost:2541\";Ref=\"test1\";");
}catch(e){
    WScript.Echo(e.name+" "+e.number);
}finally{
    o = null
}