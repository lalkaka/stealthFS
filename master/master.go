package master

import(
	"net"
	"./trie"
	"log"
	"os"
	"container/vector"
	"container/heap"
	"../include/sfs"
)

var t *trie.Trie
var nextChunk uint64 = 0
var serverIndex uint64 = 0
var servers *heap.Heap

var nReplicas int = 1

type inode struct {
	name string
	permissions uint64
	size uint64
	chunks *vector.Vector
}

type server struct {
	addr net.TCPAddr
	capacity uint64
	chunks *vector.Vector
}

type chunk struct {
	chunkID		uint64
	servers		*vector.Vector
}

type Master int

func (m *Master) ReadOpen(args *sfs.OpenArgs, info *sfs.OpenReturn) os.Error {
	i, newFile, err := OpenFile(args.Name)
	
	info.New = newFile
	info.Size = i.size
	info.Chunk = (i.chunks.At(0).(*chunk)).chunkID
	info.ServerLocation = ((i.chunks.At(0).(*chunk)).servers.At(0).(*server)).addr
	
	return err
}

func (m *Master) ReadChunkPing(args *sfs.PingArgs, info *sfs.PingReturn) os.Error {
	AddServer(args.ChunkServer, args.Capacity)

	return nil
}

func OpenFile(name string) (i *inode, newFile bool, err os.Error){
	err = nil
	
	i, newFile = QueryFile(name)
	
	newFile = !newFile
	
	if newFile {
		log.Printf("OpenFile: file %s does not exist\n", name)
		i, err = AddFile(name)
	}
	
	return i, newFile, err
}

func AddFile(name string) (i *inode, err os.Error) {
	i = new(inode)
	
	log.Printf("AddFile: nextChunk %d, len(servers) %d\n", nextChunk, servers.Len())
	
	i.size = 1
	//i.addr = *(servers.At(int(nextChunk) % servers.Len()).(*net.TCPAddr))
	//i.addr = servers[0]
	
	i.AddChunk()
	
	t.AddValue(name, i) // trie insert
	
	return i, nil
}

func QueryFile(name string) (i *inode, fileExists bool) {
	inter, exists := t.GetValue(name)
	
	if !exists{
		log.Printf("QueryFile: file %s does not exist\n", name)
		return nil, exists
	}
	
	return inter.(*inode), exists
}

func (i *inode) AddChunk() chunkID uint64 {
	serv := heap.Pop(servers)
	thisChunk := new(chunk)
	thisChunk.chunkID = nextChunk
	nextChunk += 1
	
	thisChunk.servers = new(vector.Vector)
	
	thisChunk.servers.Push(serv)	
	serv.chunks.Push(thisChunk)
	i.chunks.Push(thisChunk)
	
	heap.Push(servers, serv)
	
	return thisChunk.chunkID
}

func AddServer(servAddr net.TCPAddr, capacity uint64) os.Error {
	str := log.Sprintf("%s:%d", servAddr.IP.String(), servAddr.Port)
	log.Printf("AddServer: adding %s\n", str)
	
	var s server
	
	s.addr = servAddr
	s.capacity = capacity
	
	heap.Push(servers, s)
		
	return nil
}

func init() {
	t = trie.NewTrie()
	servers = new(serverHeap)
	heap.Init(servers)
}