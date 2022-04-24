// Copyright 2022 Linkall Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package transport

import (
	// standard libraries.
	"context"

	// third-party libraries.
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	// first-party libraries.
	"github.com/linkall-labs/raft/raftpb"
	vsraftpb "github.com/linkall-labs/vsproto/pkg/raft"
)

type peer struct {
	addr   string
	msgc   chan *raftpb.Message
	ctx    context.Context
	cancel context.CancelFunc
}

// Make sure peer implements Multiplexer.
var _ Multiplexer = (*peer)(nil)

func newPeer(ctx context.Context, endpoint string, callback string) *peer {
	ctx, cancel := context.WithCancel(ctx)

	p := &peer{
		addr:   endpoint,
		msgc:   make(chan *raftpb.Message),
		ctx:    ctx,
		cancel: cancel,
	}

	go p.run(callback)

	return p
}

func (p *peer) run(callback string) {
	opts := []grpc.DialOption{grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials())}

	preface := raftpb.Message{
		Context: []byte(callback),
	}
	var stream vsraftpb.RaftServer_SendMsssageClient

loop:
	for {
		var err error
		select {
		case msg := <-p.msgc:
			// TODO(james.yin): reconnect
			if stream == nil {
				if stream, err = p.connect(opts...); err != nil {
					break
				}
				if err = stream.Send(&preface); err != nil {
					// TODO(james.yin): handle err
				}
			}
			err = stream.Send(msg)
			if err != nil {
				// TODO(james.yin): handle error
				break
			}
		case <-p.ctx.Done():
			break loop
		}
	}

	if stream != nil {
		stream.CloseAndRecv()
	}
}

func (p *peer) Close() {
	p.cancel()
}

func (p *peer) Send(msg *raftpb.Message) {
	// TODO(james.yin):
	select {
	case p.msgc <- msg:
	case <-p.ctx.Done():
	}
}

func (p *peer) Sendv(msgs []*raftpb.Message) {
	for _, msg := range msgs {
		p.Send(msg)
	}
}

func (p *peer) connect(opts ...grpc.DialOption) (vsraftpb.RaftServer_SendMsssageClient, error) {
	conn, err := grpc.DialContext(context.TODO(), p.addr, opts...)
	if err != nil {
		return nil, err
	}
	client := vsraftpb.NewRaftServerClient(conn)
	stream, err := client.SendMsssage(context.TODO())
	if err != nil {
		return nil, err
	}
	return stream, nil
}
