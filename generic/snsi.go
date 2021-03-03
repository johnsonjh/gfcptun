package generic

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	kcp "github.com/johnsonjh/gfcp"
)

// SnsiLogger ...
func SnsiLogger(path string, interval int) {
	if path == "" || interval == 0 {
		return
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// split path into dirname and filename
			logdir, logfile := filepath.Split(path)
			// only format logfile
			f, err := os.OpenFile(logdir+time.Now().Format(logfile), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
			if err != nil {
				log.Println(err)
				return
			}
			w := csv.NewWriter(f)
			// write header in empty file
			if stat, err := f.Stat(); err == nil && stat.Size() == 0 {
				if err := w.Write(append([]string{"Unix"}, kcp.DefaultSnsi.Header()...)); err != nil {
					log.Println(err)
				}
			}
			if err := w.Write(append([]string{fmt.Sprint(time.Now().Unix())}, kcp.DefaultSnsi.ToSlice()...)); err != nil {
				log.Println(err)
			}
			// kcp.DefaultSnsi.Reset()
			w.Flush()
			f.Close()
		}
	}
}
