package main

import (
	"log"

	"github.com/heppu/cvm/client"
	"github.com/heppu/cvm/git"
)

func main() {
	v, err := git.GetVersions()
	log.Println(err)
	log.Println(v)
	return

	c := client.NewClient()
	fi, err := c.GetRevisions("Linux_x64/f958546f4f0cf2b34c46490d9ef4a4baa26e9c4c/")
	log.Println(err)
	log.Printf("%+v", fi)
	return

	if err := c.GetAll(); err != nil {
		log.Fatal(err)
	}
}
