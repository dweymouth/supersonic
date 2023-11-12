package mediaprovider

type Server interface {
	Ping() bool
	Login(username, password string) error
	MediaProvider() MediaProvider
}
