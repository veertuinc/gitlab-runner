package network

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jpillora/backoff"
	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-runner/common"
)

type requestCredentials interface {
	GetURL() string
	GetToken() string
	GetTLSCAFile() string
	GetTLSCertFile() string
	GetTLSKeyFile() string
}

var (
	dialer = net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	backOffDelayMin    = 100 * time.Millisecond
	backOffDelayMax    = 60 * time.Second
	backOffDelayFactor = 2.0
	backOffDelayJitter = true
)

type client struct {
	http.Client
	url             *url.URL
	caFile          string
	certFile        string
	keyFile         string
	caData          []byte
	skipVerify      bool
	updateTime      time.Time
	lastUpdate      string
	requestBackOffs map[string]*backoff.Backoff
	lock            sync.Mutex
}

type ResponseTLSData struct {
	CAChain  string
	CertFile string
	KeyFile  string
}

func (n *client) getLastUpdate() string {
	return n.lastUpdate
}

func (n *client) setLastUpdate(headers http.Header) {
	if lu := headers.Get("X-GitLab-Last-Update"); len(lu) > 0 {
		n.lastUpdate = lu
	}
}

func (n *client) ensureTLSConfig() {
	// certificate got modified
	if stat, err := os.Stat(n.caFile); err == nil && n.updateTime.Before(stat.ModTime()) {
		n.Transport = nil
	}

	// client certificate got modified
	if stat, err := os.Stat(n.certFile); err == nil && n.updateTime.Before(stat.ModTime()) {
		n.Transport = nil
	}

	// client private key got modified
	if stat, err := os.Stat(n.keyFile); err == nil && n.updateTime.Before(stat.ModTime()) {
		n.Transport = nil
	}

	// create or update transport
	if n.Transport == nil {
		n.updateTime = time.Now()
		n.createTransport()
	}
}

func (n *client) addTLSCA(tlsConfig *tls.Config) {
	// load TLS CA certificate
	if file := n.caFile; file != "" && !n.skipVerify {
		logrus.Debugln("Trying to load", file, "...")

		data, err := ioutil.ReadFile(file)
		if err == nil {
			pool, err := x509.SystemCertPool()
			if err != nil {
				logrus.Warningln("Failed to load system CertPool:", err)
			}
			if pool == nil {
				pool = x509.NewCertPool()
			}
			if pool.AppendCertsFromPEM(data) {
				tlsConfig.RootCAs = pool
				n.caData = data
			} else {
				logrus.Errorln("Failed to parse PEM in", n.caFile)
			}
		} else {
			if !os.IsNotExist(err) {
				logrus.Errorln("Failed to load", n.caFile, err)
			}
		}
	}
}

func (n *client) addTLSAuth(tlsConfig *tls.Config) {
	// load TLS client keypair
	if cert, key := n.certFile, n.keyFile; cert != "" && key != "" {
		logrus.Debugln("Trying to load", cert, "and", key, "pair...")

		certificate, err := tls.LoadX509KeyPair(cert, key)
		if err == nil {
			tlsConfig.Certificates = []tls.Certificate{certificate}
			tlsConfig.BuildNameToCertificate()
		} else {
			if !os.IsNotExist(err) {
				logrus.Errorln("Failed to load", cert, key, err)
			}
		}
	}
}

func (n *client) createTransport() {
	// create reference TLS config
	tlsConfig := tls.Config{
		MinVersion:         tls.VersionTLS10,
		InsecureSkipVerify: n.skipVerify,
	}

	n.addTLSCA(&tlsConfig)
	n.addTLSAuth(&tlsConfig)

	// create transport
	n.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: func(network, addr string) (net.Conn, error) {
			logrus.Debugln("Dialing:", network, addr, "...")
			return dialer.Dial(network, addr)
		},
		TLSClientConfig:       &tlsConfig,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 10 * time.Minute,
	}
	n.Timeout = common.DefaultNetworkClientTimeout
}

func (n *client) getCAChain(tls *tls.ConnectionState) string {
	if len(n.caData) != 0 {
		return string(n.caData)
	}

	if tls == nil {
		return ""
	}

	// Don't reorder certificates by putting them directly into the map
	var certificates []*x509.Certificate
	seenCertificates := make(map[string]bool, 0)

	for _, verifiedChain := range tls.VerifiedChains {
		for _, certificate := range verifiedChain {
			signature := hex.EncodeToString(certificate.Signature)
			if seenCertificates[signature] {
				continue
			}

			seenCertificates[signature] = true
			certificates = append(certificates, certificate)
		}
	}

	out := bytes.NewBuffer(nil)
	for _, certificate := range certificates {
		if err := pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: certificate.Raw}); err != nil {
			logrus.Warn("Failed to encode certificate from chain:", err)
		}
	}

	return out.String()
}

