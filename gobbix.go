 package main
//----------------------------------------------------------------------------------------------------------------------
 import (
     "bufio"
     "fmt"
     "io/ioutil"
     "log"
     "net/http"
     "net/url"
     "os"
     "os/exec"
     "path/filepath"
     "regexp"
     "strconv"
     "strings"
     "time"
 )
//----------------------------------------------------------------------------------------------------------------------
var (
    re_1c_check_base                        =   regexp.MustCompile("check\\s+availability\\s+(.*)\\s+every\\s+(.*)\\s+notify\\s+(.*)")//регулярка правила проверки базы
    re_quotes                               =   regexp.MustCompile(`\'|\"|\r|\n`)                                   //очистка от кавычек и переносов строки
    re_str                                  =   regexp.MustCompile("(?m)^(\\w+)?\\:(.+)$")                          //разбор строки конфигурации на две части
    re_filename                             =   regexp.MustCompile(`[\\|\/]([^\\|\/]*)$`)                           //получение имени любого файла (последняя часть из пути после слэшем или бэкслэшем), вместе с  / или \
    re_excp                                 =   regexp.MustCompile(`(?m),EXCP,.*Descr=(.+)`)                        //просто EXCP c Descr
    transport                               =   &http.Transport{}                                                       //transport для вызова API telegram
    client                                  =   &http.Client{                                                           //client для вызова API telegram
        Transport                           :   transport,
        Timeout                             :   time.Second * 15,
    }
    c1_client                               =   ""                                                                      //путь к запускаемому клиенту 1С
    working_dir                             =   ""                                                                      //каталог программы
    exit_epf                                =   ""                                                                      //путь к exit.epf в каталоге программы
    tzh_dir                                 =   ""                                                                      //путь к каталогу с логами ТЖ в каталоге программы
    logcfg_xml                              =   ""                                                                      //путь к logcfg.xml в папке bin/conf платформы c1_client
    conf_cfg                                =   ""                                                                      //путь к conf.cfg в папке bin/conf платформы c1_client
    excp_ignore_file                        =   ""
    conf_file                               =   ""
    conf_cfg_value                          =   "DisableUnsafeActionProtection=.*"                                      //отключение запроса на открытие внешних обработок
    bot_send_url                            =   ""                                                                      //путь к команде отправки для Telegram
)
//----------------------------------------------------------------------------------------------------------------------
//находится ли строка в файле с исключениями
//----------------------------------------------------------------------------------------------------------------------
func in_ignore(ii_str string)bool{
    ii_ret                                  :=  false
    if _, ii_err                            :=  os.Stat(excp_ignore_file); os.IsNotExist(ii_err) {
        return false
    }
    ii_file, ii_err                         :=  os.Open(excp_ignore_file)
    if ii_err                               !=  nil {
        log.Fatal(ii_err)
    }
    defer ii_file.Close()
    ii_scanner                              :=  bufio.NewScanner(ii_file)
    for ii_scanner.Scan() {
        ii_line                             :=  ii_scanner.Text()
        ii_str                              =   strings.Trim(ii_str," ")
        ii_rexp                             :=  regexp.MustCompile(ii_line)
        if(ii_rexp.MatchString(ii_str)){
            ii_ret                          =   true
            log_it("ignoring "+ii_str)
            break
        }
    }
    if ii_err                               :=  ii_scanner.Err(); ii_err != nil {
        log.Fatal(ii_err)
    }
    return ii_ret
}
//----------------------------------------------------------------------------------------------------------------------
//очистка строки ТЖ от всякого и ненужного
//----------------------------------------------------------------------------------------------------------------------
func clean_c1_excep(cce_string string)string{
    var (
        cce_guid                            =   regexp.MustCompile(`[\w\d]{8}\-[\w\d]{4}\-[\w\d]{4}\-[\w\d]{4}\-[\w\d]{12}\:`)//GUID
        cce_cpp_no_file                     =   regexp.MustCompile(`'[\w\\]+\.cpp\(\d+\)[\:|\,]`)                   //путь к файлу .CPP
        cce_cpp_file                        =   regexp.MustCompile(`line=\d+\sfile=[\w\:\\]+\.cpp`)                 //второй путь к файлу .cpp
        cce_compact                         =   regexp.MustCompile(`\s{2,}`)                                        //свернуть двойные+ пробелы в один
        cce_trim                            =   regexp.MustCompile(`^\s+|\s+$`)                                     //trim
        cce_ret                             =   ""
    )
    cce_ret                                 =   cce_guid.ReplaceAllString(cce_string,"")
    cce_ret                                 =   cce_cpp_no_file.ReplaceAllString(cce_ret,"")
    cce_ret                                 =   cce_cpp_file.ReplaceAllString(cce_ret,"")
    cce_ret                                 =   cce_compact.ReplaceAllString(cce_ret," ")
    cce_ret                                 =   cce_trim.ReplaceAllString(cce_ret,"")
    return cce_ret
}

