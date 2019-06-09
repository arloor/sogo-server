package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/arloor/sogo/utils"
	"log"
	"os"
	"strconv"
)

var configFilePath string = utils.GetWorkDir() + "sogo-server.json" //绝对路径或相对路径
var auths = make([]string, 0)                                       //允许的认证 base64{username:password}

var localAddr string

const pathPrefix = "/target?at="

var redirctAddr = "www.grove.co.uk:80"                      //用于替换
const fakeHost = "qtgwuehaoisdhuaishdaisuhdasiuhlassjd.com" //如果不是这个host，则到混淆网站

func printUsage() {
	fmt.Println("运行方式： sogo-server [-c  configFilePath ]  若不使用 -c指定配置文件，则默认使用" + configFilePath)
}

type Info struct {
	ServerPort  int
	RedirctAddr string
	Dev         bool
	Users       []User
}

type User struct {
	UserName string
	Password string
}

var Config = Info{
	80,
	"www.grove.co.uk:80",
	true,
	[]User{{"a", "b"}},
}

func init() {

	printUsage()

	if len(os.Args) == 3 && os.Args[1] == "-c" {
		configFilePath = os.Args[2]
	}

	//log.SetOutput(os.Stdout)
	logFile, _ := os.OpenFile(utils.GetWorkDir()+"log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	log.SetOutput(logFile)
	log.SetFlags(log.Lshortfile | log.Flags())
	configinit()
	log.Println("配置信息为：", Config)

	if !Config.Dev {
		log.Println("已启动sogo客户端，请在sogoserver_" + strconv.Itoa(Config.ServerPort) + ".log查看详细日志")
		f, _ := os.OpenFile("sogoserver_"+strconv.Itoa(Config.ServerPort)+".log", os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0755)
		log.SetOutput(f)
	}

	localAddr = ":" + strconv.Itoa(Config.ServerPort)
	redirctAddr = Config.RedirctAddr
	for i := 0; i < len(Config.Users); i++ {
		user := Config.Users[i]
		auths = append(auths, "Basic "+base64.StdEncoding.EncodeToString([]byte(user.UserName+":"+user.Password)))
	}
}

func configinit() {
	configFile, err := os.Open(configFilePath)
	defer configFile.Close()
	if err != nil {
		log.Println("Error", "打开"+configFilePath+"失败，使用默认配置", err)
		return
	}
	bufSize := 1024
	buf := make([]byte, bufSize)
	for {
		total := 0
		n, err := configFile.Read(buf)
		total += n
		if err != nil {
			log.Println("Error", "读取"+configFilePath+"失败，使用默认配置", err)
			return
		} else if n < bufSize {
			log.Println("OK", "读取"+configFilePath+"成功")
			buf = buf[:total]
			break
		}

	}
	err = json.Unmarshal(buf, &Config)
	if err != nil {
		log.Println("Error", "读取"+configFilePath+"失败，使用默认配置", err)
		return
	}

}

func (configInfo Info) String() string {
	str, _ := configInfo.ToJSONString()
	return str
}

//implement JSONObject
func (configInfo Info) ToJSONString() (str string, error error) {
	b, err := json.Marshal(configInfo)
	if err != nil {
		return "", err
	} else {
		return string(b), nil
	}
}
