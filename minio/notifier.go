package minio

type Notifier interface {
	Notify(text string) error
}
