 package main
//----------------------------------------------------------------------------------------------------------------------
 import (
     "fmt"
     "io/ioutil"
     "log"
     "net/http"
     "net/url"
     "os"
     "os/exec"
     "path/filepath"
     "regexp"
     "strings"
     "time"
 )
//----------------------------------------------------------------------------------------------------------------------
var (
    re_1c_check_base                =   regexp.MustCompile("check\\savailability\\s(.*)\\severy\\s(.*)\\snotify\\s(.*)")
    re_quotes                       =   regexp.MustCompile(`\'|\"|\r|\n`)
    re_str                          =   regexp.MustCompile("(?m)^(\\w+)?\\:(.+)$")
    re_filename                     =   regexp.MustCompile(`[\\|\/]([^\\|\/]*)$`)
    re_excp_descr                   =   regexp.MustCompile(`\:(.*)'$`)
    re_excp                         =   regexp.MustCompile(`,EXCP,`)
    transport                       =   &http.Transport{}
    client                          =   &http.Client{
        Transport                   :   transport,
        Timeout                     :   time.Second * 15,
    }
    c1_client                       =   ""
    working_dir                     =   ""
    exit_epf                        =   ""
    tzh_dir                         =   ""
    logcfg_xml                      =   ""
    conf_cfg                        =   ""
    conf_cfg_value                  =   "DisableUnsafeActionProtection=.*"
    bot_send_url                    =   ""
)
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
func check_1c_database_availability(config_line string){
    c1_available                    :=  true
    matches                         :=  re_1c_check_base.FindAllStringSubmatch(clean_quotes(config_line),-1)
    if(len(matches)                 ==  1){
        c1_conn_str                 :=  matches[0][1]
        c1_delay,_                  :=  time.ParseDuration(matches[0][2])
        c1_send_to                  :=  matches[0][3]
        run_1c                      :=  exec.Command(c1_client,"ENTERPRISE","/IBConnectionString"+c1_conn_str,"/DisableStartupMessages","/DisableStartupDialogs",`/execute`+exit_epf)
        c1_exe_name                 :=  re_filename.FindStringSubmatch(c1_client)[0]
        log_it("Database = "+c1_conn_str+"; Period = "+matches[0][2]+"; Notify chat = "+c1_send_to)
        url_not_available           :=  bot_send_url+c1_send_to+"&text="+url.QueryEscape("\xF0\x9F\x98\xAD database \xE2\x97\x80"+c1_conn_str+"\xE2\x96\xB6 is NOT available \xF0\x9F\x98\xB2 \xE2\x9A\xA1 \xE2\x9A\xA1 \xE2\x9A\xA1")
        url_available               :=  bot_send_url+c1_send_to+"&text="+url.QueryEscape("\xF0\x9F\x98\xB8 database \xE2\x97\x80"+c1_conn_str+"\xE2\x96\xB6 available \xF0\x9F\x99\x8F \xF0\x9F\x92\xAA \xF0\x9F\x92\xAA \xF0\x9F\x92\xAA")
        for {
            log_it("checking "+c1_conn_str)
            c1da_result             :=  run_1c.Start()
            if(c1da_result          !=  nil){
                log_it("process start error")
            }
            ret,_                   :=  os.FindProcess(run_1c.Process.Pid);
            c1_tzh_dir              :=  tzh_dir+strings.Replace(c1_exe_name,".exe","",-1)+"_"+fmt.Sprint(run_1c.Process.Pid)
            ret.Wait()
            log_it("process pid="+fmt.Sprint(run_1c.Process.Pid)+" ended"+c1_exe_name)
            log_it("checking "+c1_tzh_dir)
            var cl_files []string
            cl_is_exception         :=  false
            err                     :=  filepath.Walk(c1_tzh_dir, func(path string, info os.FileInfo, err error) error {
                cl_files            =   append(cl_files, path)
                return nil
            })
            if err                  !=  nil {
                panic(err)
            }
            for _, cl_file          := range cl_files {
                log_it("processing "+cl_file)
                cl_b, err           :=  ioutil.ReadFile(cl_file)
                if err              !=  nil {
                    log_it(err.Error())
                }
                if(re_excp.Match(cl_b)){
                    cl_is_exception =   true
                    log_it("Exception detected:"+string(cl_b))
                }
                os.Remove(cl_file)
            }
            os.Remove(c1_tzh_dir)
            if(cl_is_exception){
                if(c1_available) {
                    c1_available    =   false
                    client.Get(url_not_available)
                }
            }else{
                if(!c1_available) {
                    c1_available    =   true
                    client.Get(url_available)
                }
            }
            time.Sleep(c1_delay)
        }
    }
}
//----------------------------------------------------------------------------------------------------------------------
func check_logcfg_xml(clx_1c_exe string){
    clx_rewrite                     :=  false                                                                           //перезаписывать ли logcfg.xml
    clx_1c_filename                 :=  re_filename.FindAllString(clx_1c_exe,-1)[0]                                 //имя исполняемого файла 1с
    clx_logcfg_xml                  :=  strings.Replace(clx_1c_exe,clx_1c_filename,"",-1)+"/conf/logcfg.xml"   //путь к logcfg.xml
    log_it("chechking "+clx_logcfg_xml)
    if _, err                       :=  os.Stat(clx_logcfg_xml); os.IsNotExist(err) {
        log_it(clx_logcfg_xml+" not exists")
        clx_rewrite                 =   true
    }else{
        bytes, err                  :=  ioutil.ReadFile(clx_logcfg_xml)
        if err                      !=  nil {
            log_it(err.Error())
            os.Exit(-1)
        }
        if(string(bytes)            !=  logcfg_xml){
            log_it(clx_logcfg_xml+" incorrect")
            clx_time                :=  time.Now()
            clx_time_string         :=  clx_time.Format("2006-01-02 15-04-05")
            os.Rename(clx_logcfg_xml,clx_logcfg_xml+"."+clx_time_string)
            clx_rewrite             =   true
        }
    }
    if(clx_rewrite) {
        log_it("writing "+clx_logcfg_xml)
        clx_fo, err                 :=  os.Create(clx_logcfg_xml)
        if err                      !=  nil {
            log_it(err.Error())
            os.Exit(-1)
        }
        clx_fo.WriteString(logcfg_xml)
        clx_fo.Close()
    }
}
//----------------------------------------------------------------------------------------------------------------------
func main() {
    //set global vars~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
    working_dir,_                   =   os.Getwd()
    working_dir                     =   strings.Replace(working_dir,"\\","/",-1)
    conf_file                       :=  working_dir+"/gobbix.conf"
    tzh_dir                         =   working_dir+"/tzh"
    logcfg_xml                      =
`<config xmlns="http://v8.1c.ru/v8/tech-log">
    <log location="`+tzh_dir +`" history="1">
        <event>
            <eq property="name" value="EXCP"/>
        </event>
        <property name="all"/>
    </log>
</config>`
    exit_epf                        =   working_dir+"/exit.epf"
    //read & process config~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
    log_it("processing "+conf_file)
    bytes, err                      :=  ioutil.ReadFile(conf_file)
    if err                          !=  nil {
        log_it(err.Error())
        os.Exit(-1)
    }
    for _, record                   :=  range re_str.FindAllSubmatch(bytes,-1) {
        key                         :=  strings.ToUpper(string(record[1]))
        value                       :=  string(record[2])
        switch key {
        case "TELEGRAM_TOKEN":
            log_it("telegram_token is "+clean_quotes(value))
            bot_send_url            =   "https://api.telegram.org/bot"+clean_quotes(value)+"/sendMessage?chat_id="
            break
        case "WMI":
            log_it(value)
            break
        case "1C_CLIENT":
            c1_client               =   strings.Replace(clean_quotes(value),"\\","/",-1)
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
                check_logcfg_xml(c1_client)
                log_it("processing rule "+value)
                go check_1c_database_availability(value);
            }
            break
        }
    }
    for{
        time.Sleep(1)
    }
}
