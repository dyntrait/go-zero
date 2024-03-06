package metric

import "github.com/zeromicro/go-zero/core/prometheus"

// A VectorOpts is a general configuration.
type VectorOpts struct {
	Namespace string
	Subsystem string
	Name      string
	Help      string
	Labels    []string //向量的分区
}

func update(fn func()) {
	if !prometheus.Enabled() {
		return
	}

	fn()
}
