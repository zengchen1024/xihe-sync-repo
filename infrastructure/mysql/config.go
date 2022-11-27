package mysql

type Config struct {
	Conn            string `json:"conn" required:"true"`
	ConnMaxLifetime int    `json:"conn_max_life_time"`
	MaxOpenConns    int    `json:"max_open_conns"`
	MaxIdleConns    int    `json:"max_idle_conns"`

	TableName string `json:"table_name"   required:"true"`
}

func (cfg *Config) SetDefault() {
	cfg.ConnMaxLifetime = 900
	cfg.MaxOpenConns = 3000
	cfg.MaxIdleConns = 30
}
