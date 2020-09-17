package main

import "github.com/noriah/tavis"

func main() {
	var err error
	if err = tavis.Run(); err != nil {
		panic(err)
	}
}
