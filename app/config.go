package app

import "github.com/spf13/viper"

type Config struct {
	Proxy      Proxy      `mapstructure:"proxy"`
	Management Management `mapstructure:"management"`
}

type Proxy struct {
	PasswordsFile string      `mapstructure:"passwords_file"`
	TLS           ProxyTLS    `mapstructure:"tls"`
	Server        ProxyServer `mapstructure:"server"`
	Worker        ProxyWorker `mapstructure:"worker"`
}

type ProxyTLS struct {
	Certificate    string `mapstructure:"certificate"`
	CertificateKey string `mapstructure:"certificate_key"`
}

type ProxyServer struct {
	Ports    []int `mapstructure:"ports"`
	TLSPorts []int `mapstructure:"tls_ports"`
}

type ProxyWorker struct {
	PoolCount   int `mapstructure:"pool_count"`
	ReadBuffer  int `mapstructure:"read_buffer"`
	WriteBuffer int `mapstructure:"write_buffer"`
}

type Management struct {
	UnixSocket string `mapstructure:"unix_socket"`
}

func LoadConfig() (*Config, error) {
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func init() {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("/etc/gilgamesh")
	viper.AddConfigPath(".")
}
