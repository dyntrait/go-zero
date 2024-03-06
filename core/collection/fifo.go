package collection

import "sync"

// A Queue is a FIFO queue.
type Queue struct {
	lock     sync.Mutex
	elements []any
	size     int
	head     int //标记下个第一个要出队的元素所在的下标
	tail     int //标记下一个入队元素放在切片哪的小标
	count    int
}

// NewQueue returns a Queue object.
func NewQueue(size int) *Queue {
	return &Queue{
		elements: make([]any, size),
		size:     size,
	}
}

// Empty checks if q is empty.
func (q *Queue) Empty() bool {
	q.lock.Lock()
	empty := q.count == 0
	q.lock.Unlock()

	return empty
}

// Put puts element into q at the last position.
func (q *Queue) Put(element any) {
	q.lock.Lock()
	defer q.lock.Unlock()

	// 入队的速度赶上出队速度，切片是循环利用的
	if q.head == q.tail && q.count > 0 {
		//扩容
		nodes := make([]any, len(q.elements)+q.size)
		//把还没有出队的元素放到新队列头部
		//处理[head,旧队列尾部]元素
		copy(nodes, q.elements[q.head:])
		//[head,旧队列尾部]元素个数=len(q.elements)-q.head
		copy(nodes[len(q.elements)-q.head:], q.elements[:q.head])
		q.head = 0
		q.tail = len(q.elements)
		q.elements = nodes
	}

	q.elements[q.tail] = element
	q.tail = (q.tail + 1) % len(q.elements)
	q.count++
}

// Take takes the first element out of q if not empty.
func (q *Queue) Take() (any, bool) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.count == 0 {
		return nil, false
	}

	element := q.elements[q.head]
	q.head = (q.head + 1) % len(q.elements)
	q.count--

	return element, true
}
