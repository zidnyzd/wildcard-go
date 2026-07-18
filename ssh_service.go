package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHConfig struct {
	Host     string
	Port     int
	User     string
	Password string
}

// shellEscape wraps a string in single quotes for safe shell interpolation
func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func (s *SSHConfig) CreateChallengeFile(token, body string) map[string]interface{} {
	if s.Host == "" || s.Password == "" {
		return map[string]interface{}{"success": false, "error": "SSH credentials belum diisi"}
	}

	config := &ssh.ClientConfig{
		User: s.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(s.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return map[string]interface{}{"success": false, "error": fmt.Sprintf("SSH connection failed: %v", err)}
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return map[string]interface{}{"success": false, "error": fmt.Sprintf("SSH session failed: %v", err)}
	}
	defer session.Close()

	// Safe shell escaping for token and body
	cmd := fmt.Sprintf(
		"mkdir -p /var/www/acme/.well-known/acme-challenge && echo -n %s > /var/www/acme/.well-known/acme-challenge/%s",
		shellEscape(body), shellEscape(token),
	)
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return map[string]interface{}{"success": false, "error": fmt.Sprintf("SSH exec failed: %v - %s", err, string(output))}
	}

	return map[string]interface{}{"success": true, "message": "Challenge file created on VPS"}
}

func (s *SSHConfig) TestConnection() map[string]interface{} {
	if s.Host == "" {
		return map[string]interface{}{"success": false, "error": "SSH host kosong"}
	}
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return map[string]interface{}{"success": false, "error": fmt.Sprintf("Connection failed: %v", err)}
	}
	conn.Close()
	return map[string]interface{}{"success": true}
}
