package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/timtadh/getopt"
)

type (
	Result struct {
		domain string
		originalLine string
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

	// More public DNS servers:
	//     https://www.grc.com/dns/alternatives.htm
	//     http://www.bestfreedns.net/
	ring = DnsServerRing{-1, []string{
		"8.8.8.8", // Google - CA
		"8.8.4.4", // Google - CA
		"129.250.35.250", // Verio
		"129.250.35.251", // Verio
		"209.244.0.3", // Level3 - CA
		"209.244.0.4", // Level3 - CA
		"4.2.2.1", // Verizon
		"4.2.2.2", // Verizon
		"173.230.156.28", // OpenNIC - CA
		"172.246.141.148", // OpenNIC - CA
		"23.90.4.6", // OpenNIC - AZ
		"23.226.230.72", // OpenNIC - WA
	}}

	ch = make(chan Result, 1000)

	// Preserve input line (if this is true then don't cleanup and reduce to just the domain name).
	preserveInput = false
)


const (
	MAX_ATTEMPTS = 10

	CONCURRENCY = 1000
)


// TODO: protect this region to be accessible by only 1 thread at a time.
func (this *DnsServerRing) next() string {
	if this.index < 0 || this.index + 1 == len(this.servers) {
		this.index = 0
	} else {
		this.index++
	}
	return this.servers[this.index]
}


func resolve(line string, dnsServer string, attemptNumber int) {
	//fmt.Printf("started resolving " + domain + "\n")
	domain := inputCleanerRe.ReplaceAllString(line, "$1")
	m := new(dns.Msg)
	m.SetQuestion(domain + ".", dns.TypeA & dns.TypeCNAME)
	c := new(dns.Client)
	msg, rtt, err := c.Exchange(m, dnsServer + ":53")

	if err != nil {
		//fmt.Printf("notice :: %s\n", err)
		if attemptNumber < MAX_ATTEMPTS {
			resolve(domain, ring.next(), attemptNumber + 1)
			return
		} else {
			fmt.Printf("failed :: max attempts exhausted for domain=%s error=%s\n", domain, err)
		}
	}

	if msg.String() == "<nil> MsgHdr" {
		if attemptNumber < MAX_ATTEMPTS {
			//fmt.Printf("RETRYING %s: %s\n", domain, msg.String())
			resolve(domain, ring.next(), attemptNumber + 1)
			return
		} else {
			fmt.Printf("failed :: max attempts exhausted for domain=%s\n", domain)
		}
	}
	//fmt.Printf(dnsServer + "\n")
	ch <- Result{domain, line, dnsServer, msg, rtt, err}
}


func worker(linkChan chan string, wg *sync.WaitGroup) {
	// Decreasing internal counter for wait-group as soon as goroutine finishes
	defer wg.Done()

	for domain := range linkChan {
		// Analyze value and do the job here
		resolve(domain, ring.next(), 1)
	}
	//fmt.Printf("ALL DONE!\n")
}


func main() {

	// Parse and validate args.
	leftovers, optargs, err := getopt.GetOpt(os.Args[1:], "p", []string{"preserve"})
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	} else if len(leftovers) > 0 {
		fmt.Printf("error: unrecognized parameter: %s\n", leftovers)
		return
	}
	if len(optargs) > 0 && optargs[0].Opt() == "-p" {
		//fmt.Printf("Found opt!\n")
		preserveInput = true
	}


	domains := ReadLinesFromStdin(func(line string) string {
		return strings.TrimSpace(line)
	})

	tasks := make(chan string, CONCURRENCY)//len(domains))

	// Spawn worker goroutines.
	wg := new(sync.WaitGroup)

	// Adding routines to workgroup and running then.
	for i := 0; i < CONCURRENCY; i++ {
		wg.Add(1)
		go worker(tasks, wg)
	}

	receiver := func() {
		i := 0
Loop:
		for {
			select {
			case result := <-ch:
				//log.Println(result.msg)
				domain, ips, err := ParseResponse(result.domain, result.msg.String())
				if err != nil && len(ips) == 0 {
					fmt.Printf("failed :: domain=%s :: dns-server=%s :: error=%s\n", result.domain, result.dnsServer, err.Error())
				} else if len(ips) > 0 {
					if preserveInput {
						fmt.Printf("%s %s\n", result.originalLine, strings.Join(ips, " "))
					} else {
						fmt.Printf("%s %s\n", domain, strings.Join(ips, " "))
					}
				}
				i++
				if i == len(domains) {
					break Loop
				}
			}
		}
	}

	go receiver()

	// Processing all links by spreading them to `free` goroutines
	for _, domain := range domains {
		tasks <- domain
	}

	close(tasks)

	// Wait for the workers to finish.
	wg.Wait()
}
