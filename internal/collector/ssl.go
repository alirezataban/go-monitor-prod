package collector

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

type SSL struct{}

func (c *SSL) Name() string { return "ssl" }

func (c *SSL) Collect(ctx *Context) (interface{}, error) {
	u, err := url.Parse(ctx.URL)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	if u.Scheme != "https" {
		return map[string]interface{}{
			"sslExists": false,
			"reason":    "not https",
		}, nil
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "443"
	}
	addr := net.JoinHostPort(host, port)

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: true,
	})
	if err != nil {
		return map[string]interface{}{
			"sslExists": false,
			"error":     err.Error(),
		}, nil
	}
	defer conn.Close()

	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return map[string]interface{}{"sslExists": false, "error": "no certificates"}, nil
	}

	cert := state.PeerCertificates[0]
	now := time.Now()
	daysLeft := int(cert.NotAfter.Sub(now).Hours() / 24)

	sans := cert.DNSNames
	if len(sans) == 0 {
		sans = []string{cert.Subject.CommonName}
	}

	issuer := cert.Issuer.CommonName
	if issuer == "" && len(cert.Issuer.Organization) > 0 {
		issuer = strings.Join(cert.Issuer.Organization, ", ")
	}

	wildcard := false
	for _, s := range sans {
		if strings.HasPrefix(s, "*.") {
			wildcard = true
			break
		}
	}

	selfSigned := cert.Subject.String() == cert.Issuer.String() ||
		(len(state.PeerCertificates) == 1 && isSelfSigned(cert))

	hsts := false
	if ctx.Response != nil {
		v, _ := ctx.Response.HeaderValue("strict-transport-security")
		hsts = v != ""
	}

	return map[string]interface{}{
		"sslExists":             true,
		"certificateExpiryDays": daysLeft,
		"certificateIssuer":     issuer,
		"subject":               cert.Subject.CommonName,
		"sans":                  sans,
		"wildcardCertificate":   wildcard,
		"selfSigned":            selfSigned,
		"tlsVersion":            tlsVersionString(state.Version),
		"cipherSuite":           tls.CipherSuiteName(state.CipherSuite),
		"hstsEnabled":           hsts,
		"notBefore":             cert.NotBefore.UTC().Format(time.RFC3339),
		"notAfter":              cert.NotAfter.UTC().Format(time.RFC3339),
		"valid":                 now.After(cert.NotBefore) && now.Before(cert.NotAfter),
		"expired":               now.After(cert.NotAfter),
		"serial":                cert.SerialNumber.String(),
		"ocspStapling":          len(state.OCSPResponse) > 0,
	}, nil
}

func isSelfSigned(cert *x509.Certificate) bool {
	return cert.CheckSignatureFrom(cert) == nil
}

func tlsVersionString(v uint16) string {
	switch v {
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS10:
		return "TLS 1.0"
	default:
		return fmt.Sprintf("0x%04x", v)
	}
}
