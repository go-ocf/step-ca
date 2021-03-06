package authority

import (
	"context"
	"crypto/tls"
	"crypto/x509"

	"github.com/smallstep/certificates/authority"
	stepAuthority "github.com/smallstep/certificates/authority"
	stepProvisioner "github.com/smallstep/certificates/authority/provisioner"
	"github.com/smallstep/certificates/db"
	"github.com/smallstep/cli/crypto/pemutil"
	"github.com/smallstep/cli/crypto/tlsutil"
	"github.com/smallstep/cli/crypto/x509util"
	"golang.org/x/crypto/ssh"
)

const (
	legacyAuthority = "step-certificate-authority"
)

// Authority implements the Certificate Authority internal interface.
type Authority struct {
	config               *Config
	stepAuth             *stepAuthority.Authority
	intermediateIdentity *x509util.Identity
}

type Option interface{}

// WrapperOption sets options to the Authority.
type WrapperOption func(*Authority)

// WithDatabase sets an already initialized authority database to a new
// authority. This option is intended to be use on graceful reloads.
func WithDatabase(db db.AuthDB) stepAuthority.Option {
	return stepAuthority.WithDatabase(db)
}

// New creates and initiates a new Authority type.
func New(config *Config, opts ...Option) (*Authority, error) {
	var stepOpts []stepAuthority.Option
	var wrapOpts []WrapperOption
	for _, o := range opts {
		switch v := o.(type) {
		case WrapperOption:
			wrapOpts = append(wrapOpts, v)
		case stepAuthority.Option:
			stepOpts = append(stepOpts, v)
		}
	}

	stepAuth, err := stepAuthority.New(config.Config, stepOpts...)
	if err != nil {
		return nil, err
	}

	var intermediateIdentity *x509util.Identity

	// Decrypt and load intermediate public / private key pair.
	if len(config.Password) > 0 {
		intermediateIdentity, err = x509util.LoadIdentityFromDisk(
			config.IntermediateCert,
			config.IntermediateKey,
			pemutil.WithPassword([]byte(config.Password)),
		)
		if err != nil {
			return nil, err
		}
	} else {
		intermediateIdentity, err = x509util.LoadIdentityFromDisk(config.IntermediateCert, config.IntermediateKey)
		if err != nil {
			return nil, err
		}
	}

	return &Authority{
		config:               config,
		stepAuth:             stepAuth,
		intermediateIdentity: intermediateIdentity,
	}, nil
}

// GetDatabase returns the authority database. If the configuration does not
// define a database, GetDatabase will return a db.SimpleDB instance.
func (a *Authority) GetDatabase() db.AuthDB {
	return a.stepAuth.GetDatabase()
}

// Shutdown safely shuts down any clients, databases, etc. held by the Authority.
func (a *Authority) Shutdown() error {
	return a.stepAuth.Shutdown()
}

func (a *Authority) Authorize(ctx context.Context, ott string) ([]stepProvisioner.SignOption, error) {
	return a.stepAuth.Authorize(ctx, ott)
}

func (a *Authority) AuthorizeSign(ott string) ([]stepProvisioner.SignOption, error) {
	return a.stepAuth.AuthorizeSign(ott)
}

func (a *Authority) GetTLSOptions() *tlsutil.TLSOptions {
	return a.stepAuth.GetTLSOptions()
}

func (a *Authority) Root(shasum string) (*x509.Certificate, error) {
	return a.stepAuth.Root(shasum)
}

func (a *Authority) Sign(cr *x509.CertificateRequest, opts stepProvisioner.Options, signOpts ...stepProvisioner.SignOption) (*x509.Certificate, *x509.Certificate, error) {
	if a.isOCF(signOpts) {
		return a.OCFSign(cr, opts, signOpts...)
	}
	return a.stepAuth.Sign(cr, opts, signOpts...)
}

func (a *Authority) Renew(peer *x509.Certificate) (*x509.Certificate, *x509.Certificate, error) {
	return a.stepAuth.Renew(peer)
}

func (a *Authority) LoadProvisionerByCertificate(c *x509.Certificate) (stepProvisioner.Interface, error) {
	return a.stepAuth.LoadProvisionerByCertificate(c)
}

func (a *Authority) LoadProvisionerByID(ID string) (stepProvisioner.Interface, error) {
	return a.stepAuth.LoadProvisionerByID(ID)
}

func (a *Authority) GetProvisioners(cursor string, limit int) (stepProvisioner.List, string, error) {
	return a.stepAuth.GetProvisioners(cursor, limit)
}

func (a *Authority) Revoke(opts *authority.RevokeOptions) error {
	return a.stepAuth.Revoke(opts)
}

func (a *Authority) GetEncryptedKey(kid string) (string, error) {
	return a.stepAuth.GetEncryptedKey(kid)
}

func (a *Authority) GetRoots() (federation []*x509.Certificate, err error) {
	return a.stepAuth.GetRoots()
}

func (a *Authority) GetFederation() ([]*x509.Certificate, error) {
	return a.stepAuth.GetFederation()
}

func (a *Authority) GetTLSCertificate() (*tls.Certificate, error) {
	return a.stepAuth.GetTLSCertificate()
}

func (a *Authority) SignSSH(key ssh.PublicKey, opts stepProvisioner.SSHOptions, signOpts ...stepProvisioner.SignOption) (*ssh.Certificate, error) {
	return a.stepAuth.SignSSH(key, opts, signOpts)
}

func (a *Authority) GetRootCertificates() []*x509.Certificate {
	return a.stepAuth.GetRootCertificates()
}

func (a *Authority) SignSSHAddUser(key ssh.PublicKey, subject *ssh.Certificate) (*ssh.Certificate, error) {
	return a.stepAuth.SignSSHAddUser(key, subject)
}

func LoadConfiguration(filename string) (*Config, error) {
	config, err := stepAuthority.LoadConfiguration(filename)
	if err != nil {
		return nil, err
	}
	return &Config{
		Config: config,
	}, nil
}
