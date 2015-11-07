// Copyright (c) 2015 Uber Technologies, Inc.

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

package testutils

import (
	"flag"
	"testing"
	"time"

	"github.com/uber/tchannel-go"
)

var connectionLog = flag.Bool("connectionLog", false, "Enables connection logging in tests")

// Default service names for the test channels.
const (
	DefaultServerName = "testService"
	DefaultClientName = "testService-client"
)

// ChannelOpts contains options to create a test channel using WithServer
type ChannelOpts struct {
	tchannel.ChannelOptions

	// ServiceName defaults to DefaultServerName or DefaultClientName.
	ServiceName string

	// LogVerification contains options for controlling the log verification.
	LogVerification LogVerification

	// optFn is run with the channel options before creating the channel.
	optFn func(*ChannelOpts)
}

// LogVerification contains options to control the log verification.
type LogVerification struct {
	Disabled bool
}

// SetServiceName sets ServiceName.
func (o *ChannelOpts) SetServiceName(svcName string) *ChannelOpts {
	o.ServiceName = svcName
	return o
}

// SetStatsReporter sets StatsReporter in ChannelOptions.
func (o *ChannelOpts) SetStatsReporter(statsReporter tchannel.StatsReporter) *ChannelOpts {
	o.StatsReporter = statsReporter
	return o
}

// SetTraceReporter sets TraceReporter in ChannelOptions.
func (o *ChannelOpts) SetTraceReporter(traceReporter tchannel.TraceReporter) *ChannelOpts {
	o.TraceReporter = traceReporter
	return o
}

// SetFramePool sets FramePool in DefaultConnectionOptions.
func (o *ChannelOpts) SetFramePool(framePool tchannel.FramePool) *ChannelOpts {
	o.DefaultConnectionOptions.FramePool = framePool
	return o
}

// SetTimeNow sets TimeNow in ChannelOptions.
func (o *ChannelOpts) SetTimeNow(timeNow func() time.Time) *ChannelOpts {
	o.TimeNow = timeNow
	return o
}

func defaultString(v string, defaultValue string) string {
	if v == "" {
		return defaultValue
	}
	return v
}

func getChannelOptions(opts *ChannelOpts) *tchannel.ChannelOptions {
	if opts.Logger == nil && *connectionLog {
		opts.Logger = tchannel.SimpleLogger
	}
	if opts.optFn != nil {
		opts.optFn(opts)
	}
	return &opts.ChannelOptions
}

// NewOpts returns a new ChannelOpts that can be used in a chained fashion.
func NewOpts() *ChannelOpts { return &ChannelOpts{} }

// DefaultOpts will return opts if opts is non-nil, NewOpts otherwise.
func DefaultOpts(opts *ChannelOpts) *ChannelOpts {
	if opts == nil {
		return NewOpts()
	}
	return opts
}

// WrapLogger wraps the given logger with extra verification.
func (v *LogVerification) WrapLogger(t testing.TB, l tchannel.Logger) tchannel.Logger {
	return errorLogger{l, t}
}
