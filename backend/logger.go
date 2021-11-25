package main

import "log"

func logErr(err error, msg string) {
	if err != nil {
		log.Printf("%s: %v\n", msg, err)
	} else {
		log.Println(msg)
	}
}
