package main
//----------------------------------------------------------------------------------------------------------------------
import (
    "io/ioutil"
    "log"
    "os"
    "regexp"
    "strings"
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
func main() {
	conf_file                       :=  strings.Replace(os.Args[0], ".exe", ".config", 1)
    re_str                          :=  regexp.MustCompile("(?m)^(\\w+)?\\:(.+)$")

    if(strings.Contains(os.Args[0],"__go_build")){
        conf_file                   =  "c:/dev/gobbix/gobbix.conf"                                                      // debug shit
    }
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
            log_it(value)
            break
        case "WMI":
            log_it(value)
            break
        case "1C":
            log_it(value)
            break
        }
    }
}
