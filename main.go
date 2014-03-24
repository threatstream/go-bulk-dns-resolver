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
	Options struct {
		// Preserve input line (if this is true then don't cleanup and reduce to just the domain name).
		preserveInput bool
	}
	Result struct {
		domain       string
		originalLine string
		dnsServer    string
		response     *dns.Msg
		rtt          time.Duration
		err          error
	}
)

const (
	CONCURRENCY = 1000 //000 //250
)

var (
	options = Options{
		preserveInput: false,
	}

	domainCleanerRe = regexp.MustCompile(`^(?:[0-9]+,)?([^\/]*)(?:\/.*)?$`)

	ring DnsServerRing

	// More public DNS servers:
	//     https://www.grc.com/dns/alternatives.htm
	//     http://www.bestfreedns.net/
	extraServers = []string{
		"8.8.8.8",         // Google - CA
		"8.8.4.4",         // Google - CA
		"129.250.35.250",  // Verio
		"129.250.35.251",  // Verio
		"209.244.0.3",     // Level3 - CA
		"209.244.0.4",     // Level3 - CA
		"4.2.2.1",         // Verizon
		"4.2.2.2",         // Verizon
		"173.230.156.28",  // OpenNIC - CA
		"172.246.141.148", // OpenNIC - CA
		"23.90.4.6",       // OpenNIC - AZ
		"23.226.230.72",   // OpenNIC - WA
		"68.87.85.98",     // Comcast: West Coast
		"68.87.76.178",    // Comcast: Sacramento Primary
		"68.87.78.130",    // Comcast: Sacramento Secondary
		"68.87.76.178",    // Comcast: San Francisco Primary
		"68.87.78.130",    // Comcast: San Francisco Secondary
		"68.87.76.178",    // Comcast: Los Angeles Primary
		"68.87.78.130",    // Comcast: Los Angeles Secondary
		"68.87.69.146",    // Comcast: Orgeon Primary
		"68.87.85.98",     // Comcast: Orgeon Secondary
		"68.87.85.98",     // Comcast: Utah Primary
		"68.87.69.146",    // Comcast: Utah Secondary
	}

	ch = make(chan Result, CONCURRENCY)

	outputLock sync.Mutex
)

// Initialize the ring and remove any servers that don't work.
func init() {
	clientConfig, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		panic(err)
	}
	ring = DnsServerRing{
		Index:   -1,
		Servers: Append(clientConfig.Servers, extraServers...),
	}
	err = ring.Test()
	if err != nil {
		panic(err)
	}
}

func resolve(line string, dnsServer string, numAttempts int) {
	//SyncPrintf("started resolving line=%s\n", line)
	domain := domainCleanerRe.ReplaceAllString(line, "$1")
	m := new(dns.Msg)
	m.SetQuestion(domain+".", dns.TypeA&dns.TypeCNAME)
	c := new(dns.Client)
	c.DialTimeout = 3e9
	c.ReadTimeout = 5e9
	c.WriteTimeout = 3e9
	response, rtt, err := c.Exchange(m, dnsServer+":53")

	if err != nil {
		if strings.Contains(err.Error(), IO_TIMEOUT_ERROR_MESSAGE) {
			resolve(line, ring.Next(), numAttempts+1)
			return
		} else if numAttempts < len(ring.Servers)/2 {
			resolve(line, ring.Next(), numAttempts+1)
			return
		} else {
			SyncPrintf("failed: max attempts exhausted for domain=%s error=%s\n", domain, err)
		}
	} else {
		messageHeader := response.String()
		if messageHeader == "<nil> MsgHdr" || !answerBlockRe.MatchString(messageHeader) {
			if numAttempts < len(ring.Servers)/2 {
				//fmt.Printf("retrying %v\n", domain)
				resolve(line, ring.Next(), numAttempts+1)
				return
			} else {
				SyncPrintf("failed: max attempts exhausted, no answer found for domain=%s\n", domain)
			}
		}
	}
	//for _, a := range response.Answer {
	//	fmt.Printf("a=%v", a.String())
	//}
	//SyncPrintf(dnsServer + "\n")
	ch <- Result{domain, line, dnsServer, response, rtt, err}
}

func Worker(linkChan chan string, wg *sync.WaitGroup) {
	// Decreasing internal counter for wait-group as soon as goroutine finishes
	defer wg.Done()

	for domain := range linkChan {
		// Analyze value and do the job here
		resolve(domain, ring.Next(), 1)
	}
	//SyncPrintf("ALL DONE!\n")
}

func SyncPrintf(msg string, args ...interface{}) {
	outputLock.Lock()
	fmt.Printf(msg, args...)
	os.Stdout.Sync()
	outputLock.Unlock()
}

func main() {
	// Parse and validate args.
	leftovers, optargs, err := getopt.GetOpt(os.Args[1:], "p", []string{"preserve"})
	if err != nil {
		SyncPrintf("error: %s\n", err)
		return
	} else if len(leftovers) > 0 {
		SyncPrintf("error: unrecognized parameter: %s\n", leftovers)
		return
	}
	if len(optargs) > 0 && optargs[0].Opt() == "-p" {
		//SyncPrintf("Found opt!\n")
		options.preserveInput = true
	}

	domains := ReadLinesFromStdin(func(line string) string {
		return strings.TrimSpace(line)
	})

	tasks := make(chan string, CONCURRENCY) //len(domains))

	// Spawn worker goroutines.
	wg := new(sync.WaitGroup)

	// Adding routines to workgroup and running then.
	for i := 0; i < CONCURRENCY; i++ {
		wg.Add(1)
		go Worker(tasks, wg)
	}

	receiver := func(numDomains int) {
		defer wg.Done()

		i := 0
	Loop:
		for {
			select {
			case result := <-ch:
				//log.Println(result.response)
				domain, ips, err := ParseResponse(result.domain, result.response.String())
				if err != nil && len(ips) == 0 {
					SyncPrintf("failed: domain=%s :: dns-server=%s :: error=%s\n", result.domain, result.dnsServer, err.Error())
				} // else if len(ips) > 0 {
				if options.preserveInput { // Always include input domains in output.
					SyncPrintf("%s %s\n", result.originalLine, strings.Join(ips, " "))
				} else {
					SyncPrintf("%s %s\n", domain, strings.Join(ips, " "))
				}
				//}
				i++
				if i == numDomains {
					break Loop
				}
			}
		}
	}

	wg.Add(1)
	go receiver(len(domains))

	// Processing all links by spreading them to `free` goroutines
	for _, domain := range domains {
		tasks <- domain
	}

	close(tasks)

	// Wait for the workers to finish.
	wg.Wait()
}
