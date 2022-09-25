package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

type TLSConfig struct {
	CertFile      string // Path to the TLS certificate file.
	KeyFile       string // Path to the TLS key file.
	CAFile        string // Path to the TLS CA file.
	ServerAddress string // The address of the server.
	Server        bool   // Whether the configuration is for a server.
}

// SetupTLSConfig sets up a TLS configuration with the given CA File and Key File.
// If you'd like to know more about the TLS protocol, check out the following resources:
// - https://en.wikipedia.org/wiki/Transport_Layer_Security
// - https://codezine.jp/article/detail/15636
func SetupTLSConfig(cfg TLSConfig) (*tls.Config, error) {
	var err error
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS13}
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		tlsConfig.Certificates = make([]tls.Certificate, 1)
		tlsConfig.Certificates[0], err = tls.LoadX509KeyPair(
			cfg.CertFile,
			cfg.KeyFile,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load key pair: %v", err)
		}
	}
	if cfg.CAFile != "" {
		b, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %v", err)
		}
		ca := x509.NewCertPool()
		ok := ca.AppendCertsFromPEM([]byte(b))
		if !ok {
			return nil, fmt.Errorf(
				"failed to parse root certificate: %v",
				cfg.CAFile,
			)
		}
		if cfg.Server {
			tlsConfig.ClientCAs = ca
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		} else {
			tlsConfig.RootCAs = ca
		}
		tlsConfig.ServerName = cfg.ServerAddress
	}
	return tlsConfig, nil
}
