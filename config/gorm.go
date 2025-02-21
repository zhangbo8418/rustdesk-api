package config

const (
	TypeSqlite = "sqlite"
	TypeMysql  = "mysql"
)

type Gorm struct {
	Type         string `mapstructure:"type"`
	MaxIdleConns int    `mapstructure:"max-idle-conns"`
	MaxOpenConns int    `mapstructure:"max-open-conns"`
	Dbpath string `mapstructure:"dbpath"
}

type Mysql struct {
	Addr     string `mapstructure:"addr"`
	Socket string `mapstructure:"socket"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Dbname   string `mapstructure:"dbname"`
}
