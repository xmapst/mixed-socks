package net

import "fmt"

func GenAddr(host string, port int) string {
	if host == "*" {
		return fmt.Sprintf(":%d", port)
	}
	return fmt.Sprintf("%s:%d", host, port)
}
