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

const (
	PORT string = "9998"
	BROADCASTLISTENIP string = "0.0.0.0"
	LOOPCONTROL int = 100
	UDPREPEAT int = 5
)

var (
	// local-ip broadcast-ip username
	myUsername string = getUsername()
	myIP string = GetInterfaceIP().String()
	broadcastIP string = GetBroadcastIP().String()

	// byte buffer
	msgOn []byte = []byte("online")
	msgOff []byte = []byte("offline")
	msgBusy []byte = []byte("busy")
	myIPB []byte = []byte(myIP)
	myUsernameB []byte = []byte(myUsername)

	// buffers
	IPList []string = make([]string,0,1024)
	msgList []string = make([]string,0,1024)
	UsernameList []string = make([]string,0,1024)
	
	// counter
	onlineCount int = 0

	// inf. loop control
	receiveControl bool = true
)

func main(){
	activity()
}

func activity(){	
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	data := concatByteArray(" ", msgOn, myUsernameB)
	sendPack(broadcastIP, PORT, data)
	receiveChan := make(chan string, 1)
	go func() {
		<- sigs
		receiveControl = false
		onClose(done)
		<- done
		os.Exit(0)
	}()
	for receiveControl {
		go receive(BROADCASTLISTENIP, PORT, receiveChan)
		tempPack := <- receiveChan
		tempMsg, tempIP, tempUsername := parsePack(tempPack)
		if !hasThis(IPList, tempIP) && tempIP != myIP && tempMsg == string(msgOn){
			IPList = append(IPList, tempIP)
			msgList = append(msgList, tempMsg)
			UsernameList = append(UsernameList, tempUsername)
			onlineCount++
			msg := fmt.Sprintf(`{"stat":"%v","ip":"%v","username":"%v"}`,tempMsg, tempIP, tempUsername)
			fmt.Println(msg)
			data := concatByteArray(" ", msgOn, myUsernameB)
			sendPack(broadcastIP, PORT, data)
		}
		if hasThis(IPList, tempIP) && tempIP != myIP && tempMsg == string(msgOff){
			IPList = removeFromList(IPList, tempIP)
			msgList = removeFromList(msgList, string(msgOn))
			UsernameList = removeFromList(UsernameList, tempUsername)
			msg := fmt.Sprintf(`{"stat":"%v","ip":"%v","username":"%v"}`,tempMsg, tempIP, tempUsername)
			fmt.Println(msg)
		}
	}
}


func receive(ip, port string, ch chan<- string){
	buff := make([]byte, 1024)
	pack, err := net.ListenPacket("udp", ip + ":" + port)
	if err != nil {
		fmt.Println("Connection Fail", err)
	}
	n, addr, err := pack.ReadFrom(buff)
	if err != nil {
		fmt.Println("Read Error", err)
	}else{
		defer pack.Close()
		ipandport := strings.Split(addr.String(), ":")
		remoteIP := ipandport[0]
		buff = buff[:n]
		ch <- string(buff) + " " + remoteIP
	}
}

func sendPack(ip, port string, data []byte){
	sendValidationChan := make(chan int, 1)
	valid := 0
	count := 0
	for valid != 1 {
		for i := 0 ; i < UDPREPEAT ; i++ {
			go send(broadcastIP, port, data , sendValidationChan)
		}
		valid = <- sendValidationChan
		count++
		if count > LOOPCONTROL {
			fmt.Println("Something is wrong can't send any signal!")
			return
		}
	}

}

func send(ip, port string, data []byte, ch chan<- int){
	conn, err := net.Dial("udp", ip + ":" + port)
	   if err != nil {
		fmt.Println("Connection Fail")
		ch <- 0
	}else{
		defer conn.Close()
		conn.Write(data)
		ch <- 1
	}
}

func onClose(ch chan<- bool){
	sendValidationChan := make(chan int, 1)
	data := concatByteArray(" ", msgOff, myUsernameB)
	valid := 0
	count := 0
	for valid != 1 {
		for i := 0 ; i < UDPREPEAT ; i++ {
			go offlineFunc(broadcastIP, PORT, data , sendValidationChan)
		}
		valid = <- sendValidationChan
		count++
		if count > LOOPCONTROL {
			fmt.Println("Something is wrong can't send any signal!")
			break
		}
	}
	ch <- true
}


func offlineFunc(ip, port string, data []byte, ch chan<- int){
	conn, err := net.Dial("udp", ip + ":" + port)
	   if err != nil {
		fmt.Println("Connection Fail")
		ch <- 0
	}else{
		defer conn.Close()
		conn.Write(data)
		ch <- 1
	}
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
	return net.ParseIP(BROADCASTLISTENIP)
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