package main

import (
	"fmt"
	"struct2pb/core"
	"struct2pb/obj"
)

func main() {
	result := core.Structs2Pb(true, obj.List...)
	fmt.Println(result)
}
