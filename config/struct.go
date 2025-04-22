package config

// Config 主配置
type Config struct {
	DBGateway   DBGatewayConfig         `toml:"dbgateway"`
	FileAPI     FileAPIConfig           `toml:"server"`
	Storage     StorageConfig           `toml:"storage"`
	FileStorage []FileStorageNodeConfig `toml:"filestorage"`
}

// FileAPIConfig grpc Server 配置
type FileAPIConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
	Log  bool   `toml:"log"`
}

// DBGatewayConfig grpc DBGateway 配置
type DBGatewayConfig struct {
	Host    string `toml:"host"`
	Port    int    `toml:"port"`
	ConnNum int    `toml:"conn_num"`
	Timeout int    `toml:"sql_timeout"`
}

// FileStorageNodeConfig grpc FileStorage 存储配置 列表
type FileStorageNodeConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
	ID   int    `toml:"id"`
}

// StorageConfig 基本存储配置
type StorageConfig struct {
	CheckTime int `toml:"check_time"`
	BlockSize int `toml:"blocksize"`
	Timeout   int `toml:"timeout"`
}
