package platformimpl

type Config struct {
	Token string `json:"token" required:"true"`

	// Host is like https://gitlab.com
	Host string `json:"host" required:"true"`
}
