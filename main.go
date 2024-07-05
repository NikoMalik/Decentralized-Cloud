package main

import (
	"fmt"

	"os"

	"github.com/NikoMalik/Decentralized-Cloud/p2p"
	"github.com/fatih/color"
)

func main() {
	transport, err := p2p.NewTcpTransport(":3000")
	printBanner()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = transport.ListenAndAccept()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer transport.Stop()

	for {
		select {
		case conn := <-transport.Connections():

			if conn == nil {
				color.Red("Received nil connection")
				break
			}
			color.Blue("New connection accepted: %s", conn.RemoteAddr().String())

		case msg := <-transport.Messages():
			if msg.Payload == nil {
				break
			}
			go func() {
				if err := transport.HandleMessage(&msg); err != nil {
					color.Red("Failed to handle message: %v", err)
				}
			}()

		}
	}
}

func printBanner() {
	banner := `

  _   _ _ _         __  __       _ _ _    
 | \ | (_) |       |  \/  |     | (_) |   
 |  \| |_| | _____ | \  / | __ _| |_| | __
 | .  | | |/ / _ \| |\/| |/ _ | | | |/ /
 | |\  | |   < (_) | |  | | (_| | | |   < 
 |_| \_|_|_|\_\___/|_|  |_|\__,_|_|_|_|\_\
										
`

	fmt.Println(color.RedString(banner))
}
