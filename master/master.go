package master

import(
	"net"
	"./trie"
	"fmt"
	"os"
	"container/vector"
	"../include/sfs"
)

var t *trie.Trie
var nextChunk uint64 = 0
var serverIndex uint64 = 0
var servers *vector.Vector

type inode struct {
	name string
	permissions uint64
	size uint64
	chunk uint64
	addr net.TCPAddr
}

/*type chunk struct {
	chunkId		uint64
	pFileChunk	*chunk
	nFileChunk	*chunk
	pServChunk	*chunk
	nServChunk	*chunk
	server		*server
}*/

type Master int

func (m *Master) ReadOpen(args *sfs.OpenArgs, info *sfs.OpenReturn) os.Error {
	i, newFile, err := OpenFile(args.Name)
	
	info.New = newFile
	info.Size = i.size
	info.Chunk = i.chunk
	info.ServerLocation = i.addr
	
	return err
}

func (m *Master) ReadChunkPing(args *sfs.PingArgs, info *sfs.PingReturn) os.Error {
	AddServer(args.ChunkServer)

	return nil
}

func OpenFile(name string) (i *inode, newFile bool, err os.Error){
	err = nil
	
	i, newFile = QueryFile(name)
	
	newFile = !newFile
	
	if newFile {
		fmt.Printf("OpenFile: file %s does not exist\n", name)
		i, err = AddFile(name)
	}
	
	return i, newFile, err
}

func AddFile(name string) (i *inode, err os.Error) {
	i = new(inode)
	
	fmt.Printf("AddFile: nextChunk %d, len(servers) %d\n", nextChunk, servers.Len())
	
	i.size = 1
	i.addr = *(servers.At(int(nextChunk) % servers.Len()).(*net.TCPAddr))
	//i.addr = servers[0]
	i.chunk = nextChunk
	nextChunk += 1
	
	t.AddValue(name, i)
	
	return i, nil
}

func QueryFile(name string) (i *inode, fileExists bool) {
	inter, exists := t.GetValue(name)
	
	if !exists{
		fmt.Printf("QueryFile: file %s does not exist\n", name)
		return nil, exists
	}
	
	return inter.(*inode), exists
}

func AddServer(server net.TCPAddr) os.Error {
	str := fmt.Sprintf("%s:%d", server.IP.String(), server.Port)
	fmt.Printf("AddServer: adding %s\n", str)
	servers.Push(&server)
		
	return nil
}

func init() {
	t = trie.NewTrie()
	servers = new(vector.Vector)
}