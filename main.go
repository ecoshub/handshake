package main

import (
	"fmt"
	"net"
	"strings"
	"os"
    "os/signal"
    "os/user"
    "syscall"
)

var port string = "9998"
var mainListen int = 10000
var broadcastIP string = GetBroadcastIP().String()
var myIP string = GetInterfaceIP().String()
var myIPB []byte = []byte(myIP)
var broadcastListenIP string = "0.0.0.0"
var loopControl int = 100
var receiveControl bool = true
var IPList []string = make([]string,0,1024)
var msgList []string = make([]string,0,1024)
var UsernameList []string = make([]string,0,1024)
var onlineCount int = 0
var myUsername string = getUsername()
var myUsernameB []byte = []byte(myUsername)
var msgOn []byte = []byte("online")
var msgOff []byte = []byte("offline")
var msgBusy []byte = []byte("busy")
var udpRepeat int = 5


func main(){
	Start()
}

func Start(){

	sigs := make(chan os.Signal, 1)
    done := make(chan bool, 1)
    signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	data := concatByteArray(" ", msgOn, myUsernameB)
    sendPack(broadcastIP, port, data)
	receiveChan := make(chan string, 1)
    go func() {
        <- sigs
		data := concatByteArray(" ", msgOff, myUsernameB)
        offlineFunc(broadcastIP, port, data, done)
        <- done
      	os.Exit(0)
    }()
	for receiveControl {
		go receive(broadcastListenIP, port, receiveChan)
		tempPack := <- receiveChan
		tempMsg, tempIP, tempUsername := parsePack(tempPack)
		if !hasThis(IPList, tempIP) && tempIP != myIP && tempMsg == string(msgOn){
			IPList = append(IPList, tempIP)
			msgList = append(msgList, tempMsg)
			UsernameList = append(UsernameList, tempUsername)
			onlineCount++
			data := concatByteArray(" ", msgOn, myUsernameB)
    		sendPack(broadcastIP, port, data)
			msg := fmt.Sprintf("stat:%v:%v:%v",tempMsg, tempIP, tempUsername)
			fmt.Println(myIP, mainListen, msg)
		}
		if hasThis(IPList, tempIP) && tempIP != myIP && tempMsg == string(msgOff){
			IPList = removeFromList(IPList, tempIP)
			msgList = removeFromList(msgList, string(msgOn))
			UsernameList = removeFromList(UsernameList, tempUsername)
			msg := fmt.Sprintf("stat:%v:%v:%v",tempMsg, tempIP, tempUsername)
			fmt.Println(myIP, mainListen, msg)
		}
	}
}

func receive(ip, port string, ch chan<- string){
	buff := make([]byte, 1024)
    pack, err := net.ListenPacket("udp", ip + ":" + port)
    if err != nil {
        fmt.Println("Connection Fail", err)
        panic(err)
    }
    n, addr, err := pack.ReadFrom(buff)
    if err != nil {
        fmt.Println("Read Error", err)
        panic(err)
    }
    ipandport := strings.Split(addr.String(), ":")
    remoteIP := ipandport[0]
    buff = buff[:n]
    ch <- string(buff) + " " + remoteIP
    pack.Close()
}

func sendPack(ip, port string, data []byte){
	sendValidationChan := make(chan int, 1)
	valid := 0
	count := 0
	for valid != 1 {
		for i := 0 ; i < udpRepeat ; i++ {
			go send(broadcastIP, port, data , sendValidationChan)
		}
		valid = <- sendValidationChan
		count++
		if count > loopControl {
			fmt.Println("Something is wrong can't send any signal!")
			return
		}
	}

}

func send(ip, port string, data []byte, ch chan<- int){
    conn, err := net.Dial("udp", ip + ":" + port)
       if err != nil {
        fmt.Println("Connection Fail")
        panic(err)
    	ch <- 0
    }
    conn.Write(data)
    conn.Close()
    ch <- 1
}

func offlineFunc(ip, port string, data []byte, ch chan<- bool){
    conn, err := net.Dial("udp", ip + ":" + port)
       if err != nil {
        fmt.Println("Connection Fail")
    }
    conn.Write(data)
    conn.Close()
    ch <- true
}

func GetInterfaceIP() net.IP{
    ins, _ := net.Interfaces()
    inslen := len(ins)
    myAddr := ""
    for i := 0 ; i < inslen ; i++ {
        if ins[i].Flags &  net.FlagLoopback != net.FlagLoopback && ins[i].Flags & net.FlagUp == net.FlagUp{
            addr, _ := ins[i].Addrs()
            if addr != nil {
                for _,ad := range addr{
                    if strings.Contains(ad.String(), "."){
                        myAddr = ad.String()
                        break
                    }
                }
                ip, _, _ := net.ParseCIDR(myAddr)
                return ip
            }
        }
    }
    fmt.Println("Interface IP resolve error in func GetInterfaceIP()")
    return net.ParseIP("0.0.0.0")
}

func GetBroadcastIP() net.IP{
	IP := GetInterfaceIP()
	IP[len(IP) - 1] = 255
	return IP
}

func hasThis(list []string, el string) bool {
	for _, v := range list {
		if v == el {
			return true
		}
	}
	return false
}

func parsePack(pack string) (msg string, IP string, username string){
	tokens := strings.Split(pack, " ")
	msg = tokens[0]
	username = tokens[1]
	IP = tokens[2]
	return
}

func removeFromList(list []string, el string) []string{
	lenl := len(list)
	if lenl < 2 {
		return nil
	}
	newList := make([]string,lenl - 1,1024)
	count := 0
	for _, v := range list {
		if v != el  {
			newList[count] = v
			count++
		}
	}
	return newList
}

func getUsername() string{
	user, err := user.Current()
	if err != nil {
		return "unknown"
	}else{
		return user.Username
	}
}

func concatByteArray(sep string, arr ...[]byte) []byte {
	newArr := make([]byte,0,1024)
	lena := len(arr)
	sepB := []byte(sep)
	for i, v := range arr {
		newArr = append(newArr, v...)
		if i != lena - 1 {
			newArr = append(newArr, sepB...)
		} 
	}
	return newArr
}