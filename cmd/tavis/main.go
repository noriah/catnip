package main

import "github.com/noriah/tavis"

func main() {
	if err := tavis.Run(); err != nil {
		panic(err)
	}
}
