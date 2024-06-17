package stat

import "container/heap"

type taskHeap []Task

func (h *taskHeap) Len() int {
	return len(*h)
}

func (h *taskHeap) Less(i, j int) bool {
	return (*h)[i].Duration < (*h)[j].Duration
}

func (h *taskHeap) Swap(i, j int) {
	(*h)[i], (*h)[j] = (*h)[j], (*h)[i]
}

func (h *taskHeap) Push(x any) { //在尾部追加
	*h = append(*h, x.(Task))
}

func (h *taskHeap) Pop() any { //在尾部拿走
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func topK(all []Task, k int) []Task { //返回all里面前k最小的值
	h := new(taskHeap)
	heap.Init(h)

	for _, each := range all {
		if h.Len() < k {
			heap.Push(h, each)
		} else if (*h)[0].Duration < each.Duration {
			heap.Pop(h)
			heap.Push(h, each)
		}
	}

	return *h
}
