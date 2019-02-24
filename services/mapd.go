package services

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bitmark-inc/bitmark-node/fault"
	"github.com/bitmark-inc/logger"
)

var (
	ErrMapdIsNotRunning = fault.InvalidError("mapd is not running")
	ErrMapdIsRunning    = fault.InvalidError("mapd is running")
)

var client = &http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	},
}

type Mapd struct {
	sync.RWMutex
	initialised bool
	log         *logger.L
	rootPath    string
	configFile  string
	network     string
	running     bool
}

type Node struct {
	PublicKey string    `json:"publicKey"`
	Ip        string    `json:"ip"`
	Height    uint64    `json:"height"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	Timestamp time.Time `json:"timestamp"`
	TimeDiff  string    `json:"timediff"`
}

func NewMapd() *Mapd {
	return &Mapd{}
}

func (mapd *Mapd) GetPath() string {
	return mapd.rootPath
}

func (mapd *Mapd) GetNetwork() string {
	if len(mapd.network) == 0 {
		return "bitmark"
	}
	return mapd.network
}

func (mapd *Mapd) Initialise(rootPath string) error {
	mapd.Lock()
	defer mapd.Unlock()

	if mapd.initialised {
		return fault.ErrAlreadyInitialised
	}

	mapd.rootPath = rootPath
	mapd.log = logger.New("service-mapd")

	mapd.running = false

	// all data initialised
	mapd.initialised = true
	return nil
}

func (mapd *Mapd) Finalise() error {
	mapd.Lock()
	defer mapd.Unlock()

	if !mapd.initialised {
		return fault.ErrNotInitialised
	}

	mapd.initialised = false
	return nil
}

func (mapd *Mapd) IsRunning() bool {
	return mapd.running
}

func (mapd *Mapd) Status() map[string]interface{} {
	return map[string]interface{}{
		"started": mapd.running,
		"running": mapd.running,
		"ipPort":  os.Getenv("MAP_IP_PORT"),
	}
}

func (mapd *Mapd) SetNetwork(network string) {
	if mapd.running {
		mapd.Stop()
	}
	mapd.network = network
	switch network {
	case "testing":
		mapd.configFile = filepath.Join(mapd.rootPath, "testing/mapd.conf")
	case "bitmark":
		fallthrough
	default:
		mapd.configFile = filepath.Join(mapd.rootPath, "bitmark/mapd.conf")
	}
}

func (mapd *Mapd) Start() error {
	if mapd.running {
		mapd.mapdLog("Start mapd failed: %v", ErrMapdIsRunning)
		return ErrMapdIsRunning
	}
	mapd.running = true
	mapd.mapdLog("mapd.running")

	go func() {
		for mapd.running {
			time.Sleep(time.Second * 3)

			var detailReply DetailReply
			err := mapd.getBitmarkdApi("details", &detailReply)
			if err != nil {
				continue
			}

			publicKey := detailReply.PublicKey

			var peerReplies []PeerReply
			err = mapd.getBitmarkdApi("peers", &peerReplies)
			if err != nil {
				continue
			}

			for i := range peerReplies {
				var height uint64
				mapd.mapdLog("%v", peerReplies[i])
				if peerReplies[i].PublicKey == publicKey {
					height = detailReply.Blocks.Local
				}

				ips := peerReplies[i].Listeners
				var ip string
				for j := range ips {
					// ipv4 checker
					if strings.Contains(ips[j], ".") {
						// remove port
						ip = strings.Split(ips[j], ":")[0]
						break
					}
				}

				// skip this peer if ip is empty
				if len(ip) == 0 {
					continue
				}

				register := "http://" + os.Getenv("MAP_IP_PORT") + "/register"
				n := &Node{
					PublicKey: peerReplies[i].PublicKey,
					Ip:        ip,
					Height:    height,
					Lat:       0, // map server will get Lat, Lng
					Lng:       0,
					Timestamp: peerReplies[i].Timestamp,
					TimeDiff:  "", // map server will calculates TimeDiff
				}
				data, _ := json.Marshal(n)
				body := bytes.NewReader([]byte(data))
				resp, err := http.Post(register, "application/json", body)
				if err != nil {
					mapd.mapdLog("unable to post register info")
					continue
				}
				defer resp.Body.Close()
			}
		}
	}()
	return nil

}

func (mapd *Mapd) Stop() error {
	if !mapd.running {
		mapd.mapdLog("Stop mapd failed: %v", ErrMapdIsNotRunning)
		return ErrMapdIsNotRunning
	}
	mapd.mapdLog("mapd.stopped")

	mapd.running = false
	return nil
}

func (mapd *Mapd) getBitmarkdApi(api string, reply interface{}) error {
	resp, err := client.Get("https://127.0.0.1:2131/bitmarkd/" + api)
	if err != nil {
		mapd.mapdLog("unable to get bitmark api, retry it")
		time.Sleep(time.Second * 5)
		return err
	}
	defer resp.Body.Close()
	bb := bytes.Buffer{}
	io.Copy(&bb, resp.Body)

	if resp.StatusCode != http.StatusOK {
		mapd.mapdLog("unable to get bitmark api. message: %s", bb.String())
		return err
	}

	d := json.NewDecoder(&bb)

	if err := d.Decode(reply); err != nil {
		mapd.mapdLog("fail to read bitmark api response. error: %s\n", err.Error())
		return err
	}
	return nil
}

func (mapd *Mapd) mapdLog(format string, a ...interface{}) {
	f, err := os.OpenFile("/.config/mapd.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	//set output of logs to f
	log.SetOutput(f)
	log.Printf(format+"\n", a...)
}
