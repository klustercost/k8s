package observer

type Controller interface {
	Run(int) error
	FriendlyName() string
}
