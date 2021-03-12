package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/connector/http"
	"github.com/shipyard-run/connector/protos/shipyard"
	"github.com/shipyard-run/connector/remote"
	"github.com/shipyard-run/shipyard/pkg/server"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func newConnectorRunCommand() *cobra.Command {
	var grpcBindAddr string
	var httpBindAddr string
	var apiBindAddr string
	var pathCertRoot string
	var pathCertServer string
	var pathKeyServer string
	var logLevel string
	var logFile string

	connectorRunCmd := &cobra.Command{
		Use:   "run",
		Short: "Run the connector",
		Long:  `Runs the connector with the given options`,
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {

			lo := hclog.LoggerOptions{}
			lo.Level = hclog.LevelFromString(logLevel)

			if logFile != "" {
				// create a new log file
				if _, err := os.Stat(utils.GetConnectorLogFile()); err == nil {
					os.RemoveAll(utils.GetConnectorLogFile())
				}

				f, err := os.Create(utils.GetConnectorLogFile())
				if err != nil {
					return fmt.Errorf("unable to create log file %s: %s", utils.GetConnectorLogFile(), err)
				}
				defer f.Close()

				lo.Output = f // set the logger to use file output
			}

			l := hclog.New(&lo)

			grpcServer := grpc.NewServer()
			s := remote.New(l.Named("grpc_server"), nil, nil, nil)

			// do we need to set up the server to use TLS?
			if pathCertServer != "" && pathKeyServer != "" && pathCertRoot != "" {
				certificate, err := tls.LoadX509KeyPair(pathCertServer, pathKeyServer)
				if err != nil {
					return fmt.Errorf("could not load server key pair: %s", err)
				}

				// Create a certificate pool from the certificate authority
				certPool := x509.NewCertPool()
				ca, err := ioutil.ReadFile(pathCertRoot)
				if err != nil {
					return fmt.Errorf("could not read ca certificate: %s", err)
				}

				// Append the client certificates from the CA
				if ok := certPool.AppendCertsFromPEM(ca); !ok {
					return errors.New("failed to append client certs")
				}

				creds := credentials.NewTLS(&tls.Config{
					ClientAuth:   tls.RequireAndVerifyClientCert,
					Certificates: []tls.Certificate{certificate},
					ClientCAs:    certPool,
				})

				grpcServer = grpc.NewServer(grpc.Creds(creds))
				s = remote.New(l.Named("grpc_server"), certPool, &certificate, nil)
			}

			shipyard.RegisterRemoteConnectionServer(grpcServer, s)

			// create a listener for the server
			l.Info("Starting gRPC server", "bind_addr", grpcBindAddr)
			lis, err := net.Listen("tcp", grpcBindAddr)
			if err != nil {
				l.Error("Unable to list on address", "bind_addr", grpcBindAddr)
				os.Exit(1)
			}

			// start the gRPC server
			go grpcServer.Serve(lis)

			// start the http server in the background
			l.Info("Starting HTTP server", "bind_addr", httpBindAddr)
			httpS := http.NewLocalServer(pathCertRoot, pathCertServer, pathKeyServer, grpcBindAddr, httpBindAddr, l)

			err = httpS.Serve()
			l.Info("Started")
			if err != nil {
				l.Error("Unable to start HTTP server", "error", err)
				os.Exit(1)
			}

			// start the API server
			// we should look at merging the connector server and the API server
			l.Info("Starting API server", "bind_addr", apiBindAddr)
			api := server.New(apiBindAddr, l.Named("api_server"))
			api.Start()

			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)
			signal.Notify(c, os.Kill)

			// Block until a signal is received.
			sig := <-c
			log.Println("Got signal:", sig)

			s.Shutdown()

			return nil
		},
	}

	connectorRunCmd.Flags().StringVarP(&grpcBindAddr, "grpc-bind", "", ":9090", "Bind address for the gRPC API")
	connectorRunCmd.Flags().StringVarP(&httpBindAddr, "http-bind", "", ":9091", "Bind address for the HTTP API")
	connectorRunCmd.Flags().StringVarP(&apiBindAddr, "api-bind", "", ":9092", "Bind address for the API Server")
	connectorRunCmd.Flags().StringVarP(&pathCertRoot, "root-cert-path", "", "", "Path for the PEM encoded TLS root certificate")
	connectorRunCmd.Flags().StringVarP(&pathCertServer, "server-cert-path", "", "", "Path for the servers PEM encoded TLS certificate")
	connectorRunCmd.Flags().StringVarP(&pathKeyServer, "server-key-path", "", "", "Path for the servers PEM encoded Private Key")
	connectorRunCmd.Flags().StringVarP(&logLevel, "log-level", "", "info", "Log output level [debug, trace, info]")
	connectorRunCmd.Flags().StringVarP(&logFile, "log-file", "", "./connector.log", "Log file for connector logs")

	return connectorRunCmd
}
