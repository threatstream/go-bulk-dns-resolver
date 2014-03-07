package main

import (
	"log"
	"bulkdns"
	//"testing"
)

func ParseResponseF() {
	message := `A

;; ANSWER SECTION:
www.hotmail.com.	2688	IN	CNAME	dispatch.kahuna.glbdns2.microsoft.com.
dispatch.kahuna.glbdns2.microsoft.com.	4	IN	A	65.55.143.16
dispatch.kahuna.glbdns2.microsoft.com.	4	IN	A	65.55.143.17

;;`
	bulkdns.ParseResponse(message)
}


func main() {
	log.Println("Tests started")
	ParseResponseF()
}
