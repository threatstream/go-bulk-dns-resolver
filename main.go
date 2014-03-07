package main

import (
	"fmt"
	"time"
	"regexp"
	"strings"

	//"bulkdns"
	"github.com/miekg/dns"
)

type (
	Result struct {
		domain string
		dnsServer string
		msg *dns.Msg
		rtt time.Duration
		err error
	}

	DnsServerRing struct {
		index int
		servers []string
	}
)


var (
	inputCleanerRe = regexp.MustCompile(`^(?:[0-9]+,)?([^\/]*)(?:\/.*)?$`)

	ring = DnsServerRing{-1, []string{
		"8.8.8.8", // Google - CA
		"8.8.4.4", // Google - CA
		"209.244.0.3", // Level3 - CA
		"209.244.0.4", // Level3 - CA
		"4.2.2.1", // Verizon
		"4.2.2.2", // Verizon
		"173.230.156.28", // OpenNIC - CA
		"172.246.141.148", // OpenNIC - CA
		"23.90.4.6", // OpenNIC - AZ
		"23.226.230.72", // OpenNIC - WA
	}}

	ch = make(chan Result)
)

const (
	MAX_ATTEMPTS = 5
)


func (this *DnsServerRing) next() string {
	if this.index < 0 || this.index + 1 == len(this.servers) {
		this.index = 0
	} else {
		this.index++
	}
	return this.servers[this.index]
}


func resolve(domain string, dnsServer string, attemptNumber int) {
	m := new(dns.Msg)
	m.SetQuestion(domain + ".", dns.TypeA & dns.TypeCNAME)
	c := new(dns.Client)
	msg, rtt, err := c.Exchange(m, dnsServer + ":53")

	if err != nil {
		//fmt.Printf("notice :: %s\n", err)
		if attemptNumber < MAX_ATTEMPTS {
			resolve(domain, ring.next(), attemptNumber + 1)
		} else {
			fmt.Printf("failed :: max attempts exhausted for domain=%s error=%s\n", domain, err)
		}
	}

	if msg.String() == "<nil> MsgHdr" {
		if attemptNumber < MAX_ATTEMPTS { // && strings.Contains(domain, "onclickads.net") {
			//fmt.Printf("!!!!!!!!!! JAY !!!!!!!!!!!!!! RETRYING %s: %s\n", domain, msg.String())
			resolve(domain, ring.next(), attemptNumber + 1)
		} else {
			fmt.Printf("failed :: max attempts exhausted for domain=%s\n", domain)
		}
	}
	//fmt.Printf(dnsServer + "\n")
	ch <- Result{domain, dnsServer, msg, rtt, err}
}


func main() {

	domains := readLinesFromStdin(func(line string) string {
		return strings.TrimSpace(inputCleanerRe.ReplaceAllString(line, "$1"))
	})

	//fmt.Println(domains)

	for _, domain := range domains {
		go resolve(domain, ring.next(), 1)
	}
	/*for _, domain := range domains {
		fmt.Println("------- " + domain)
	}*/
	i := 0

Loop:
	for {
		select {
		case result := <-ch:
			//log.Println(result.msg)
			domain, ips, err := ParseResponse(result.domain, result.msg.String())
			if err != nil {
				fmt.Printf("failed :: domain=%s :: dns-server=%s :: error=%s", result.domain, result.dnsServer, err.Error())
			}
			fmt.Printf("%s %s\n", domain, strings.Join(ips, " "))
			/*if i == 0 {
				first = result
				fmt.Println(first.msg)
			} else {
				if first.rtt != result.rtt {
					fmt.Println("All rtt should be equal")
					return
					//t.Fail()
				}
			}*/
			i++
			if i == len(domains) {
				break Loop
			}
		}
	}
	//fmt.Println("done")
	/*args := os.Args
	if len(args) < 2 {
		fmt.Fatalln("expected at least one argument")
		return
	}
	switch args[1] {
	case "server":
		fmt.Println(new(Server).start())
	default:
		new(Client).Do(args)
	}*/

}
