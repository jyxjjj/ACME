package main

import (
	"strings"
)

func isRootDomain(domain string) bool {
	return !strings.HasPrefix(domain, "*.")
}
