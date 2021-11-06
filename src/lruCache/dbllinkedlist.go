package lruCache

type Node struct {
	Prev  *Node
	Next  *Node
	Value interface{}
}

type DblLinkedList struct {
	Head   *Node
	Tail   *Node
	Length int
}

func (dll *DblLinkedList) SetListHead(n *Node) {
	if dll.Head == nil {
		dll.Head = n
		dll.Tail = n
	} else {
		prevHead := dll.Head
		newHead := n
		newHead.Next = prevHead
		prevHead.Prev = newHead
		dll.Head = newHead
	}
	dll.Length++
}

func (dll *DblLinkedList) DeleteListTail() {
	if dll.Tail != nil && dll.Tail.Prev != nil {
		dll.Tail = dll.Tail.Prev
		dll.Tail.Next = nil
	} else if dll.Tail != nil {
		dll.Tail = dll.Tail.Prev
		dll.Head = nil
	}
	dll.Length--
}

func (dll *DblLinkedList) DeleteListHead() {
	if dll.Head != nil && dll.Head.Next != nil {
		dll.Head = dll.Head.Next
		dll.Head.Prev = nil
	} else if dll.Head != nil {
		dll.Head = dll.Head.Next
		dll.Tail = nil
	}
	dll.Length--
}
