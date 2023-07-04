package common

import (
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	gatewayId             *identity.X509Identity
	gatewaySign           identity.Sign
	gatewayTLSCredentials *credentials.TransportCredentials
)

func CreateGrpcConnection(endpoint string) (*grpc.ClientConn, error) {
	// Check TLS credential was created
	if gatewayTLSCredentials == nil {
		gatewayServerName := os.Getenv("FABRIC_GATEWAY_NAME")

		cred, err := createTransportCredential(GetTLSCert(), gatewayServerName)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create tls credentials")
		}

		gatewayTLSCredentials = &cred
	}

	// Create client grpc connection
	return grpc.Dial(endpoint, grpc.WithTransportCredentials(*gatewayTLSCredentials))
}

func CreateGatewayConnection(grpcConn *grpc.ClientConn) (*client.Gateway, error) {
	// Check identity was created
	if gatewayId == nil {
		id, err := newIdentity(GetTLSCert(), GetMSPID())
		if err != nil {
			return nil, errors.Wrap(err, "failed to create new identity")
		}
		gatewayId = id
	}

	// Check sign function was created
	if gatewaySign == nil {
		sign, err := newSign(GetTLSKey())
		if err != nil {
			return nil, errors.Wrap(err, "failed to create new sign function")
		}

		gatewaySign = sign
	}

	// Create a Gateway connection for a specific client identity.
	return client.Connect(
		gatewayId,
		client.WithSign(gatewaySign),
		client.WithClientConnection(grpcConn),

		// Default timeouts for different gRPC calls
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
}

// Create transport credential
func createTransportCredential(tlsCertPath, serverName string) (credentials.TransportCredentials, error) {
	certificate, err := loadCertificate(tlsCertPath)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(certificate)
	return credentials.NewClientTLSFromCert(certPool, serverName), nil
}

// Creates a client identity for a gateway connection using an X.509 certificate.
func newIdentity(certPath, mspID string) (*identity.X509Identity, error) {
	certificate, err := loadCertificate(certPath)
	if err != nil {
		return nil, err
	}

	id, err := identity.NewX509Identity(mspID, certificate)
	if err != nil {
		return nil, err
	}

	return id, nil
}

// Creates a function that generates a digital signature from a message digest using a private key.
func newSign(keyPath string) (identity.Sign, error) {
	privateKeyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read private key file")
	}

	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return nil, err
	}

	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create signer function")
	}

	return sign, nil
}

func loadCertificate(filename string) (*x509.Certificate, error) {
	certificatePEM, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}
	return identity.CertificateFromPEM(certificatePEM)
}
