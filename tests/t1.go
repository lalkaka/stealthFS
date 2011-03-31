package main

import (
	"../client/client"
	"fmt"
	"flag"
	"os"
)

func main(){
	master := flag.String("m", "", "specify a master!")
	flag.Parse();

	client.Initialize(*master)

	fd := client.Open("newfile.txt", client.O_WRONLY|client.O_CREATE)
	if(fd < 0) {
		panic("open failed")
	}

	fmt.Printf("pass\n")
	os.Exit(0)
}

