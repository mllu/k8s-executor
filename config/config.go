package config

// Config struct contains kubewatch configuration
type Config struct {
	Namespace string `json:"namespace,omitempty"`
	Token     string `json:"token"`
	Channel   string `json:"channel"`
}

// New creates new config object
func New(namespace, token, channel string) (*Config, error) {
	c := &Config{
		Namespace: namespace,
		Token:     token,
		Channel:   channel,
	}
	return c, nil
}
