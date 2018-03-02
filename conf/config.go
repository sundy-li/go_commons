package conf

var (
	baseResource *ResourceConfig
)

type ResourceConfig struct {
	Mongo map[string]*DbConfig
	Redis map[string]*DbConfig
	Mysql map[string]*DbConfig
	Kafka map[string]*DbConfig
	Es    map[string]*DbConfig
	Pg    map[string]*DbConfig
	Etcd  map[string]*DbConfig
}

type DbConfig struct {
	Host string
	Port int

	Db string

	User string
	Pwd  string
}

func init() {
	baseResource = &ResourceConfig{
		Mongo: make(map[string]*DbConfig),
		Redis: make(map[string]*DbConfig),
		Mysql: make(map[string]*DbConfig),
		Kafka: make(map[string]*DbConfig),
		Es:    make(map[string]*DbConfig),
		Pg:    make(map[string]*DbConfig),
	}
}
func InitResConfig(res *ResourceConfig) {
	baseResource = res
}

func GetResConfig() *ResourceConfig {
	return baseResource
}
