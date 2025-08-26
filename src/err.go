package main

const (
	ErrUnauthorized    = 0x400001
	ErrServerMisconfig = 0x500003
	ErrDomainRequired  = 0x400000
	ErrFileNotFound    = 0x400004
	ErrReadFileFailed  = 0x500000
	ErrSuccess         = 0x000000
)

var errorMap = map[int]struct {
	code int
	msg  string
}{
	ErrUnauthorized:    {0x400001, "Unauthorized"},
	ErrServerMisconfig: {0x500003, "Server misconfiguration"},
	ErrDomainRequired:  {0x400000, "Domain Required"},
	ErrFileNotFound:    {0x400004, "File Not Found"},
	ErrReadFileFailed:  {0x500000, "Read File Failed"},
	ErrSuccess:         {0x000000, "SUCCESS"},
}
