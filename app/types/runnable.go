package types

type Runnable interface {
	Run() error
	Shutdown() error
}
