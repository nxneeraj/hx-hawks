package utils

import (
	"bufio"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
)

// ReadLines reads a file line by line and returns a slice of strings.
func ReadLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && (strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://")) {
			// Basic URL validation/normalization can be added here
			_, err := url.ParseRequestURI(line)
			if err == nil {
				lines = append(lines, line)
			} else {
				log.Printf("[!] Skipping invalid URL format: %s", line)
			}
		} else if line != "" {
			log.Printf("[!] Skipping line (missing http/https prefix): %s", line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// GetIP attempts to resolve the IP address for a given URL's host.
func GetIP(targetURL string) string {
	u, err := url.Parse(targetURL)
	if err != nil {
		return "" // Cannot parse URL
	}
	host := u.Hostname()
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return "" // Cannot resolve IP
	}
	// Return the first resolved IP (prefer IPv4 if available)
	for _, ip := range ips {
		if ip.To4() != nil {
			return ip.String()
		}
	}
	return ips[0].String() // Fallback to the first IP (likely IPv6)
}
