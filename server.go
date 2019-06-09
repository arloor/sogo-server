package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/arloor/sogo-server/mio"
	"github.com/arloor/sogo/utils"
	"log"
	"net"
	"strconv"
	"strings"
)

func main() {

	ln, err := net.Listen("tcp", localAddr)
	if err != nil {
		fmt.Println("监听", localAddr, "失败 ", err)
		return
	}
	defer ln.Close()
	fmt.Println("成功监听 ", ln.Addr(), "日志在", utils.GetWorkDir()+"log.txt")
	for {
		c, err := ln.Accept()
		if err != nil {
			log.Println("接受连接失败 ", err)
		} else {
			go handleClientConnnection(c)
		}
	}
}

//头
//POST /target?goog:80 HTTP/1.1
//Host: qtgwuehaoisdhuaishdaisuhdasiuhlassjd.com
//Accept: */*
//Content-Type: text/plain
//accept-encoding: gzip, deflate
//content-length: 447
//
//GET / HTTP/1.1  （负载）
////Host: aaaaaaa.com
////Connection: keep-alive
////Cache-Control: max-age=0
////Upgrade-Insecure-Requests: 1
////User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.86 Safari/537.36
////Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3
////Accept-Encoding: gzip, deflate
////Accept-Language: zh,en;q=0.9,zh-CN;q=0.8

func handleClientConnnection(conn net.Conn) {
	var serverConn net.Conn = nil
	defer conn.Close()
	//将请求写到标准输出
	//bufio.NewReader(conn).WriteTo(os.Stdout)
	redundancy := make([]byte, 0)
	for {
		payload, redundancyRetain, target, readErr := read(conn, redundancy)
		redundancy = redundancyRetain
		if readErr != nil {
			//log.Print("readErr", readErr)
			return
		} else if target != "" {
			//fmt.Print(string(payload))
			//fmt.Print(target)
			//fmt.Println("<<end")
			if serverConn == nil {
				newConn, dialErr := net.Dial("tcp", target)
				log.Println(target, "---FROM---", conn.RemoteAddr())
				if dialErr != nil {
					log.Println("dialErr", target, dialErr)
					return
				} else {
					serverConn = newConn
					go handleServerConn(serverConn, conn)
				}
			}
			mio.Simple(&payload, len(payload))
			writeErr := mio.WriteAll(serverConn, payload)
			if writeErr != nil {
				serverConn.Close()
				return
			}
		} else {
			log.Println("已转发到混淆网站", redirctAddr, "---FROM---", conn.RemoteAddr())
		}

	}

}

func handleServerConn(serverConn net.Conn, clientConn net.Conn) {
	//bufio.NewReader(serverConn).WriteTo(clientCon)//好像这句话是一样地作用
	buf := make([]byte, 8196)
	for {
		num, readErr := serverConn.Read(buf)
		if readErr != nil {
			//log.Print("readErr ", readErr, serverConn.RemoteAddr())
			clientConn.Close()
			serverConn.Close()
			return
		}

		writeErr := mio.WriteAll(clientConn, mio.AppendHttpResponsePrefix(buf[:num]))
		if writeErr != nil {
			//log.Print("writeErr ", writeErr)
			clientConn.Close()
			serverConn.Close()
			return
		}
		buf = buf[0:]
	}
}

