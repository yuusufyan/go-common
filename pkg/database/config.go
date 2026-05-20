package database

type DBConfig struct {
	Host     string
	User     string
	Password string
	DBName   string
	Port            int
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime int // In minutes
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}
