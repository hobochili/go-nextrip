package nextrip

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Config struct {
	// Address is the address of the NexTrip API server
	Address string

	// Scheme is the URI scheme for the NexTrip API server
	Scheme string

	// BasePath is the base path of all API requests
	BasePath string

	// HttpClient is the client to use. Default will be
	// used if not provided.
	HttpClient *http.Client
}

// A Client manages communication with the Nextrip API.
type Client struct {
	config Config
}

type request struct {
	config *Config
	method string
	url    *url.URL
	params url.Values
	body   io.Reader
	header http.Header
}

func NewClient(config *Config) *Client {
	defaultConfig := &Config{
		Address:  "svc.metrotransit.org",
		Scheme:   "http",
		BasePath: "/NexTrip",
	}

	if len(config.Address) == 0 {
		config.Address = defaultConfig.Address
	}

	if len(config.Scheme) == 0 {
		config.Scheme = defaultConfig.Scheme
	}

	if config.HttpClient == nil {
		config.HttpClient = &http.Client{}
	}

	return &Client{config: *config}
}

func (c *Client) newRequest(method, path string) *request {
	p := []string{c.config.BasePath, path}

	r := &request{
		config: &c.config,
		method: method,

		url: &url.URL{
			Scheme: c.config.Scheme,
			Host:   c.config.Address,
			Path:   strings.Join(p, ""),
		},

		params: make(map[string][]string),
		header: make(http.Header),
	}

	return r
}

func (r *request) toHTTP() (*http.Request, error) {
	r.url.RawQuery = r.params.Encode()

	req, err := http.NewRequest(r.method, r.url.RequestURI(), r.body)
	if err != nil {
		return nil, err
	}

	req.URL.Host = r.url.Host
	req.URL.Scheme = r.url.Scheme
	req.Host = r.url.Host
	req.Header = r.header

	return req, nil
}

func (c *Client) doRequest(r *request) (*http.Response, error) {
	req, err := r.toHTTP()
	if err != nil {
		return nil, err
	}

	return c.config.HttpClient.Do(req)
}

func (c *Client) get(endpoint string, out interface{}) error {
	r := c.newRequest("GET", endpoint)

	resp, err := c.doRequest(r)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}

		return err
	}

	if resp.StatusCode != 200 {
		var buf bytes.Buffer

		io.Copy(&buf, resp.Body)
		resp.Body.Close()

		return fmt.Errorf("Unexpected response code: %d (%s)", resp.StatusCode, buf.Bytes())
	}

	defer resp.Body.Close()

	err = decodeBody(resp, out)
	if err != nil {
		return err
	}

	return nil
}

func decodeBody(resp *http.Response, out interface{}) error {
	dec := json.NewDecoder(resp.Body)
	return dec.Decode(out)
}
