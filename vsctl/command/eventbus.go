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

package command

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/fatih/color"
	"github.com/golang/protobuf/ptypes/empty"
	ctrlpb "github.com/linkall-labs/vsproto/pkg/controller"
	metapb "github.com/linkall-labs/vsproto/pkg/meta"
	"github.com/spf13/cobra"
)

func NewEventbusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eventbus sub-command",
		Short: "sub-commands for eventbus operations",
	}
	cmd.AddCommand(createEventbusCommand())
	cmd.AddCommand(deleteEventbusCommand())
	cmd.AddCommand(getEventbusInfoCommand())
	cmd.AddCommand(listEventbusInfoCommand())
	return cmd
}

func createEventbusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create a eventbus",
		Run: func(cmd *cobra.Command, args []string) {
			if eventbus == "" {
				cmdFailedf("the --name flag MUST> be set")
			}
			ctx := context.Background()
			grpcConn := mustGetLeaderControllerGRPCConn(ctx, cmd)
			defer func() {
				_ = grpcConn.Close()
			}()

			cli := ctrlpb.NewEventBusControllerClient(grpcConn)
			_, err := cli.CreateEventBus(ctx, &ctrlpb.CreateEventBusRequest{
				Name: eventbus,
			})
			if err != nil {
				cmdFailedf("create eventbus failed: %s", err)
			}
			color.Green("create eventbus: %s success\n", eventbus)
		},
	}
	cmd.Flags().StringVar(&eventbus, "name", "", "eventbus name to deleting")
	return cmd
}

func deleteEventbusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <eventbus-name> ",
		Short: "delete a eventbus",
		Run: func(cmd *cobra.Command, args []string) {
			if eventbus == "" {
				cmdFailedf("the --name flag MUST> be set")
			}
			ctx := context.Background()
			grpcConn := mustGetLeaderControllerGRPCConn(ctx, cmd)
			defer func() {
				_ = grpcConn.Close()
			}()

			cli := ctrlpb.NewEventBusControllerClient(grpcConn)
			_, err := cli.DeleteEventBus(ctx, &metapb.EventBus{Name: args[0]})
			if err != nil {
				cmdFailedf("delete eventbus failed: %s", err)
			}
			color.Green("delete eventbus: %s success\n", args[0])
		},
	}
	cmd.Flags().StringVar(&eventbus, "name", "", "eventbus name to deleting")
	return cmd
}

func getEventbusInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <eventbus-name> ",
		Short: "get the eventbus info",
		Run: func(cmd *cobra.Command, args []string) {
			if args[0] == "" {
				cmdFailedf("the eventbus must be set")
			}
			ctx := context.Background()
			grpcConn := mustGetLeaderControllerGRPCConn(ctx, cmd)
			defer func() {
				_ = grpcConn.Close()
			}()

			cli := ctrlpb.NewEventBusControllerClient(grpcConn)
			res, err := cli.GetEventBus(ctx, &metapb.EventBus{Name: args[0]})
			if err != nil {
				cmdFailedf("delete eventbus failed: %s", err)
			}
			data, _ := json.Marshal(res)
			var out bytes.Buffer
			_ = json.Indent(&out, data, "", "\t")
			color.Green("%s", out.String())
		},
	}
	cmd.Flags().StringVar(&eventbus, "name", "", "eventbus name to deleting")
	return cmd
}

func listEventbusInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list the eventbus",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			grpcConn := mustGetLeaderControllerGRPCConn(ctx, cmd)
			defer func() {
				_ = grpcConn.Close()
			}()

			cli := ctrlpb.NewEventBusControllerClient(grpcConn)
			res, err := cli.ListEventBus(ctx, &empty.Empty{})
			if err != nil {
				cmdFailedf("list eventbus failed: %s", err)
			}
			data, _ := json.Marshal(res)
			var out bytes.Buffer
			_ = json.Indent(&out, data, "", "\t")
			color.Green("%s", out.String())
		},
	}
	cmd.Flags().StringVar(&eventbus, "name", "", "eventbus name to deleting")
	return cmd
}
