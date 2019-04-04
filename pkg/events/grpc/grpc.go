// Copyright © 2019 The Things Network Foundation, The Things Industries B.V.
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

// Package grpc contains an implementation of the EventsServer, which is used to
// stream all events published for a set of identifiers.
package grpc

import (
	"context"
	"runtime"

	grpc_runtime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"go.thethings.network/lorawan-stack/pkg/auth/rights"
	"go.thethings.network/lorawan-stack/pkg/events"
	"go.thethings.network/lorawan-stack/pkg/rpcmiddleware/warning"
	"go.thethings.network/lorawan-stack/pkg/ttnpb"
	"google.golang.org/grpc"
)

const workersPerCPU = 2

// NewEventsServer returns a new EventsServer on the given PubSub.
func NewEventsServer(ctx context.Context, pubsub events.PubSub) *EventsServer {
	srv := &EventsServer{
		ctx:    ctx,
		events: make(events.Channel, 256),
		filter: events.NewIdentifierFilter(),
	}

	hander := events.ContextHandler(ctx, srv.events)
	pubsub.Subscribe("**", hander)
	go func() {
		<-ctx.Done()
		pubsub.Unsubscribe("**", hander)
		close(srv.events)
	}()

	for i := 0; i < runtime.NumCPU()*workersPerCPU; i++ {
		go func() {
			for evt := range srv.events {
				proto, err := events.Proto(evt)
				if err != nil {
					return
				}
				srv.filter.Notify(marshaledEvent{
					Event: evt,
					proto: proto,
				})
			}
		}()
	}

	return srv
}

type marshaledEvent struct {
	events.Event
	proto *ttnpb.Event
}

// EventsServer streams events from a PubSub over gRPC.
type EventsServer struct {
	ctx    context.Context
	events events.Channel
	filter events.IdentifierFilter
}

// Stream implements the EventsServer interface.
func (srv *EventsServer) Stream(req *ttnpb.StreamEventsRequest, stream ttnpb.Events_StreamServer) (err error) {
	ctx := stream.Context()

	if len(req.Identifiers) == 0 {
		return nil
	}

	for _, entityIDs := range req.Identifiers {
		switch ids := entityIDs.Identifiers().(type) {
		case *ttnpb.ApplicationIdentifiers:
			err = rights.RequireApplication(ctx, *ids, ttnpb.RIGHT_APPLICATION_ALL)
		case *ttnpb.ClientIdentifiers:
			err = rights.RequireClient(ctx, *ids, ttnpb.RIGHT_CLIENT_ALL)
		case *ttnpb.EndDeviceIdentifiers:
			err = rights.RequireApplication(ctx, ids.ApplicationIdentifiers, ttnpb.RIGHT_APPLICATION_ALL)
		case *ttnpb.GatewayIdentifiers:
			err = rights.RequireGateway(ctx, *ids, ttnpb.RIGHT_GATEWAY_ALL)
		case *ttnpb.OrganizationIdentifiers:
			err = rights.RequireOrganization(ctx, *ids, ttnpb.RIGHT_ORGANIZATION_ALL)
		case *ttnpb.UserIdentifiers:
			err = rights.RequireUser(ctx, *ids, ttnpb.RIGHT_USER_ALL)
		}
		if err != nil {
			return err
		}
	}

	ch := make(events.Channel, 8)
	handler := events.ContextHandler(ctx, ch)
	srv.filter.Subscribe(ctx, req, handler)
	defer srv.filter.Unsubscribe(ctx, req, handler)

	if req.Tail > 0 || req.After != nil {
		warning.Add(ctx, "Historical events not implemented")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case evt := <-ch:
			marshaled := evt.(marshaledEvent)
			if err := stream.Send(marshaled.proto); err != nil {
				return err
			}
		}
	}
}

