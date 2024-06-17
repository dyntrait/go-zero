//go:generate mockgen -package internal -destination etcdclient_mock.go -source etcdclient.go EtcdClient

package internal

import (
	"context"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

// EtcdClient interface represents an etcd client.
type EtcdClient interface {
	// grpc.ClientConn represents a virtual connection to a conceptual endpoint, to
	// perform RPCs.
	//
	// A ClientConn is free to have zero or more actual connections to the endpoint
	// based on configuration, load, etc. It is also free to determine which actual
	// endpoints to use and may change it every RPC, permitting client-side load
	// balancing.
	ActiveConnection() *grpc.ClientConn
	Close() error
	Ctx() context.Context
	Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
	Grant(ctx context.Context, ttl int64) (*clientv3.LeaseGrantResponse, error)
	KeepAlive(ctx context.Context, id clientv3.LeaseID) (<-chan *clientv3.LeaseKeepAliveResponse, error)
	Put(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error)
	Revoke(ctx context.Context, id clientv3.LeaseID) (*clientv3.LeaseRevokeResponse, error)
	Watch(ctx context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan
}
