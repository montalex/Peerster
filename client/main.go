package client

import (
	"flag"
	"fmt"
)

func main() {
	var UIPort = flag.Int("UIPort", 4242, "Port for UI client")
	var msg = flag.String("msg", "", "message to be sent")

	flag.Parse()

	//** DEBUG:
	fmt.Println("UIPort = ", *UIPort)
	fmt.Println("msg = ", *msg)
	//*/
}