//----------------------------------------------------------------------------------------------------------------------
//Склейка висячих строк ТЖ 1С в строки с событиями
//----------------------------------------------------------------------------------------------------------------------
func reparce_1c_records(r1r_in_strings string)string{
    r1r_first_string                        :=  true
    r1r_re_str                              :=  regexp.MustCompile("(?m)^(.+)$")
    r1r_re_tzh_start                        :=  regexp.MustCompile("^\\d\\d:\\d\\d\\.\\d+")
    r1r_ret                                 :=  ""
    r1r_div                                 :=  " "
    for _, r1r_rec                          :=  range r1r_re_str.FindAllStringSubmatch(r1r_in_strings,-1) {
        if(!r1r_first_string){
            if(r1r_re_tzh_start.MatchString(r1r_rec[0])){r1r_div="\r\n"}else{r1r_div=" "}                               //если новая строка, то префикс переноса строки, иначе - пробел
        }
        r1r_ret                             +=  r1r_div+r1r_rec[0]
        r1r_first_string                    =   false
    }
    return r1r_ret+"\r\n"
}
//----------------------------------------------------------------------------------------------------------------------
//пишем сообщение с временем в лог файл и на stdout
//----------------------------------------------------------------------------------------------------------------------
func log_it(msg string) {
    log.SetOutput(os.Stdout)
    log.Print(msg)
    log_file                                :=  os.Args[0]+".log"
    f, err                                  :=  os.OpenFile(log_file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
    if err                                  !=  nil {
        log.Fatalf("error opening file: %v", err)
    }
    defer f.Close()
    log.SetOutput(f)
    log.Println(msg)
}
//----------------------------------------------------------------------------------------------------------------------
//просто очистка от кавычек и символов переноса строки
//----------------------------------------------------------------------------------------------------------------------
func clean_quotes(cq_str string)string{
    return re_quotes.ReplaceAllString(cq_str,"")
}
//----------------------------------------------------------------------------------------------------------------------
//файл ли if_name?
//----------------------------------------------------------------------------------------------------------------------
func is_file(if_name string)bool{
    is_ret                                  :=  false
    if_fdir, if_err                         :=  os.Open(if_name)
    if if_err                               !=  nil {
        log_it(if_err.Error())
        return false
    }
    defer if_fdir.Close()
    if_finfo, if_err                        :=  if_fdir.Stat()
    if if_err                               !=  nil {
        log_it(if_err.Error())
        return false
    }
    switch mode                             :=  if_finfo.Mode(); {
    case mode.IsDir():
        is_ret                              =   false
    case mode.IsRegular():
        is_ret                              =   true
    }
    return is_ret
}
//----------------------------------------------------------------------------------------------------------------------
//func
//----------------------------------------------------------------------------------------------------------------------
func wait_and_kill(wak_pid int,wak_wait time.Duration){
    log_it("waiting "+fmt.Sprint(wak_pid)+" for "+fmt.Sprint(wak_wait))
    time.Sleep(wak_wait)
    wak_proc,_                              :=  os.FindProcess(wak_pid);                                                //нахожу его PID
    if(wak_proc                             !=  nil) {
        log_it("killing pid "+fmt.Sprint(wak_pid))
        wak_proc.Kill()
    }
}
//----------------------------------------------------------------------------------------------------------------------
func check_1c_database_availability(config_line string,c1_local_client string){                                         //проверка доступности базы (строка конфигурации)
    c1_available                            :=  true                                                                    //по умолчанию - база доступна
    matches                                 :=  re_1c_check_base.FindAllStringSubmatch(clean_quotes(config_line),-1) //разбираю строку подключения
    if(len(matches)                         ==  1){
        c1_conn_str                         :=  matches[0][1]                                                           //строка подключения
        c1_delay,_                          :=  time.ParseDuration(matches[0][2])                                       //интервал опроса
        c1_send_to                          :=  matches[0][3]                                                           //чат для отправки сообщений
        c1_start_part                       :=  c1_conn_str+" to "+c1_send_to+" "
        c1_exe_name                         :=  re_filename.FindStringSubmatch(c1_client)[0]                            //имя исполняемого файла 1С
        log_it(c1_start_part+"Database = "+c1_conn_str+"; Period = "+matches[0][2]+"; Notify chat = "+c1_send_to)
        url_not_available                   :=  bot_send_url + c1_send_to + "&text=" +
                                                url.QueryEscape(    "\xF0\x9F\x98\xAD database " +
                                                                    "\xE2\x97\x80"+c1_conn_str+"\xE2\x96\xB6 " +
                                                                    "is NOT available \xF0\x9F\x98\xB2 " +
                                                                    "\xE2\x9A\xA1 \xE2\x9A\xA1 \xE2\x9A\xA1")           //отправка сообщения о недоступности базы
        url_available                       :=  bot_send_url + c1_send_to + "&text=" +
                                                url.QueryEscape(    "\xF0\x9F\x98\xB8 database " +
                                                                    "\xE2\x97\x80"+c1_conn_str+"\xE2\x96\xB6 "+
                                                                    "available \xF0\x9F\x99\x8F "+
                                                                    "\xF0\x9F\x92\xAA \xF0\x9F\x92\xAA \xF0\x9F\x92\xAA")//отправка сообщения о доступности базы
        for {
            log_it(c1_start_part+"checking "+c1_conn_str+" with "+c1_local_client)
            run_1c                          :=  exec.Command(c1_local_client,
                                                "ENTERPRISE",
                                                    "/IBConnectionString"+c1_conn_str,
                                                    "/DisableStartupMessages",
                                                    "/DisableStartupDialogs",
                                                    `/execute`+exit_epf)                                                //команда запуска 1С
            c1da_result                     :=  run_1c.Start()                                                          //запускаю 1С
            if(c1da_result                  !=  nil){
                log_it(c1_start_part+"process start error")
            }
            ret,_                           :=  os.FindProcess(run_1c.Process.Pid);                                     //нахожу его PID
            c1_tzh_dir                      :=  tzh_dir + strings.Replace(c1_exe_name,".exe","",-1) +
                                                "_" + fmt.Sprint(run_1c.Process.Pid)                                    //каталог с ТЖ
            go wait_and_kill(run_1c.Process.Pid,c1_delay/2)                                                             //лекарство для повисших 1С
            ret.Wait()                                                                                                  //жду завершения процесса
            log_it(c1_start_part+"process pid="+fmt.Sprint(run_1c.Process.Pid)+" ended"+c1_exe_name)
            log_it(c1_start_part+"checking "+c1_tzh_dir)
            var c1_files []string
            err                             :=  filepath.Walk(c1_tzh_dir, func(path string, info os.FileInfo, err error) error {//формирую списко файлов ТЖ
                c1_files                    =   append(c1_files, path)
                return nil
            })
            if err                          !=  nil {
                panic(err)
            }
            c1_excp_txt                     :=  ""
            c1_excp_cnt                     :=  0
            for _, c1_file                  :=  range c1_files {                                                        //обойти все файлы ТЖ
                if(!is_file(c1_file)){                                                                                  //и только файлы!
                    continue
                }
                log_it(c1_start_part+"processing "+c1_file)
                c1_b, err                   :=  ioutil.ReadFile(c1_file)                                                //читаю
                if err                      !=  nil {
                    log_it(c1_start_part+err.Error())
                }
                c1_compacted_recs           :=  reparce_1c_records(string(c1_b))
                var c1_errors     []string
                for _, c1_rec               :=  range re_excp.FindAllStringSubmatch(c1_compacted_recs,-1) {
                    c1_err                  :=  clean_quotes(clean_c1_excep(c1_rec[1]))
                    c1_full_err             :=  clean_quotes(c1_rec[0])
                    if(in_ignore(c1_full_err)){
                        continue
                    }
                    found                   :=  false
                    for _,c1_e              :=  range c1_errors{                                                        //не посылать дубли
                        if c1_e             ==  c1_err{
                            found           =   true
                        }
                    }
                    if(!found) {
                        c1_excp_cnt         +=  1
                        c1_excp_txt         += "\r\n " + strconv.Itoa(c1_excp_cnt) + " : " + c1_err
                        c1_errors           =   append(c1_errors,c1_err)
                    }
                    log_it(c1_start_part+"Exception detected:"+c1_rec[0])
                    log_it(c1_start_part+"cleaned = "+c1_err)
                }
                os.Remove(c1_file)                                                                                      //удаляю обработанный файл
            }
            os.Remove(c1_tzh_dir)                                                                                       //удаляю обработанный каталог
            if(c1_excp_cnt>0){                                                                                          //если ошибки были
                if(c1_available) {                                                                                      //и база была доступна
                    c1_available            =   false
                    log_it(c1_start_part+"message to send"+c1_excp_txt)
                    client.Get(url_not_available+url.QueryEscape(c1_excp_txt))                                          //уведомляю
                }
            }else{                                                                                                      //ошибок нет
                if(!c1_available) {                                                                                     //база была недоступной
                    c1_available            =   true
                    client.Get(url_available)                                                                           //уведомляю о доступности
                }
            }
            time.Sleep(c1_delay)
        }
    }
}
//----------------------------------------------------------------------------------------------------------------------
func check_logcfg_xml(clx_1c_exe string){
    clx_rewrite                             :=  false                                                                   //перезаписывать ли logcfg.xml
    clx_1c_filename                         :=  re_filename.FindAllString(clx_1c_exe,-1)[0]                          //имя исполняемого файла 1с
    clx_logcfg_xml                          :=  strings.Replace(clx_1c_exe,clx_1c_filename,"",-1)+"/conf/logcfg.xml"   //путь к logcfg.xml
    log_it("chechking "+clx_logcfg_xml)
    if _, err                               :=  os.Stat(clx_logcfg_xml); os.IsNotExist(err) {                           //проверяю наличие logcdg.xml
        log_it(clx_logcfg_xml+" not exists")
        clx_rewrite                         =   true
    }else{
        bytes, err                          :=  ioutil.ReadFile(clx_logcfg_xml)
        if err                              !=  nil {
            log_it(err.Error())
            os.Exit(-1)
        }
        if(string(bytes)                    !=  logcfg_xml){                                                            //проверяю соответствие logcfg.xml шаблону, если не подошёл
            log_it(clx_logcfg_xml+" incorrect")
            clx_time                        :=  time.Now()
            clx_time_string                 :=  clx_time.Format("2006-01-02 15-04-05")
            os.Rename(clx_logcfg_xml,clx_logcfg_xml+"."+clx_time_string)                                                //то бэкаплю текущий
            clx_rewrite                     =   true
        }
    }
    if(clx_rewrite) {                                                                                                   //[пере]записываю logcfg.xml
        log_it("writing "+clx_logcfg_xml)
        clx_fo, err                         :=  os.Create(clx_logcfg_xml)
        if err                              !=  nil {
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
    working_dir,_                           =   os.Getwd()
    log_it("working_dir = "+working_dir)
    working_dir                             =   strings.Replace(working_dir,"\\","/",-1)
    conf_file                               =   working_dir+"/gobbix.conf"
    log_it("conf_file = "+conf_file)
    excp_ignore_file                        =   working_dir+"/gobbix.ignore"
    log_it("excp_ignore_file = "+excp_ignore_file)
    tzh_dir                                 =   working_dir+"/tzh"
    log_it("tzh_dir = "+tzh_dir)
    logcfg_xml                              =
`<config xmlns="http://v8.1c.ru/v8/tech-log">
    <log location="`+tzh_dir +`" history="1">
        <event>
            <eq property="name" value="EXCP"/>
        </event>
        <property name="all"/>
    </log>
</config>`
    exit_epf                                =   working_dir+"/exit.epf"
    log_it("exit_epf = "+exit_epf)
    //read & process config~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
    log_it("processing "+conf_file)
    bytes, err                              :=  ioutil.ReadFile(conf_file)
    if err                                  !=  nil {
        log_it(err.Error())
        os.Exit(-1)
    }
    for _, record                           :=  range re_str.FindAllSubmatch(bytes,-1) {
        key                                 :=  strings.ToUpper(string(record[1]))
        value                               :=  string(record[2])
        switch key {
        case "TELEGRAM_TOKEN":
            log_it("telegram_token is "+clean_quotes(value))
            bot_send_url                    =   "https://api.telegram.org/bot"+clean_quotes(value)+"/sendMessage?chat_id="
            break
        case "WMI":
            log_it(value)
            break
        case "1C_CLIENT":
            c1_client                       =   strings.Replace(clean_quotes(value),"\\","/",-1)
            break
        case "PROXY":
            proxyURL,_                      :=  url.Parse(clean_quotes(value))
            client.Transport                =   &http.Transport{
                Proxy: http.ProxyURL(proxyURL),
            }
            log_it("proxy is "+value)
            break
        case "1C":
            if(re_1c_check_base.MatchString(value)){
                check_logcfg_xml(c1_client)
                log_it("processing rule "+value)
                go check_1c_database_availability(value,c1_client);
            }
            break
        }
    }
    for{
        time.Sleep(1)
    }
}