// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"errors"
	"strconv"
	"time"

	"github.com/yarpc/yab/encoding"
)

// Options are parsed from flags using go-flags.
type Options struct {
	ROpts          RequestOptions   `group:"request"`
	TOpts          TransportOptions `group:"transport"`
	BOpts          BenchmarkOptions `group:"benchmark"`
	DisplayVersion bool             `long:"version" description:"Displays the application version"`
	ManPage        bool             `long:"man-page" hidden:"yes" description:"Print yab's man page to stdout"`
}

// RequestOptions are request related options
type RequestOptions struct {
	Encoding    encoding.Encoding `short:"e" long:"encoding" description:"The encoding of the data, options are: Thrift, JSON, raw. Defaults to Thrift if the method contains '::' or a Thrift file is specified"`
	ThriftFile  string            `short:"t" long:"thrift" description:"Path of the .thrift file"`
	MethodName  string            `short:"m" long:"method" description:"The full Thrift method name (Svc::Method) to invoke"`
	RequestJSON string            `short:"r" long:"request" unquote:"false" description:"The request body, in JSON or YAML format"`
	RequestFile string            `short:"f" long:"file" description:"Path of a file containing the request body in JSON or YAML"`
	HeadersJSON string            `long:"headers" unquote:"false" description:"The headers in JSON or YAML format"`
	HeadersFile string            `long:"headers-file" description:"Path of a file containing the headers in JSON or YAML"`
	Health      bool              `long:"health" description:"Hit the health endpoint, Meta::health"`
	Timeout     timeMillisFlag    `long:"timeout" default:"1s" description:"The timeout for each request. E.g., 100ms, 0.5s, 1s. If no unit is specified, milliseconds are assumed."`

	// Thrift options
	ThriftDisableEnvelopes bool `long:"disable-thrift-envelope" description:"Disables Thrift envelopes (disabled by default for TChannel)"`
	ThriftMultiplexed      bool `long:"multiplexed-thrift" description:"Enables the Thrift TMultiplexedProtocol used by services that host multiple Thrift services on a single endpoint."`

	// These are aliases for tcurl compatibility.
	Aliases struct {
		Endpoint stringAlias `long:"endpoint" hidden:"true"`
		Arg1     stringAlias `short:"1" long:"arg1" hidden:"true"`
		Arg2     stringAlias `short:"2" long:"arg2" unquote:"false" hidden:"true"`
		Arg3     stringAlias `short:"3" long:"arg3" unquote:"false" hidden:"true"`
		Body     stringAlias `long:"body" unquote:"false" hidden:"true"`
		JSON     bool        `long:"json" hidden:"true"`
		Raw      bool        `long:"raw" hidden:"true"`
	}
}

// TransportOptions are transport related options.
type TransportOptions struct {
	ServiceName      string            `short:"s" long:"service" description:"The TChannel/Hyperbahn service name"`
	HostPorts        []string          `short:"p" long:"peer" description:"The host:port of the service to call"`
	HostPortFile     string            `short:"P" long:"peer-list" description:"Path of a JSON or YAML file containing a list of host:ports"`
	CallerOverride   string            `long:"caller" description:"Caller will override the default caller name (which is yab-$USER)."`
	TransportOptions map[string]string `long:"topt" description:"Custom options for the specific transport being used"`

	// benchmarking is a private flag set when a transport is required for benchmarking.
	benchmarking bool

	// Alias for tcurl compatibility.
	Hostlist stringAlias `short:"H" long:"hostlist" hidden:"true"`
}

// BenchmarkOptions are benchmark-specific options
type BenchmarkOptions struct {
	MaxRequests int           `short:"n" long:"max-requests" default:"1000000" description:"The maximum number of requests to make"`
	MaxDuration time.Duration `short:"d" long:"max-duration" default:"0s" description:"The maximum amount of time to run the benchmark for"`

	// NumCPUs is the value for GOMAXPROCS. The default value of 0 will not update GOMAXPROCS.
	NumCPUs int `long:"cpus" description:"The number of OS threads"`

	Connections    int `long:"connections" description:"The number of TCP connections to use"`
	WarmupRequests int `long:"warmup" description:"The number of requests to make to warmup each connection" default:"10"`
	Concurrency    int `long:"concurrency" default:"1" description:"The number of concurrent calls per connection"`
	RPS            int `long:"rps" default:"0" description:"Limit on the number of requests per second. The default (0) is no limit."`

	// Benchmark metrics can optionally be reported via statsd.
	StatsdHostPort string `long:"statsd" description:"Optional host:port of a StatsD server to report metrics"`
}

func newOptions() *Options {
	var opts Options
	aliases := &opts.ROpts.Aliases
	aliases.Arg1.dest = &opts.ROpts.MethodName
	aliases.Endpoint.dest = &opts.ROpts.MethodName
	aliases.Arg2.dest = &opts.ROpts.HeadersJSON
	aliases.Arg3.dest = &opts.ROpts.RequestJSON
	aliases.Body.dest = &opts.ROpts.RequestJSON

	opts.TOpts.Hostlist.dest = &opts.TOpts.HostPortFile
	return &opts
}

type timeMillisFlag time.Duration

func (t *timeMillisFlag) setDuration(d time.Duration) {
	*t = timeMillisFlag(d)
}

func (t timeMillisFlag) Duration() time.Duration {
	return time.Duration(t)
}

func (t *timeMillisFlag) UnmarshalFlag(value string) error {
	valueInt, err := strconv.Atoi(value)
	if err == nil {
		// We received a number without a unit, assume milliseconds.
		t.setDuration(time.Duration(valueInt) * time.Millisecond)
		return nil
	}

	d, err := time.ParseDuration(value)
	if err != nil {
		return err
	}

	t.setDuration(d)
	return nil
}

var errStringAliasMissing = errors.New("string alias missing destination")

type stringAlias struct {
	dest *string
}

func (s *stringAlias) UnmarshalFlag(value string) error {
	if s.dest == nil {
		return errStringAliasMissing
	}
	*s.dest = value
	return nil
}

func setEncodingOptions(opts *Options) {
	if opts.ROpts.Aliases.JSON {
		opts.ROpts.Encoding = encoding.JSON
	}
	if opts.ROpts.Aliases.Raw {
		opts.ROpts.Encoding = encoding.Raw
	}
}
