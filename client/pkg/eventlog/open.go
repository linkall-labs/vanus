// Copyright 2022 Linkall Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eventlog

import "time"

const (
	defaultPollingTimeout = 3000 // in milliseconds.
)

type Option func(*ReaderConfig)

func PollingTimeout(d time.Duration) Option {
	return func(cfg *ReaderConfig) {
		cfg.PollingTimeout = uint32(d.Milliseconds())
	}
}

func DisablePolling() Option {
	return func(cfg *ReaderConfig) {
		cfg.PollingTimeout = 0
	}
}

type ReaderConfig struct {
	PollingTimeout uint32
}

// OpenWriter open a Writer of EventLog identified by vrn.
func OpenWriter(vrn string) (LogWriter, error) {
	el, err := Get(vrn)
	if err != nil {
		return nil, err
	}
	defer Put(el)

	w, err := el.Writer()
	if err != nil {
		return nil, err
	}

	return w, nil
}

// OpenReader open a Reader of EventLog identified by vrn.
func OpenReader(vrn string, opts ...Option) (LogReader, error) {
	el, err := Get(vrn)
	if err != nil {
		return nil, err
	}
	defer Put(el)

	cfg := ReaderConfig{PollingTimeout: defaultPollingTimeout}
	for _, option := range opts {
		option(&cfg)
	}

	r, err := el.Reader(cfg)
	if err != nil {
		return nil, err
	}

	return r, nil
}
