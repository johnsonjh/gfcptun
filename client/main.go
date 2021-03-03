package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"github.com/johnsonjh/gfcptun/generic"
	smux "github.com/johnsonjh/gfsmux"
)

const (
	// maximum supported smux version
	maxSmuxVer = 2
)

// VERSION is injected by buildflags
var VERSION = "SELFBUILD"

// handleClient aggregates connection p1 on mux with 'writeLock'
func handleClient(session *smux.Session, p1 net.Conn, quiet bool) {
	logln := func(v ...interface{}) {
		if !quiet {
			log.Println(v...)
		}
	}
	defer p1.Close()
	p2, err := session.OpenStream()
	if err != nil {
		logln(err)
		return
	}

	defer p2.Close()

	logln("stream opened", "in:", p1.RemoteAddr(), "out:", fmt.Sprint(p2.RemoteAddr(), "(", p2.ID(), ")"))
	defer logln("stream closed", "in:", p1.RemoteAddr(), "out:", fmt.Sprint(p2.RemoteAddr(), "(", p2.ID(), ")"))

	// start tunnel & wait for tunnel termination
	streamCopy := func(dst io.Writer, src io.ReadCloser) {
		if _, err := generic.Copy(dst, src); err != nil {
			// report protocol error
			if err == smux.ErrInvalidProtocol {
				log.Println("smux", err, "in:", p1.RemoteAddr(), "out:", fmt.Sprint(p2.RemoteAddr(), "(", p2.ID(), ")"))
			}
		}
		p1.Close()
		p2.Close()
	}

	go streamCopy(p1, p2)
	streamCopy(p2, p1)
}

func checkError(err error) {
	if err != nil {
		log.Printf("%+v\n", err)
		os.Exit(-1)
	}
}

type timedSession struct {
	session    *smux.Session
	expiryDate time.Time
}

