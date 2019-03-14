package main

import (
	"fmt"
	"os"

	"github.com/globalsign/mgo"
)

func main() {
	_, err := mgo.Dial("localhost:27017")
	if err.Error() == "no reachable servers" {
		fmt.Println("sorted")
		os.Exit(1)
	}
	// try reading a config.txt, if not exist, create one + db (if not exists.....)
	fmt.Println("aye-go")
}
