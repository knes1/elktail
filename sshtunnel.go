package main

import (
	"os"
	"fmt"
	"io"
	"net"
	"os/user"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"regexp"
	"strconv"
)

type Endpoint struct {
	Host string
	Port int
}

func (endpoint *Endpoint) String() string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

type SSHTunnel struct {
	Local  *Endpoint
	Server *Endpoint
	Remote *Endpoint

	Config *ssh.ClientConfig
}

func (tunnel *SSHTunnel) Start() error {
	listener, err := net.Listen("tcp", tunnel.Local.String())
	if err != nil {
		Error.Printf("SSH Tunnel: Failed to start server at %s. Error: %s", tunnel.Local.String(), err)
		return err
	}
	defer listener.Close()

	for {
		serverConn, err := ssh.Dial("tcp", tunnel.Server.String(), tunnel.Config)
		if err != nil {
			Error.Fatalf("SSH Tunnel: %s\n", err)
			return err
		}
		conn, err := listener.Accept()
		if err != nil {
			Error.Printf("SSH Tunnel: Failed to accept connection: %s", err)
			return err
		}
		Info.Print("SSH Tunnel: Accepted connection to forward to the tunnel...")
		go tunnel.forward(conn, serverConn)
	}
}

func (tunnel *SSHTunnel) forward(localConn net.Conn, sshServerConn *ssh.Client) {
	/*
	serverConn, err := ssh.Dial("tcp", tunnel.Server.String(), tunnel.Config)
	if err != nil {
		Error.Fatalf("SSH Tunnel: Server dial error: %s\n", err)
		return
	}*/

	remoteConn, err := sshServerConn.Dial("tcp", tunnel.Remote.String())
	if err != nil {
		Error.Fatalf("SSH Tunnel: Remote dial error: %s\n", err)
		return
	}

	copyConn := func(writer, reader net.Conn) {
		_, err:= io.Copy(writer, reader)
		if err != nil {
			Error.Fatalf("SSH Tunnel: Could not forward conenction: %s\n", err)
		}
	}

	go copyConn(localConn, remoteConn)
	go copyConn(remoteConn, localConn)
}

func SSHAgent() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}
	return nil
}

//
// sshHostDef user@sshhost.tld:port
// tunnelDef  local_port:remote_host:remote_port
//
func NewSSHTunnelFromHostStrings(sshHostDef string, tunnelDef string) *SSHTunnel {
	sshHostRegexp := regexp.MustCompile(`((\w*)@)?([^:@]+)(:(\d{2,5}))?`)
	match := sshHostRegexp.FindAllStringSubmatch(sshHostDef, -1)
	if len(match) == 0 {
		Error.Fatalf("SSH Tunnel: Failed to parse ssh host %s\n", sshHostDef)
	}
	result := match[0]
	sshUser := result[2]
	if sshUser == "" {
		osUser, _ := user.Current()
		sshUser = osUser.Username
	}
	sshHost := result[3]
	sshPort := parsePort(result[5], 22)

	Trace.Printf("SSH Tunnel: Server - User: %s, Host: %s, Port: %d\n", sshUser, sshHost, sshPort)

	//Setting up defaults
	localPort := 9199
	remotePort := 9200
	remoteHost := "localhost"

	tunnelRegexp := regexp.MustCompile(`((\d{2,5}):)?([^:@]+)(:(\d{2,5}))?`)
	match = tunnelRegexp.FindAllStringSubmatch(tunnelDef, -1)
	if len(match) == 0 {
		Trace.Print("SSH Tunnel: Failed to parse remote tunnel host/port, using defaults\n")
	} else {
		result = match[0]
		localPort = parsePort(result[2], 9199)
		remotePort = parsePort(result[5], 9200)
		remoteHost = result[3]
	}

	Trace.Printf("SSH Tunnel: Local port : %d, Remote Host: %s, Remote Port: %d\n", localPort, remoteHost, remotePort)

	return NewSSHTunnel(sshUser, sshHost, sshPort, localPort, remoteHost, remotePort)
}


func parsePort(portStr string, defaultPort int) int {
	if portStr != "" {
		port, err := strconv.Atoi(portStr)
		if (err != nil) {
			Error.Printf("SSH Tunnel: Reverting to port %d because given port was not numeric: %s\n", defaultPort, err)
			port = defaultPort
		}
		return port
	}
	return defaultPort
}

func passwordCallback() (string, error) {
	fmt.Println("Enter ssh password:")
	pwd := readPasswd();
	return pwd, nil;
}

func NewSSHTunnel(sshUser string, sshHost string, sshPort int, localPort int,
						remoteHost string, remotePort int) *SSHTunnel {
	localEndpoint := &Endpoint{
		Host: "localhost",
		Port: localPort,
	}

	serverEndpoint := &Endpoint{
		Host: sshHost,
		Port: sshPort,
	}

	remoteEndpoint := &Endpoint{
		Host: remoteHost,
		Port: remotePort,
	}

	sshConfig := &ssh.ClientConfig{
		User: sshUser,
		Auth: []ssh.AuthMethod{
			SSHAgent(),
			ssh.PasswordCallback(passwordCallback),
		},
	}

	return &SSHTunnel{
		Config: sshConfig,
		Local:  localEndpoint,
		Server: serverEndpoint,
		Remote: remoteEndpoint,
	}
}