func (n *client) ensureBackoff(method, uri string) *backoff.Backoff {
	n.lock.Lock()
	defer n.lock.Unlock()

	key := fmt.Sprintf("%s_%s", method, uri)
	if n.requestBackOffs[key] == nil {
		n.requestBackOffs[key] = &backoff.Backoff{
			Min:    backOffDelayMin,
			Max:    backOffDelayMax,
			Factor: backOffDelayFactor,
			Jitter: backOffDelayJitter,
		}
	}

	return n.requestBackOffs[key]
}

func (n *client) backoffRequired(res *http.Response) bool {
	return res.StatusCode >= 400 && res.StatusCode < 600
}

func (n *client) doBackoffRequest(req *http.Request) (res *http.Response, err error) {
	res, err = n.Do(req)
	if err != nil {
		err = fmt.Errorf("couldn't execute %v against %s: %v", req.Method, req.URL, err)
		return
	}

	backoffDelay := n.ensureBackoff(req.Method, req.RequestURI)
	if n.backoffRequired(res) {
		time.Sleep(backoffDelay.Duration())
	} else {
		backoffDelay.Reset()
	}

	return
}

func (n *client) do(uri, method string, request io.Reader, requestType string, headers http.Header) (res *http.Response, err error) {
	url, err := n.url.Parse(uri)
	if err != nil {
		return
	}

	req, err := http.NewRequest(method, url.String(), request)
	if err != nil {
		err = fmt.Errorf("failed to create NewRequest: %v", err)
		return
	}

	if headers != nil {
		req.Header = headers
	}

	if request != nil {
		req.Header.Set("Content-Type", requestType)
		req.Header.Set("User-Agent", common.AppVersion.UserAgent())
	}

	n.ensureTLSConfig()

	res, err = n.doBackoffRequest(req)
	return
}

func (n *client) doJSON(uri, method string, statusCode int, request interface{}, response interface{}) (int, string, ResponseTLSData, *http.Response) {
	var body io.Reader

	if request != nil {
		requestBody, err := json.Marshal(request)
		if err != nil {
			return -1, fmt.Sprintf("failed to marshal project object: %v", err), ResponseTLSData{}, nil
		}
		body = bytes.NewReader(requestBody)
	}

	headers := make(http.Header)
	if response != nil {
		headers.Set("Accept", "application/json")
	}

	res, err := n.do(uri, method, body, "application/json", headers)
	if err != nil {
		return -1, err.Error(), ResponseTLSData{}, nil
	}
	defer res.Body.Close()
	defer io.Copy(ioutil.Discard, res.Body)

	if res.StatusCode == statusCode {
		if response != nil {
			isApplicationJSON, err := isResponseApplicationJSON(res)
			if !isApplicationJSON {
				return -1, err.Error(), ResponseTLSData{}, nil
			}

			d := json.NewDecoder(res.Body)
			err = d.Decode(response)
			if err != nil {
				return -1, fmt.Sprintf("Error decoding json payload %v", err), ResponseTLSData{}, nil
			}
		}
	}

	n.setLastUpdate(res.Header)

	TLSData := ResponseTLSData{
		CAChain:  n.getCAChain(res.TLS),
		CertFile: n.certFile,
		KeyFile:  n.keyFile,
	}

	return res.StatusCode, res.Status, TLSData, res
}

func isResponseApplicationJSON(res *http.Response) (result bool, err error) {
	contentType := res.Header.Get("Content-Type")

	mimetype, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false, fmt.Errorf("Content-Type parsing error: %v", err)
	}

	if mimetype != "application/json" {
		return false, fmt.Errorf("Server should return application/json. Got: %v", contentType)
	}

	return true, nil
}

func fixCIURL(url string) string {
	url = strings.TrimRight(url, "/")
	if strings.HasSuffix(url, "/ci") {
		url = strings.TrimSuffix(url, "/ci")
	}
	return url
}

func (n *client) findCertificate(certificate *string, base string, name string) {
	if *certificate != "" {
		return
	}
	path := filepath.Join(base, name)
	if _, err := os.Stat(path); err == nil {
		*certificate = path
	}
}

func newClient(requestCredentials requestCredentials) (c *client, err error) {
	url, err := url.Parse(fixCIURL(requestCredentials.GetURL()) + "/api/v4/")
	if err != nil {
		return
	}

	if url.Scheme != "http" && url.Scheme != "https" {
		err = errors.New("only http or https scheme supported")
		return
	}

	c = &client{
		url:             url,
		caFile:          requestCredentials.GetTLSCAFile(),
		certFile:        requestCredentials.GetTLSCertFile(),
		keyFile:         requestCredentials.GetTLSKeyFile(),
		requestBackOffs: make(map[string]*backoff.Backoff),
	}

	host := strings.Split(url.Host, ":")[0]
	if CertificateDirectory != "" {
		c.findCertificate(&c.caFile, CertificateDirectory, host+".crt")
		c.findCertificate(&c.certFile, CertificateDirectory, host+".auth.crt")
		c.findCertificate(&c.keyFile, CertificateDirectory, host+".auth.key")
	}

	return
}
