package main

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	answerBlockRe = regexp.MustCompile(`;; ANSWER SECTION:(?:[^\n]|\n[^\n])+`)
	answerDomainExtractorRe = regexp.MustCompile(`^((?:[^\.]|\.[^ \t])+)\.[ \t].*$`)
	answerIpExtractorRe = regexp.MustCompile(`^.*[ \t]([^ \t]+)$`)
)

func ParseResponse(domain, message string) (string, []string, error) {
	answerSection := answerBlockRe.FindAllString(message, 1)
	if len(answerSection) == 0 {
		return "", []string{}, fmt.Errorf("No answers found")
		//return "", []string{}, fmt.Errorf("Failed to parse answer section for `" + domain + "` from `" + message + "`")
	}
	answers := strings.Split(answerSection[0], "\n")[1:]
	parsedDomain := answerDomainExtractorRe.ReplaceAllString(answers[0], "$1")
	if domain != parsedDomain { // Sanity check.
		return "", []string{}, fmt.Errorf("Expected parsed domain value to be '" + domain + "', but instead found '" + parsedDomain + "'")
	}
	ips := []string{}
	for _, answer := range answers {
		if strings.Contains(answer, "\tA\t") {
			ips = Append(ips, answerIpExtractorRe.ReplaceAllString(answer, "$1"))
		}
	}
	return domain, ips, nil
}