func main() {
	rand.Seed(int64(time.Now().Nanosecond()))
	if VERSION == "SELFBUILD" {
		// add more log flags for debugging
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	myApp := cli.NewApp()
	myApp.Name = "gfcptun"
	myApp.Usage = "client(with SMUX)"
	myApp.Version = VERSION
	myApp.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "localaddr,l",
			Value: ":12948",
			Usage: "local listen address",
		},
		cli.StringFlag{
			Name:  "remoteaddr, r",
			Value: "vps:29900",
			Usage: "gfcp server address",
		},
		cli.StringFlag{
			Name:   "key",
			Value:  "it's a secrect",
			Usage:  "pre-shared secret between client and server",
			EnvVar: "GFCPTUN_KEY",
		},
		cli.StringFlag{
			Name:  "mode",
			Value: "fast",
			Usage: "profiles: fast3, fast2, fast, normal, manual",
		},
		cli.IntFlag{
			Name:  "conn",
			Value: 1,
			Usage: "set num of UDP connections to server",
		},
		cli.IntFlag{
			Name:  "autoexpire",
			Value: 0,
			Usage: "set auto expiration time(in seconds) for a single UDP connection, 0 to disable",
		},
		cli.IntFlag{
			Name:  "scavengettl",
			Value: 600,
			Usage: "set how long an expired connection can live (in seconds)",
		},
		cli.IntFlag{
			Name:  "mtu",
			Value: 1350,
			Usage: "set maximum transmission unit for UDP packets",
		},
		cli.IntFlag{
			Name:  "sndwnd",
			Value: 128,
			Usage: "set send window size(num of packets)",
		},
		cli.IntFlag{
			Name:  "rcvwnd",
			Value: 512,
			Usage: "set receive window size(num of packets)",
		},
		cli.IntFlag{
			Name:  "datashard,ds",
			Value: 10,
			Usage: "set reed-solomon erasure coding - datashard",
		},
		cli.IntFlag{
			Name:  "parityshard,ps",
			Value: 3,
			Usage: "set reed-solomon erasure coding - parityshard",
		},
		cli.IntFlag{
			Name:  "dscp",
			Value: 0,
			Usage: "set DSCP(6bit)",
		},
		cli.BoolFlag{
			Name:  "nocomp",
			Usage: "disable compression",
		},
		cli.BoolFlag{
			Name:   "acknodelay",
			Usage:  "flush ack immediately when a packet is received",
			Hidden: true,
		},
		cli.IntFlag{
			Name:   "nodelay",
			Value:  0,
			Hidden: true,
		},
		cli.IntFlag{
			Name:   "interval",
			Value:  50,
			Hidden: true,
		},
		cli.IntFlag{
			Name:   "resend",
			Value:  0,
			Hidden: true,
		},
		cli.IntFlag{
			Name:   "nc",
			Value:  0,
			Hidden: true,
		},
		cli.IntFlag{
			Name:  "sockbuf",
			Value: 4194304, // socket buffer size in bytes
			Usage: "per-socket buffer in bytes",
		},
		cli.IntFlag{
			Name:  "smuxver",
			Value: 1,
			Usage: "specify smux version, available 1,2",
		},
		cli.IntFlag{
			Name:  "smuxbuf",
			Value: 4194304,
			Usage: "the overall de-mux buffer in bytes",
		},
		cli.IntFlag{
			Name:  "streambuf",
			Value: 2097152,
			Usage: "per stream receive buffer in bytes, smux v2+",
		},
		cli.IntFlag{
			Name:  "keepalive",
			Value: 10, // nat keepalive interval in seconds
			Usage: "seconds between heartbeats",
		},
		cli.StringFlag{
			Name:  "snsilog",
			Value: "",
			Usage: "collect snsi to file, aware of timeformat in golang, like: ./snsi-20060102.log",
		},
		cli.IntFlag{
			Name:  "snsiperiod",
			Value: 60,
			Usage: "snsi collect period, in seconds",
		},
		cli.StringFlag{
			Name:  "log",
			Value: "",
			Usage: "specify a log file to output, default goes to stderr",
		},
		cli.BoolFlag{
			Name:  "quiet",
			Usage: "to suppress the 'stream open/close' messages",
		},
		cli.BoolFlag{
			Name:  "tcp",
			Usage: "to emulate a TCP connection(linux)",
		},
		cli.StringFlag{
			Name:  "c",
			Value: "", // when the value is not empty, the config path must exists
			Usage: "config from json file, which will override the command from shell",
		},
	}
	myApp.Action = func(c *cli.Context) error {
		config := Config{}
		config.LocalAddr = c.String("localaddr")
		config.RemoteAddr = c.String("remoteaddr")
		config.Key = c.String("key")
		config.Mode = c.String("mode")
		config.Conn = c.Int("conn")
		config.AutoExpire = c.Int("autoexpire")
		config.ScavengeTTL = c.Int("scavengettl")
		config.MTU = c.Int("mtu")
		config.SndWnd = c.Int("sndwnd")
		config.RcvWnd = c.Int("rcvwnd")
		config.DataShard = c.Int("datashard")
		config.ParityShard = c.Int("parityshard")
		config.DSCP = c.Int("dscp")
		config.NoComp = c.Bool("nocomp")
		config.AckNodelay = c.Bool("acknodelay")
		config.NoDelay = c.Int("nodelay")
		config.Interval = c.Int("interval")
		config.Resend = c.Int("resend")
		config.NoCongestion = c.Int("nc")
		config.SockBuf = c.Int("sockbuf")
		config.SmuxBuf = c.Int("smuxbuf")
		config.StreamBuf = c.Int("streambuf")
		config.SmuxVer = c.Int("smuxver")
		config.KeepAlive = c.Int("keepalive")
		config.Log = c.String("log")
		config.SnsiLog = c.String("snsilog")
		config.SnsiPeriod = c.Int("snsiperiod")
		config.Quiet = c.Bool("quiet")
		config.TCP = c.Bool("tcp")

		if c.String("c") != "" {
			err := parseJSONConfig(&config, c.String("c"))
			checkError(err)
		}

		// log redirect
		if config.Log != "" {
			f, err := os.OpenFile(config.Log, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
			checkError(err)
			defer f.Close()
			log.SetOutput(f)
		}

		switch config.Mode {
		case "normal":
			config.NoDelay, config.Interval, config.Resend, config.NoCongestion = 0, 40, 2, 1
		case "fast":
			config.NoDelay, config.Interval, config.Resend, config.NoCongestion = 0, 30, 2, 1
		case "fast2":
			config.NoDelay, config.Interval, config.Resend, config.NoCongestion = 1, 20, 2, 1
		case "fast3":
			config.NoDelay, config.Interval, config.Resend, config.NoCongestion = 1, 10, 2, 1
		}

		log.Println("version:", VERSION)
		addr, err := net.ResolveTCPAddr("tcp", config.LocalAddr)
		checkError(err)
		listener, err := net.ListenTCP("tcp", addr)
		checkError(err)

		log.Println("smux version:", config.SmuxVer)
		log.Println("listening on:", listener.Addr())
		log.Println("nodelay parameters:", config.NoDelay, config.Interval, config.Resend, config.NoCongestion)
		log.Println("remote address:", config.RemoteAddr)
		log.Println("sndwnd:", config.SndWnd, "rcvwnd:", config.RcvWnd)
		log.Println("compression:", !config.NoComp)
		log.Println("mtu:", config.MTU)
		log.Println("datashard:", config.DataShard, "parityshard:", config.ParityShard)
		log.Println("acknodelay:", config.AckNodelay)
		log.Println("dscp:", config.DSCP)
		log.Println("sockbuf:", config.SockBuf)
		log.Println("smuxbuf:", config.SmuxBuf)
		log.Println("streambuf:", config.StreamBuf)
		log.Println("keepalive:", config.KeepAlive)
		log.Println("conn:", config.Conn)
		log.Println("autoexpire:", config.AutoExpire)
		log.Println("scavengettl:", config.ScavengeTTL)
		log.Println("snsilog:", config.SnsiLog)
		log.Println("snsiperiod:", config.SnsiPeriod)
		log.Println("quiet:", config.Quiet)
		log.Println("tcp:", config.TCP)

		// parameters check
		if config.SmuxVer > maxSmuxVer {
			log.Fatal("unsupported smux version:", config.SmuxVer)
		}

		createConn := func() (*smux.Session, error) {
			kcpconn, err := dial(&config)
			if err != nil {
				return nil, errors.Wrap(err, "dial()")
			}
			kcpconn.SetStreamMode(true)
			kcpconn.SetWriteDelay(false)
			kcpconn.SetNoDelay(config.NoDelay, config.Interval, config.Resend, config.NoCongestion)
			kcpconn.SetWindowSize(config.SndWnd, config.RcvWnd)
			kcpconn.SetMtu(config.MTU)
			kcpconn.SetACKNoDelay(config.AckNodelay)

			if err := kcpconn.SetDSCP(config.DSCP); err != nil {
				log.Println("SetDSCP:", err)
			}
			if err := kcpconn.SetReadBuffer(config.SockBuf); err != nil {
				log.Println("SetReadBuffer:", err)
			}
			if err := kcpconn.SetWriteBuffer(config.SockBuf); err != nil {
				log.Println("SetWriteBuffer:", err)
			}
			log.Println("smux version:", config.SmuxVer, "on connection:", kcpconn.LocalAddr(), "->", kcpconn.RemoteAddr())
			smuxConfig := smux.DefaultConfig()
			smuxConfig.Version = config.SmuxVer
			smuxConfig.MaxReceiveBuffer = config.SmuxBuf
			smuxConfig.MaxStreamBuffer = config.StreamBuf
			smuxConfig.KeepAliveInterval = time.Duration(config.KeepAlive) * time.Second

			if err := smux.VerifyConfig(smuxConfig); err != nil {
				log.Fatalf("%+v", err)
			}

			// stream multiplex
			var session *smux.Session
			if config.NoComp {
				session, err = smux.Client(kcpconn, smuxConfig)
			} else {
				session, err = smux.Client(generic.NewCompStream(kcpconn), smuxConfig)
			}
			if err != nil {
				return nil, errors.Wrap(err, "createConn()")
			}
			return session, nil
		}

		// wait until a connection is ready
		waitConn := func() *smux.Session {
			for {
				if session, err := createConn(); err == nil {
					return session
				}
				log.Println("re-connecting:", err)
				time.Sleep(time.Second)
			}
		}

		// start snsi logger
		go generic.SnsiLogger(config.SnsiLog, config.SnsiPeriod)

		// start scavenger
		chScavenger := make(chan timedSession, 128)
		go scavenger(chScavenger, &config)

		// start listener
		numconn := uint16(config.Conn)
		muxes := make([]timedSession, numconn)
		rr := uint16(0)
		for {
			p1, err := listener.AcceptTCP()
			if err != nil {
				log.Fatalf("%+v", err)
			}
			idx := rr % numconn

			// do auto expiration && reconnection
			if muxes[idx].session == nil || muxes[idx].session.IsClosed() ||
				(config.AutoExpire > 0 && time.Now().After(muxes[idx].expiryDate)) {
				muxes[idx].session = waitConn()
				muxes[idx].expiryDate = time.Now().Add(time.Duration(config.AutoExpire) * time.Second)
				if config.AutoExpire > 0 { // only when autoexpire set
					chScavenger <- muxes[idx]
				}
			}

			go handleClient(muxes[idx].session, p1, config.Quiet)
			rr++
		}
	}
	myApp.Run(os.Args)
}

func scavenger(ch chan timedSession, config *Config) {
	// When AutoExpire is set to 0 (default), sessionList will keep empty.
	// Then this routine won't need to do anything; thus just terminate it.
	if config.AutoExpire <= 0 {
		return
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var sessionList []timedSession
	for {
		select {
		case item := <-ch:
			sessionList = append(sessionList, timedSession{
				item.session,
				item.expiryDate.Add(time.Duration(config.ScavengeTTL) * time.Second),
			})
		case <-ticker.C:
			if len(sessionList) == 0 {
				continue
			}

			var newList []timedSession
			for k := range sessionList {
				s := sessionList[k]
				if s.session.IsClosed() {
					log.Println("scavenger: session normally closed:", s.session.LocalAddr())
				} else if time.Now().After(s.expiryDate) {
					s.session.Close()
					log.Println("scavenger: session closed due to ttl:", s.session.LocalAddr())
				} else {
					newList = append(newList, sessionList[k])
				}
			}
			sessionList = newList
		}
	}
}
