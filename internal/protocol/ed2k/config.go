package ed2k

const (
	defaultServerList = "45.82.80.155:5687,176.123.5.89:4725,85.121.5.137:4232,176.123.2.239:4232,145.239.2.134:4661,91.208.162.87:4232,37.15.61.236:4232"
	defaultServerMet  = "ed2k://|serverlist|http://upd.emule-security.org/server.met|/"
	defaultNodesDat   = "https://upd.emule-security.org/nodes.dat"
)

type config struct {
	ListenPort int    `json:"listenPort"`
	UDPPort    int    `json:"udpPort"`
	ServerAddr string `json:"serverAddr"`
	ServerMet  string `json:"serverMet"`
	NodesDat   string `json:"nodesDat"`
}
