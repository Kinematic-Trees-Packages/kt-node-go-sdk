package main

import (
	"fmt"
	"log"

	kt "github.com/kinematic-trees-packages/kt-node-go-sdk/go/ktnode"
)

func main() {
	major, minor := kt.ABIVersion()
	fmt.Printf("kt_node ABI %d.%d\n", major, minor)
	rtMajor, rtMinor, rtPatch, err := kt.RuntimeVersion()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("kt_node runtime %d.%d.%d\n", rtMajor, rtMinor, rtPatch)
	fmt.Printf("build_id %s\n", kt.BuildID())
}
