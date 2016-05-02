package client

import (
	"fmt"
	"io"
	"os"

	"src/github.com/docker/engine-api/client"

	"github.com/docker/docker/api"
	"github.com/docker/docker/cliconfig"
	"github.com/docker/docker/cliconfig/configfile"
	"github.com/docker/docker/cliconfig/credentials"
	"github.com/docker/docker/pkg/term"
)

// DockerCli represents the docker command line client.
// Instances of the client can be returned from NewDockerCli.
type DockerCli struct {
	// initialize closure.
	init func() error
	// configFile has the client configuration file
	configFile *configfile.ConfigFile
	// holds the input stream and sloser
	in io.ReadCloser
	// holds the output stream
	out io.Writer
	// holds the error stream
	err io.Writer
	// holds the key file as a string
	keyFile string
	// holds file descriptor of the clinent's STDIN (if valid)
	inFd uintptr
	// holds file descriptor of the clients STDOUT
	outFd uintptr
	// TTYとは : 実端末に接続しているかどうかということ
	// indicates whether the clients STDIN is a TTY
	isTerminalIn bool
	// indicates whether the clients STDOUT is a TTY
	isTerminalOut bool
	// the http client that performs all API operations
	client client.APIClient
	// holds the terminal state
	state *term.State
}

// Initialize calls the init function that will setup the configuration for the client
// such as the TLS, tcp and other parameters used to run the client.
func (cli *DockerCli) Initialize() error {
	if cli.init == nil {
		return nil
	}
	return cli.init()
}

func NewDockerCli(in io.ReadCloser, out, err io.Writer, clientFlags *cliflags.ClientFlags) *DockerCli {
	cli := &DockerCli{
		in:      in,
		out:     out,
		err:     err,
		keyFile: clientFlags.Common.TrustKey,
	}

	cli.init = func() error {
		clientFlags.PostParse()
		configFile, e := cliconfig.Load(cliconfig.ConfigDir())
		if e != nil {
			fmt.Fprintf(cli.err, "WARNINT: Error loading config file: %v\n", e)
		}
		if !configFile.ContainsAuth() {
			credentials.DetectDefaultStore(configFile)
		}
		cli.configFile = configFile

		host, err := getServerHost(clientFlags.Common.Hosts, clientFlags.Common.TLSOptions)
		if err != nil {
			return err
		}

		cutomHeaders := cli.configFile.HTTPHeaders
		if customHeaders == nil {
			customHeaders = map[string]string{}
		}
		customHeaders["User-Agent"] = clientUserAgent()

		verStr := api.DefaultVersion
		if tmpStr := os.Getenv("DOCKER_API_VERSION"); tmpStr != "" {
			verStr = tmpStr
		}

		httpClient, err := newHTTPClient(host, clientFlags.Common.TLSOptions)
		if err != nil {
			return err
		}

		client, err := client.NewClient(host, verStr, httpClient, customHeaders)
		if err != nil {
			return err
		}

		cli.client = client
		if cli.in != nil {
			cli.inFd, cli.isTerminalIn = term.GetFdInfo(cli.in)
		}
		if cli.out != nil {
			cli.outFd, cli.isTerminalOut = term.GetFdInfo(cli.out)
		}
		return nil
	}
	return cli
}
