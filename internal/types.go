package internal

type Mutex interface {
	Lock()
	Unlock()
}
