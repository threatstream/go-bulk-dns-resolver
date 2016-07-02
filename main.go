package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"net"

	//"github.com/miekg/dns"
	"github.com/miekg/unbound"
	"github.com/timtadh/getopt"
)

type (
	Result struct {
		domain       string
		originalLine string
		addresses    []string
		err          error
	}

	DnsServerRing struct {
		index   int
		servers []string
	}
)

const (
	MAX_ATTEMPTS = 3

	CONCURRENCY = 250 //1000

	LOOKUP_TIMEOUT_SECONDS = 5
)

var (
	domainCleanerRe = regexp.MustCompile(`^([0-9]+,)?([^\/]*)(?:\/.*)?$`)

	// More public DNS servers:
	//     https://www.grc.com/dns/alternatives.htm
	//     http://www.bestfreedns.net/
	ring = DnsServerRing{-1, []string{
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
		"68.87.85.98",     // Comcast West Coast
		"68.87.76.178",    // Comcast Sacramento Primary
		"68.87.78.130",    // Comcast Sacramento Secondary
		"68.87.76.178",    // Comcast San Francisco Primary
		"68.87.78.130",    // Comcast San Francisco Secondary
		"68.87.76.178",    // Comcast Los Angeles Primary
		"68.87.78.130",    // Comcast Los Angeles Secondary
		"68.87.69.146",    // Comcast Orgeon Primary
		"68.87.85.98",     // Comcast Orgeon Secondary
		"68.87.85.98",     // Comcast Utah Primary
		"68.87.69.146",    // Comcast Utah Secondary
	}}

	ch = make(chan Result, CONCURRENCY)

	// Preserve input line (if this is true then don't cleanup and reduce to just the domain name).
	preserveInput = false

	outputLock sync.Mutex

	unboundInstance = unbound.New()
)

func init() {
	unboundInstance.ResolvConf("/etc/resolv.conf")
}

// TODO: protect this region to be accessible by only 1 thread at a time.
func (this *DnsServerRing) next() string {
	if this.index < 0 || this.index+1 == len(this.servers) {
		this.index = 0
	} else {
		this.index++
	}
	return this.servers[this.index]
}

func resolve(line string, attemptNumber int) {
	domain := domainCleanerRe.ReplaceAllString(line, "$2")
	// Clean out any trailing characters after the domain name (i.e. useful when a url is submitted *cough* Alexa).
	cleanLine := domainCleanerRe.ReplaceAllString(line, "$1$2")
	addresses := []string{}
	var err error

	timeout := make(chan error)
	go func() {
		ip := net.ParseIP(domain)
		if ip == nil {
			addresses, err = unboundInstance.LookupHost(domain)
		} else {
			addresses, err = unboundInstance.LookupAddr(domain)
		}
		timeout <- err
	}()

	select {
	case err = <-timeout:
		if err != nil {
			if attemptNumber < MAX_ATTEMPTS {
				resolve(line, attemptNumber+1)
				return
			}
			SyncPrintf("failed: max attempts exhausted for domain=%s error=%s\n", domain, err)
		}
	case <-time.After(LOOKUP_TIMEOUT_SECONDS * time.Second):
		err = fmt.Errorf("error: timed out for \"%v\" after %v seconds", domain, LOOKUP_TIMEOUT_SECONDS)
	}
	ch <- Result{domain, cleanLine, addresses, err}
}

/*func resolveOldWay(line string, dnsServer string, attemptNumber int) {
	//SyncPrintf("started resolving line=%s\n", line)
	domain := domainCleanerRe.ReplaceAllString(line, "$2")
	// Clean out any trailing characters after the domain name (i.e. useful when a url is submitted *cough* Alexa).
	cleanLine := domainCleanerRe.ReplaceAllString(line, "$1$2")

	u := unbound.New()
	defer u.Destroy()
	u.ResolvConf("/etc/resolv.conf")
	a, err := u.LookupHost(domain)
	if err != nil {
		fmt.Printf("FAILURE: %v\n", err)
	}
	fmt.Printf("a=%v\n", a)

	m := new(dns.Msg)
	m.SetQuestion(domain+".", dns.TypeA&dns.TypeCNAME)
	c := new(dns.Client)
	response, rtt, err := c.Exchange(m, dnsServer+":53")

	if err != nil {
		//SyncPrintf("notice :: %s\n", err)
		if attemptNumber < MAX_ATTEMPTS {
			resolve(line, ring.next(), attemptNumber+1)
			return
		} else {
			SyncPrintf("failed :: max attempts exhausted for domain=%s error=%s\n", domain, err)
		}
	}

	messageHeader := response.String()
	if messageHeader == "<nil> MsgHdr" || !answerBlockRe.MatchString(messageHeader) {
		if attemptNumber < MAX_ATTEMPTS {
			//SyncPrintf("RETRYING %s: %s\n", domain, response.String())
			resolve(line, ring.next(), attemptNumber+1)
			return
		} else {
			SyncPrintf("failed :: no answer found for domain=%s, max attempts exhausted\n", domain)
		}
	}
	//SyncPrintf(dnsServer + "\n")
	_, addresses, err := ParseResponse(response.domain, response.response.String())
	if err != nil {
		fmt.Printf("failed :: response parser returned error: %v\n", err)
	}
	ch <- Result{domain, cleanLine, addresses, response, rtt, err}
}*/

func worker(linkChan chan string, wg *sync.WaitGroup) {
	// Decreasing internal counter for wait-group as soon as goroutine finishes
	defer wg.Done()

	for domain := range linkChan {
		// Analyze value and do the job here
		resolve(domain, 1)
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
	// Destroy after `main()` runs.
	defer unboundInstance.Destroy()

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
		preserveInput = true
	}

	domains := ReadLinesFromStdin(func(line string) string {
		return strings.TrimSpace(line)
	})

	resultMap := make(map[string]bool)
	for _, d := range domains {
		resultMap[d] = false
	}

	tasks := make(chan string, CONCURRENCY) //len(domains))

	// Spawn worker goroutines.
	wg := new(sync.WaitGroup)

	// Adding routines to workgroup and running then.
	for i := 0; i < CONCURRENCY; i++ {
		wg.Add(1)
		go worker(tasks, wg)
	}

	receiver := func(numDomains int) {
		defer wg.Done()

		i := 0
	Loop:
		for {
			select {
			case result := <-ch:
				//log.Println(result.response)
				//domain, ips, err := ParseResponse(result.domain, result.response.String())
				if err != nil {
					SyncPrintf("failed :: domain=%s :: error=%s\n", result.domain, err.Error())
					if preserveInput {
						SyncPrintf("%s %s\n", result.originalLine, strings.Join(result.addresses, " "))
					} else {
						SyncPrintf("%s %s\n", result.domain, strings.Join(result.addresses, " "))
					}
				} else {
					if preserveInput {
						SyncPrintf("%s %s\n", result.originalLine, strings.Join(result.addresses, " "))
					} else {
						SyncPrintf("%s %s\n", result.domain, strings.Join(result.addresses, " "))
					}
				}
				resultMap[result.domain] = true
				i++
				//fmt.Printf("%v/%v\n", i, numDomains)
				if i == numDomains {
					break Loop
				}
				/*for d, _ := range resultMap {
					if resultMap[d] == false {
						fmt.Printf("still waiting on: %v\n", d)
					}
				}*/
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
