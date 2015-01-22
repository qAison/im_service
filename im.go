package main

import "net"
import "fmt"
import "flag"
import "time"
import "runtime"
import "github.com/garyburd/redigo/redis"
import log "github.com/golang/glog"

var app_route *AppRoute
//var cluster *Cluster
var storage *Storage
var group_manager *GroupManager
var group_server *GroupServer
var state_center *StateCenter
var redis_pool *redis.Pool
var config *Config
var server_summary *ServerSummary

func init() {
	app_route = NewAppRoute()
	state_center = NewStateCenter()
	server_summary = NewServerSummary()
}

func handle_client(conn *net.TCPConn) {
	client := NewClient(conn)
	client.Run()
}

func handle_peer_client(conn *net.TCPConn) {
	return
	// conn.SetKeepAlive(true)
	// conn.SetKeepAlivePeriod(time.Duration(10 * 60 * time.Second))
	// client := NewPeerClient(conn)
	// client.Run()
}

func Listen(f func(*net.TCPConn), port int) {
	ip := net.ParseIP("0.0.0.0")
	addr := net.TCPAddr{ip, port, ""}

	listen, err := net.ListenTCP("tcp", &addr)
	if err != nil {
		fmt.Println("初始化失败", err.Error())
		return
	}
	for {
		client, err := listen.AcceptTCP()
		if err != nil {
			return
		}
		f(client)
	}

}
func ListenClient() {
	Listen(handle_client, config.port)
}

func NewRedisPool(server, password string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     100,
		MaxActive:   500,
		IdleTimeout: 480 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			if len(password) > 0 {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	if len(flag.Args()) == 0 {
		fmt.Println("usage: im config")
		return
	}

	config = read_cfg(flag.Args()[0])
	log.Infof("port:%d storage root:%s redis address:%s\n",
		config.port, config.storage_root, config.redis_address)

	redis_pool = NewRedisPool(config.redis_address, "")

//	cluster = NewCluster(config.peer_addrs)
//	cluster.Start()
	storage = NewStorage(config.storage_root)
	group_server = NewGroupServer(config.group_api_port)
	group_server.Start()
	group_manager = NewGroupManager()
	group_manager.Start()

	StartHttpServer(config.http_listen_address)

	go StartSocketIO(config.socket_io_address)
	ListenClient()

}
