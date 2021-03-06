package chunk

import (
	"../include/sfs" 
	"os"
	"rpc"
	"log"
	"net"
//	"fmt"
	"container/vector"
	//"syscall"
	"time"
	"exec"
)

type Server int

const CHUNK_TABLE_SIZE = 1024*1024*1024 / sfs.CHUNK_SIZE
const STATUS_CMD = "../stats.sh"
const STATUS_ARGS = ""
const STATUS_LEN = 17

var chunkTable = map[uint64] sfs.Chunk {}
var capacity uint64
var addedChunks vector.Vector
var chunkServerID uint64

func Init(masterAddress string) {

	var args sfs.ChunkBirthArgs
	var ret sfs.ChunkBirthReturn 

	args.Capacity = 5
	host,_ := os.Hostname()
	_,iparray,_ := net.LookupHost(host)
	tcpAddr,_ := net.ResolveTCPAddr(iparray[0] + ":1337")
	args.ChunkServerIP = *tcpAddr
	log.Println(args.ChunkServerIP)

	master, err := rpc.Dial("tcp", masterAddress + ":1338")
	if err != nil {
		log.Fatal("chunk dial error:", err)
	}

	err = master.Call("Master.BirthChunk", &args, &ret)
	if err != nil {
		log.Fatal("chunk call error: ", err)
	}
	chunkServerID = ret.ChunkServerID
}

func (t *Server) Read(args *sfs.ReadArgs, ret *sfs.ReadReturn) os.Error {
	data,present := chunkTable[args.ChunkID]
	if !present{
		ret.Status = -1
		return nil
	}
	log.Println("chunk: Reading from chunk ", args.ChunkID)

	ret.Data.Data = data.Data

	ret.Status = 0
	return nil	
}

func (t *Server) Write(args *sfs.WriteArgs, ret *sfs.WriteReturn) os.Error {

	log.Println("chunk: Writing to chunk ", args.Info.ChunkID)
	data,present := chunkTable[args.Info.ChunkID]
	if !present{
		addedChunks.Push(args.Info)
		capacity --
	}

	data.Data = args.Data.Data
	chunkTable[args.Info.ChunkID] = data

	tempServ := args.Info.Servers[0]
	var inRet sfs.WriteReturn

	for {
		if len(args.Info.Servers) < 2 {
			break
		}
		

		args.Info.Servers = args.Info.Servers[1:len(args.Info.Servers)]

		client, err := rpc.Dial("tcp", args.Info.Servers[0].String())
		if err != nil {
			log.Printf("chunk: dialing:", err)
			continue
		}

		err = client.Call("Server.Write", &args, &inRet)
		if err != nil {
			log.Fatal("chunk: server error: ", err)
		}
		break
	}

	ret.Info.Servers = make([]net.TCPAddr, len(inRet.Info.Servers) + 1)
	
	for i:=0;i<len(ret.Info.Servers);i++ {
		if(i < len(inRet.Info.Servers)) {
			ret.Info.Servers[i] = inRet.Info.Servers[i]		
		} else {
			ret.Info.Servers[i] = tempServ		
		}
	}
	
	return nil	
}

/*func (t *Server) Get(args *sfs.PingArgs, ret *sfs.PingReturn) os.Error {
	return nil
}*/

func LogStats(){
	//first, open the daily log file
	
	//log.Println("We're alive")
	host,_ := os.Hostname()
	//current := time.Seconds()
	filename:= host
	logFile, err := os.Open(filename, os.O_CREAT | os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal("chunk: unable to init logging")
		log.Println("chunk fails opening file")
	}
	//now, enter the happy place of logging all our fun stuff
	//var info syscall.Sysinfo_t
	args := make([]string, 1)
	var result []byte
	result = make([]byte, STATUS_LEN)
	for {
		command, err2 := exec.Run(STATUS_CMD, args, nil, ".", exec.PassThrough, exec.Pipe, exec.PassThrough)
		if err2 != nil{
			log.Println("chunk fails in command:" + err2.String())
			log.Fatal("chunk: unable to obtain remote command")
		}
		/*length,err3 := command.Stdout.Read(result)
		if err3 != nil{
			log.Println("chunk fails read from command: " + err3.String())
		}*/
		length := 0
		var err3 os.Error
		for j :=0; j < 4; j ++ {
			if (length == 0) {
				time.Sleep(4000000000)
				length,err3 =command.Stdout.Read(result)
				if err3 != nil{
				log.Println("chunk fails read from command: " + err3.String())
				}
			}
		}
		logFile.Write(result)
		logFile.WriteString("\nNextRecord\n");
	}
	logFile.Close();		
}

func SendHeartbeat(masterAddress string){
	var args sfs.HeartbeatArgs
	var ret  sfs.HeartbeatReturn
	
	master, err := rpc.Dial("tcp", masterAddress + ":1338")
	if err != nil {
		log.Fatal("chunk: dialing:", err)
	}

	host,_ := os.Hostname()
	_,iparray,_ := net.LookupHost(host)
	tcpAddr,_ := net.ResolveTCPAddr(iparray[0] + ":1337")
	args.ChunkServerIP = *tcpAddr
	args.ChunkServerID = chunkServerID

	for {
		args.Capacity = capacity
		addedChunkSlice := make([]sfs.ChunkInfo, addedChunks.Len())
		for i := 0; i < addedChunks.Len(); i++ {
			addedChunkSlice[i] = addedChunks.At(i).(sfs.ChunkInfo)
		}
		args.AddedChunks = addedChunkSlice
		err = master.Call("Master.BeatHeart", &args, &ret)
		if err != nil {
			log.Fatal("chunk: heartbeat error: ", err)
		}
		addedChunks.Resize(0, 0)
		time.Sleep(sfs.HEARTBEAT_WAIT)	
	}
	return
}

func (t *Server) ReplicateChunk(args *sfs.ReplicateChunkArgs, ret *sfs.ReplicateChunkReturn) os.Error {

	if args.Servers == nil {
		log.Printf("chunk: replication call: nil address.")
		return nil
	}

	log.Printf("replication request for site %s and chunk %d\n",
		args.Servers[0].String(),args.ChunkID);


	for i := 0; i < len(args.Servers); i++ {
		replicationHost, err := rpc.Dial("tcp", args.Servers[i].String())
		if err != nil {
			log.Println("chunk: replication call:", err)
			continue
		}
		
		var readArgs sfs.ReadArgs
		var readRet sfs.ReadReturn
		readArgs.ChunkID = args.ChunkID
		
		err = replicationHost.Call("Server.Read", &readArgs, &readRet)
		if err != nil {
			log.Println("chunk: replication call:", err)
			continue
		}
		
		chunkTable[args.ChunkID] = readRet.Data
		break
	}
	return nil
}