// StreamApplication implements the EventsServer interface.
func (srv *EventsServer) StreamApplication(req *ttnpb.StreamApplicationEventsRequest, stream ttnpb.Events_StreamApplicationServer) error {
	streamReq := &ttnpb.StreamEventsRequest{Tail: req.Tail, After: req.After}
	for _, id := range req.ApplicationIDs {
		streamReq.Identifiers = append(streamReq.Identifiers, ttnpb.ApplicationIdentifiers{ApplicationID: id}.EntityIdentifiers())
	}
	return srv.Stream(streamReq, stream)
}

// StreamClient implements the EventsServer interface.
func (srv *EventsServer) StreamClient(req *ttnpb.StreamClientEventsRequest, stream ttnpb.Events_StreamClientServer) error {
	streamReq := &ttnpb.StreamEventsRequest{Tail: req.Tail, After: req.After}
	for _, id := range req.ClientIDs {
		streamReq.Identifiers = append(streamReq.Identifiers, ttnpb.ClientIdentifiers{ClientID: id}.EntityIdentifiers())
	}
	return srv.Stream(streamReq, stream)
}

// StreamEndDevice implements the EventsServer interface.
func (srv *EventsServer) StreamEndDevice(req *ttnpb.StreamEndDeviceEventsRequest, stream ttnpb.Events_StreamEndDeviceServer) error {
	streamReq := &ttnpb.StreamEventsRequest{Tail: req.Tail, After: req.After}
	for _, id := range req.DeviceIDs {
		streamReq.Identifiers = append(streamReq.Identifiers, ttnpb.EndDeviceIdentifiers{
			ApplicationIdentifiers: ttnpb.ApplicationIdentifiers{ApplicationID: req.ApplicationIDs},
			DeviceID:               id,
		}.EntityIdentifiers())
	}
	return srv.Stream(streamReq, stream)
}

// StreamGateway implements the EventsServer interface.
func (srv *EventsServer) StreamGateway(req *ttnpb.StreamGatewayEventsRequest, stream ttnpb.Events_StreamGatewayServer) error {
	streamReq := &ttnpb.StreamEventsRequest{Tail: req.Tail, After: req.After}
	for _, id := range req.GatewayIDs {
		streamReq.Identifiers = append(streamReq.Identifiers, ttnpb.GatewayIdentifiers{GatewayID: id}.EntityIdentifiers())
	}
	return srv.Stream(streamReq, stream)
}

// StreamOrganization implements the EventsServer interface.
func (srv *EventsServer) StreamOrganization(req *ttnpb.StreamOrganizationEventsRequest, stream ttnpb.Events_StreamOrganizationServer) error {
	streamReq := &ttnpb.StreamEventsRequest{Tail: req.Tail, After: req.After}
	for _, id := range req.OrganizationIDs {
		streamReq.Identifiers = append(streamReq.Identifiers, ttnpb.OrganizationIdentifiers{OrganizationID: id}.EntityIdentifiers())
	}
	return srv.Stream(streamReq, stream)
}

// StreamUser implements the EventsServer interface.
func (srv *EventsServer) StreamUser(req *ttnpb.StreamUserEventsRequest, stream ttnpb.Events_StreamUserServer) error {
	streamReq := &ttnpb.StreamEventsRequest{Tail: req.Tail, After: req.After}
	for _, id := range req.UserIDs {
		streamReq.Identifiers = append(streamReq.Identifiers, ttnpb.UserIdentifiers{UserID: id}.EntityIdentifiers())
	}
	return srv.Stream(streamReq, stream)
}

// Roles implements rpcserver.Registerer.
func (srv *EventsServer) Roles() []ttnpb.PeerInfo_Role {
	return nil
}

// RegisterServices implements rpcserver.Registerer.
func (srv *EventsServer) RegisterServices(s *grpc.Server) {
	ttnpb.RegisterEventsServer(s, srv)
}

// RegisterHandlers implements rpcserver.Registerer.
func (srv *EventsServer) RegisterHandlers(s *grpc_runtime.ServeMux, conn *grpc.ClientConn) {
	ttnpb.RegisterEventsHandler(srv.ctx, s, conn)
}
