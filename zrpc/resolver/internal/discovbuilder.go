package internal

import (
	"strings"

	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc/resolver/internal/targets"
	"google.golang.org/grpc/resolver"
)

type discovBuilder struct{}

func (b *discovBuilder) Build(target resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (
	resolver.Resolver, error) {
	// 首先从 target 中解析出 etcd 的地址，和服务对应的 key。
	//然后创建 etcd 连接，接着执行 update 方法，在 update 方法中，通过调用 cc.UpdateState 方法进行服务状态的更新。
	hosts := strings.FieldsFunc(targets.GetAuthority(target), func(r rune) bool {
		return r == EndpointSepChar
	})
	sub, err := discov.NewSubscriber(hosts, targets.GetEndpoints(target))
	if err != nil {
		return nil, err
	}
	/*
	func (ccr *ccResolverWrapper) UpdateState(s resolver.State) error {
	    ccr.incomingMu.Lock()
	    defer ccr.incomingMu.Unlock()
	    if ccr.done.HasFired() {
	        return nil
	    }
	    ccr.addChannelzTraceEvent(s)
	    ccr.curState = s
	    if err := ccr.cc.updateResolverState(ccr.curState, nil); err == balancer.ErrBadResolverState {
	        return balancer.ErrBadResolverState
	    }
	    return nil
	}
	 */

	update := func() {
		var addrs []resolver.Address
		for _, val := range subset(sub.Values(), subsetSize) {
			addrs = append(addrs, resolver.Address{
				Addr: val,
			})
		}
		if err := cc.UpdateState(resolver.State{
			Addresses: addrs,
		}); err != nil {
			logx.Error(err)
		}
	}
	//update 方法会被添加到事件监听中，当有 PUT 和 DELETE 事件触发，都会调用 update 方法进行服务状态的更新，
	//事件监听是通过 etcd 的 Watch 机制实现
	sub.AddListener(update)
	update()

	return &nopResolver{cc: cc}, nil
}

func (b *discovBuilder) Scheme() string {
	return DiscovScheme
}
