package ios

import "github.com/coversocks/gocs/netstack/tcpip"

func Create() {
	tcpip.NewStack(false, nil)
}
