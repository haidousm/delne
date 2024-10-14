package certloader

import (
	"crypto/tls"

	"github.com/foomo/simplecert"
)

type SSLConfig struct {
	Local    bool
	Email    string
	CacheDir string

	Domains []string
}

type DynamicCertLoader struct {
	certLoader *simplecert.CertReloader
}

func (dcl *DynamicCertLoader) NewCertLoader(sslConfig SSLConfig) (*simplecert.CertReloader, error) {
	scCfg := simplecert.Default
	scCfg.Local = sslConfig.Local

	scCfg.SSLEmail = sslConfig.Email
	scCfg.CacheDir = sslConfig.CacheDir

	scCfg.Domains = sslConfig.Domains

	certLoader, err := simplecert.Init(scCfg, nil)
	return certLoader, err
}

func (dcl *DynamicCertLoader) ReloadCerts(sslConfig SSLConfig) error {
	certLoader, err := dcl.NewCertLoader(sslConfig)
	dcl.certLoader = certLoader
	return err
}

func (dcl *DynamicCertLoader) GetCertificateFunc() func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return dcl.certLoader.GetCertificateFunc()
}
