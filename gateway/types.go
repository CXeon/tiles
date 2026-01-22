package gateway

type API struct {
	Env      string //环境 比如Test 测试环境，Dev 开发环境，Prod 生产环境
	Cluster  string //集群 比如China 中国集群，America 美国集群，Europe 欧洲集群
	Company  string //公司名称 比如 TalentLimited
	Project  string //项目名称
	Service  string //服务的名称
	Color    string //染色 比如Red
	Protocol string //通信协议 比如http
	Ip       string //地址
	Port     uint16 //端口
}