func read(clientConn net.Conn, redundancy []byte) (payload, redundancyRetain []byte, target string, readErr error) {
	contentlength := -1
	prefixAll := false
	prefix := make([]byte, 0)
	//redundancy:=make([]byte,0)
	//payload:=make([]byte,0)
	buf := make([]byte, 512)

	//method := ""
	path := ""
	//version := ""
	target = ""
	host := ""
	auth := ""

	var hunxiaoConn net.Conn = nil
	for {

		num := 0
		var readErr error = nil
		if len(redundancy) > 0 {
			num += len(redundancy)
			buf = redundancy
			redundancy = redundancy[0:0]
		} else {
			num, readErr = clientConn.Read(buf)
		}

		if readErr != nil {
			//log.Println("readErr", readErr)
			return nil, nil, "", readErr
		} else if num <= 0 {
			return nil, nil, "", errors.New("读到<=0字节，未预期地情况")
		} else {
			if !prefixAll { //追加到前缀
				prefix = append(prefix, buf[:num]...)
				if index := strings.Index(string(prefix), "\r\n\r\n"); index >= 0 {
					if index+4 < len(prefix) {
						payload = append(payload, prefix[index+4:]...)
					}
					prefix = prefix[:index]
					prefixAll = true
					//分析头部
					headrs := strings.Split(string(prefix), "\r\n")

					requestline := headrs[0]
					parts := strings.Split(requestline, " ")
					if len(parts) != 3 {
						fmt.Println(requestline)
						if hunxiaoConn != nil {
							hunxiaoConn.Close()
						}
						return nil, nil, "", errors.New("不是以GET xx HTTP/1.1这种开头，说明上个请求有问题。")
					}
					//method = parts[0]
					path = parts[1]
					//version = parts[2]
					if strings.HasPrefix(path, pathPrefix) {
						targettemp := path[len(pathPrefix):]
						decodeBytes, decodeErr := base64.NewEncoding("abcdefghijpqrzABCKLMNOkDEFGHIJl345678mnoPQRSTUVstuvwxyWXYZ0129+/").DecodeString(targettemp)
						if decodeErr == nil {
							target = string(decodeBytes)
						}
					}

					var headmap = make(map[string]string)
					for i := 1; i < len(headrs); i++ {
						headsplit := strings.Split(headrs[i], ": ")
						if len(headsplit) == 2 {
							headmap[headsplit[0]] = headsplit[1]
						}
					}
					host = headmap["Host"]
					//log.Println("[前缀]", method, host, target, version)
					if headmap["content-length"] == "" {
						contentlength = 0
					} else {
						contentlength, _ = strconv.Atoi(headmap["content-length"])
					}
					auth = headmap["Authorization"]
				}
			} else { //追加到payload
				payload = append(payload, buf[:num]...)
			}
		}
		buf = buf[0:]
		if contentlength != -1 && contentlength < len(payload) { //这说明读多了，要把多的放到redundancy
			redundancy = append(redundancy, payload[contentlength:]...)
			payload = payload[:contentlength]
		}
		if contentlength == len(payload) {
			hostTemp := strings.Split(host, ":")[0]
			//if strings.Contains(hunxiaoHost, hostTemp)||!strings.Contains(hostTemp,"qtgwuehaoisdhuaishdaisuhdasiuhlassjd.com") {
			if !strings.Contains(hostTemp, fakeHost) {
				//todo:判断，如果是访问混淆网站，就转到混淆设置地网站
				if hunxiaoConn == nil {
					newConn, err := net.Dial("tcp", redirctAddr)
					if err != nil {
						return nil, nil, "", errors.New("连接到混淆网站失败，混淆网站应该挂了")
					} else {
						hunxiaoConn = newConn
						go handleHunxiaoConn(hunxiaoConn, clientConn)
					}
				}
				//更换host
				prefix = []byte(strings.Replace(string(prefix), host, redirctAddr, -1))
				writeHunxiaoErr := mio.WriteAll(hunxiaoConn, append(append(prefix, []byte("\r\n\r\n")...), payload...))
				if writeHunxiaoErr != nil {
					hunxiaoConn.Close()
				}
				return nil, redundancy, "", writeHunxiaoErr
			}
			//如果确实是代理，并且auth字段正确，则返回
			for _, cell := range auths {
				if cell == auth {
					return payload, redundancy, target, nil
				}
			}
			return payload, redundancy, target, errors.New("错误的认证信息")
		}
	}
}

func handleHunxiaoConn(hunxiaoConn, clientCon net.Conn) {
	//bufio.NewReader(hunxiaoConn).WriteTo(clientCon)//好像这句话是一样地作用
	buf := make([]byte, 2048)
	for {
		num, readErr := hunxiaoConn.Read(buf)
		if readErr != nil {
			//log.Print("readErr ", readErr, hunxiaoConn.RemoteAddr())
			clientCon.Close()
			hunxiaoConn.Close()
			return
		}
		writeErr := mio.WriteAll(clientCon, buf[:num])
		if writeErr != nil {
			//log.Print("writeErr ", writeErr)
			clientCon.Close()
			hunxiaoConn.Close()
			return
		}
		buf = buf[0:]
	}
}
