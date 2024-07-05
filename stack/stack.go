package stack

// LockFreeStack is an interface representing a thread-safe, lock-free stack.
type LockFreeStack[T any] interface {
	Push(T)
	Pop() T
}
