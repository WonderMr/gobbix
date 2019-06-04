package main

import (
    "log"
    "regexp"
)
var (s  = `12:03.652002-0,EXCP,0,process=1cv8c,OSThread=9856,setTerminateHandler=setTerminateHandler
12:20.668005-0,EXCP,2,process=1cv8c,OSThread=15680,Exception=580392e6-ba49-4280-ac67-fcd6f2180121,Descr='src\vrscore\src\vresourcesessionimpl.cpp(529):
580392e6-ba49-4280-ac67-fcd6f2180121: HTTP: Forbidden
Ошибка при выполнении запроса POST к ресурсу /e1cib/extmd/processing:
f6f167a0-dcc9-49ad-8f8e-2c9d9904e4fe: Ошибка подключения внешних метаданных
Отсутствуют права на интерактивную загрузку внешних обработок
f6f167a0-dcc9-49ad-8f8e-2c9d9904e4fe: Нарушение прав доступа!'
12:20.668006-0,EXCP,2,process=1cv8c,OSThread=15680,Exception=babb3b7c-1e8c-4d44-9eba-e1cd06dba9eb,Descr='src\mngui\src\exceptionwriteruiimpl.cpp(436), shown to the user:
babb3b7c-1e8c-4d44-9eba-e1cd06dba9eb: Ошибка загрузки документа.
f6f167a0-dcc9-49ad-8f8e-2c9d9904e4fe: Ошибка подключения внешних метаданных
Отсутствуют права на интерактивную загрузку внешних обработок
f6f167a0-dcc9-49ad-8f8e-2c9d9904e4fe: Нарушение прав доступа!'`
)

func main() {
    log.Print(reparce_1c_records(s))
}

func reparce_1c_records(r1r_in_strings string)string{
    r1r_first_string                :=  true
    r1r_re_str                      :=  regexp.MustCompile("(?m)^(.+)$")
    r1r_re_tzh_start                :=  regexp.MustCompile("^\\d\\d:\\d\\d\\.\\d+")
    r1r_ret                         :=  ""
    r1r_div                         :=  " "
    for _, r1r_rec                  :=  range r1r_re_str.FindAllStringSubmatch(r1r_in_strings,-1) {
        if(!r1r_first_string){
            if(r1r_re_tzh_start.MatchString(r1r_rec[0])){r1r_div="\r\n"}else{r1r_div=" "}
        }
        r1r_ret                     +=  r1r_div+r1r_rec[0]
        r1r_first_string            =   false
    }
    return r1r_ret+"\r\n"
}


//perl -ne '{
//    $_=~s/\r*\n/ /g;                        #\r*\n в пробел
//    $_="\r\n".$_ if(/^\d\d:\d\d\.\d+/);     #если строка начинается с заголовка записи 1с строка начнётся с переноса
//    print $_;}'