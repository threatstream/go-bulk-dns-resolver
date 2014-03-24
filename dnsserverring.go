package main

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
)

type (
	DnsServerRing struct {
		Index   int
		Servers []string
	}
)

const (
	IO_TIMEOUT_ERROR_MESSAGE = "i/o timeout"
)

// TODO: protect this region to be accessible by only 1 thread at a time.
func (this *DnsServerRing) Next() string {
	if this.Index < 0 || this.Index+1 == len(this.Servers) {
		this.Index = 0
	} else {
		this.Index++
	}
	return this.Servers[this.Index]
}

func (this *DnsServerRing) Test() error {
	domain := "google.com"
	m := new(dns.Msg)
	m.SetQuestion(domain+".", dns.TypeA&dns.TypeCNAME)
	c := new(dns.Client)
	c.DialTimeout = 3e9
	c.ReadTimeout = 5e9
	c.WriteTimeout = 3e9
	working := []string{}
	for _, server := range this.Servers {
		/*response*/ _, rtt, err := c.Exchange(m, server+":53")
		fmt.Printf("rtt=%v\n", rtt)
		if err != nil {
			fmt.Printf("failed: server=%v: %v\n", server, err)
		} else {
			fmt.Printf("info: good server=%v\n", server)
			working = Append(working, server)
		}
		//fmt.Printf("%v\n", response)
	}
	this.Servers = working
	if len(this.Servers) == 0 {
		return fmt.Errorf("fatal: no working dns servers found")
	}
	return nil
}

func (this *DnsServerRing) Resolve(domain string) (*dns.Msg, error) {
	m := new(dns.Msg)
	m.SetQuestion(domain+".", dns.TypeA&dns.TypeCNAME)
	c := new(dns.Client)
	c.DialTimeout = 3e9
	c.ReadTimeout = 5e9
	c.WriteTimeout = 3e9
	response, rtt, err := c.Exchange(m, this.Next()+":53")
	fmt.Printf("rtt=%v\n", rtt)
	if err != nil && strings.Contains(err.Error(), IO_TIMEOUT_ERROR_MESSAGE) {
		fmt.Printf("error: %v\n", err)
		return this.Resolve(domain)
	} else if err != nil {
		return nil, fmt.Errorf("failed: max attempts exhausted for domain=%s error=%s\n", domain, err)
	}
	if response.Rcode != dns.RcodeSuccess {
		fmt.Printf("warning: Rcode wasn't success: %v\n", response.Rcode)
	}
	return response, nil
}
