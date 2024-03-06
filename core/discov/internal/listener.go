package internal

// Listener interface wraps the OnUpdate method. 处理etcd事件的
type Listener interface {
	OnUpdate(keys, values []string, newKey string)
}
