package main

import "C"
import "github.com/coversocks/gocs/netstack/coversocks"

func main() {

}

//export hello
func hello() {
	coversocks.Hello()
}
