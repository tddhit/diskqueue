package service

import "github.com/tddhit/diskqueue/pb"

type pqueue []*diskqueuepb.Message

func (q pqueue) Len() int { return len(q) }
func (q pqueue) Less(i, j int) bool {
	return q[i].GetTimestamp() < q[j].GetTimestamp()
}
func (q pqueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }

func (q *pqueue) Push(x interface{}) {
	*q = append(*q, x.(*diskqueuepb.Message))
}

func (q *pqueue) Pop() interface{} {
	old := *q
	n := len(old)
	x := old[n-1]
	*q = old[0 : n-1]
	return x
}
