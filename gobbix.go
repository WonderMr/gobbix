 package main
//----------------------------------------------------------------------------------------------------------------------
import (
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "net/url"
    "os"
    "regexp"
    "strings"
    ole "github.com/go-ole/go-ole"
    "github.com/go-ole/go-ole/oleutil"
    "time"
)
//----------------------------------------------------------------------------------------------------------------------
var (
    re_1c_check_base                =   regexp.MustCompile("check\\savailability\\s(.*)\\severy\\s(.*)\\snotify\\s(.*)")
    re_quotes                       =   regexp.MustCompile(`\'|\"|\r|\n`)
    //proxyURL, err                   =   url.Parse("http://110.235.207.203:3128")
    transport                       =   &http.Transport{
        //Proxy: http.ProxyURL(proxyURL),
    }
    client                          =   &http.Client{
        Transport                   :   transport,
        Timeout                     :   time.Second * 15,
    }
)
//----------------------------------------------------ё------------------------------------------------------------------
type Block struct {
     Try     func()
     Catch   func(Exception)
     Finally func()
}
type Exception interface{}
    func Throw(up Exception) {
        panic(up)
    }
    func (tcf Block) Do() {
        if tcf.Finally              !=  nil {
            defer tcf.Finally()
        }
        if tcf.Catch                !=  nil {
            defer func() {
            if r                    :=  recover(); r !=  nil {
                tcf.Catch(r)
            }
        }()
    }
    tcf.Try()
}
//----------------------------------------------------------------------------------------------------------------------
func log_it(msg string) {
    log.SetOutput(os.Stdout)
    log.Print(msg)
    log_file                        :=  os.Args[0]+".log"
    f, err                          :=  os.OpenFile(log_file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
    if err                          !=  nil {
        log.Fatalf("error opening file: %v", err)
    }
    defer f.Close()
    log.SetOutput(f)
    log.Println(msg)
}
//----------------------------------------------------------------------------------------------------------------------
func clean_quotes(cq_str string)string{
    return re_quotes.ReplaceAllString(cq_str,"")
}
//----------------------------------------------------------------------------------------------------------------------
func check_1c_database_availability(config_line string,bot_send string){
     matches                        :=  re_1c_check_base.FindAllStringSubmatch(clean_quotes(config_line),-1)
     if(len(matches)                ==  1){
         c1_conn_str                :=  matches[0][1]
         log_it("Database = "+c1_conn_str)
         c1_delay,_                 :=  time.ParseDuration(matches[0][2])
         log_it("Period = "+matches[0][2])
         c1_send_to                 :=  matches[0][3]
         log_it("Notify chat = "+c1_send_to)
         var c1 *ole.IDispatch
         var com *ole.IUnknown
         c1                         =   nil
         com                        =   nil
         url                        :=  bot_send+c1_send_to+"&text="+url.QueryEscape("database "+c1_conn_str+" is not available")
         for {
             log_it("checking "+c1_conn_str)
             Block{
                 Try: func() {
                     ole.CoInitialize(0)
                     com, _         =   oleutil.CreateObject("v83c.Application")
                     c1, _          =   com.QueryInterface(ole.IID_IDispatch)
                     c2             :=  oleutil.MustCallMethod(c1, "Connect", c1_conn_str)
                     if(c2          !=  nil) {
                         //time.Sleep(5 * time.Second)
                         //ole.CoInitialize(0)
                         c3         :=  oleutil.MustCallMethod(c1,"Exit",false)
                         if(c3      !=  nil) {
                             log_it("all ok")
                         }else{
                             log_it("client wasnt closed")
                         }
                     }else{
                         ret,ret2   :=  client.Get(url)
                         fmt.Printf("Caught %v\n", ret, ret2)
                     }
                 },
                 Catch: func(e Exception) {
                     fmt.Printf("Caught %v\n", e)
                 },
                 Finally: func() {
                     c1.Release()
                     c1             =   nil
                     com.Release()
                     com            =   nil
                     ole.CoUninitialize()
                 },
             }.Do()
             time.Sleep(c1_delay)
         }
     }
 }
//----------------------------------------------------------------------------------------------------------------------
func main() {
	conf_file                       :=  strings.Replace(os.Args[0], ".exe", ".config", 1)
    re_str                          :=  regexp.MustCompile("(?m)^(\\w+)?\\:(.+)$")
    if(strings.Contains(os.Args[0],"go_build")){
        conf_file                   =  "c:/repos/gobbix/gobbix.conf"                                                      // debug shit
    }
    log_it("processing "+conf_file)
    bytes, err                      :=  ioutil.ReadFile(conf_file)
    if err                          !=  nil {
        log_it(err.Error())
        os.Exit(-1)
    }
    bot_send                        :=  ""
    for _, record                   :=  range re_str.FindAllSubmatch(bytes,-1) {
        key                         :=  strings.ToUpper(string(record[1]))
        value                       :=  string(record[2])
        switch key {
        case "TELEGRAM_TOKEN":
            log_it("telegram_token is "+clean_quotes(value))
            bot_send                =   "https://api.telegram.org/bot"+clean_quotes(value)+"/sendMessage?chat_id="
            break
        case "WMI":
            log_it(value)
            break
        case "PROXY":
            proxyURL,_              :=  url.Parse(clean_quotes(value))
            client.Transport        =   &http.Transport{
                Proxy: http.ProxyURL(proxyURL),
            }
            log_it("proxy is "+value)
            break
        case "1C":
            if(re_1c_check_base.MatchString(value)){
                log_it("processing rule "+value)
                go check_1c_database_availability(value,bot_send);
            }
            break
        }
    }
    for{
        time.Sleep(1)
    }
}
