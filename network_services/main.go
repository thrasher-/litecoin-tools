package main

import (
	"fmt"
	"strings"
)

const (
	NODE_NONE    uint64 = 0
	NODE_NETWORK uint64 = (1 << 0)
	NODE_GETUTXO uint64 = (1 << 1)
	NODE_BLOOM   uint64 = (1 << 2)
	NODE_WITNESS uint64 = (1 << 3)
	NODE_XTHIN   uint64 = (1 << 4)
)

func FormatServices(mask uint64) string {
	services := []string{}
	for i := 0; i < 8; i++ {
		var check uint64 = 1 << uint64(i)
		if mask&check != 0 {
			switch check {
			case NODE_NETWORK:
				services = append(services, "NETWORK")
				break
			case NODE_GETUTXO:
				services = append(services, "GETUTXO")
				break
			case NODE_BLOOM:
				services = append(services, "BLOOM")
				break
			case NODE_WITNESS:
				services = append(services, "WITNESS")
				break
			case NODE_XTHIN:
				services = append(services, "XTHIN")
				break
			default:
				services = append(services, fmt.Sprintf("UNKNOWN[%v]", check))
			}
		}
	}
	if len(services) > 0 {
		return strings.Join(services, " & ")
	} else {
		return "None"
	}
}

func main() {
	fmt.Println(FormatServices(NODE_NETWORK))
	fmt.Println(FormatServices(13))
	nServices := NODE_NETWORK | NODE_GETUTXO | NODE_BLOOM | NODE_WITNESS | NODE_XTHIN
	fmt.Println(FormatServices(nServices))
	fmt.Println(FormatServices(0))
}
