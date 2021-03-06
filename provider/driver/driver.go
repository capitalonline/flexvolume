package driver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/capitalonline/flexvolume/provider/monitor"
	"github.com/capitalonline/flexvolume/provider/nas"
	"github.com/capitalonline/flexvolume/provider/utils"
	log "github.com/sirupsen/logrus"
)

// FluxVolumePlugin: VolumePlugin interface for plugins
type FluxVolumePlugin interface {
	NewOptions() interface{} // not called by kubelet
	Init() utils.Result
	Mount(opt interface{}, mountPath string) utils.Result
	Unmount(mountPoint string) utils.Result
	Attach(opt interface{}, nodeName string) utils.Result
	Waitforattach(devicePath string, opt interface{}) utils.Result
	Mountdevice(mountPath string, opt interface{}) utils.Result
	Detach(volumeName string, nodeName string) utils.Result
}

// const values
const (
	MB_SIZE = 1024 * 1024

	// Only support nas with mount and unmount
	TYPE_PLUGIN_NAS   = "nas"
	PLUGIN_MONITORING = "monitoring"
	LOGFILE_PREFIX    = "/var/log/cds/flexvolume_"
)

// RunK8sAction run kubernetes command
func Run() {
	if len(os.Args) < 2 {
		utils.Finish(utils.Fail("Expected at least one parameter"))
	}

	// set log file
	setLogAttribute()

	driver := filepath.Base(os.Args[0])
	if driver == TYPE_PLUGIN_NAS {
		RunPlugin(&nas.NasPlugin{})
	} else if os.Args[1] == PLUGIN_MONITORING {
		monitor.Monitoring()
	} else {
		utils.Finish(utils.Fail("Not Support Plugin Driver: " + os.Args[0]))
	}
}

// RunPlugin only support mount and detach now.
func RunPlugin(plugin FluxVolumePlugin) {

	switch os.Args[1] {
	case "init":
		log.Info("Plugin Init")
		utils.Finish(plugin.Init())

	case "mount":
		if len(os.Args) != 4 {
			utils.FinishError("Mount expected exactly 4 arguments; got: " + strings.Join(os.Args, ","))
		}

		opt := plugin.NewOptions()
		if err := json.Unmarshal([]byte(os.Args[3]), opt); err != nil {
			utils.FinishError("Mount Options illegal; got: " + os.Args[3])
		}

		mountPath := os.Args[2]
		utils.Finish(plugin.Mount(opt, mountPath))

	case "unmount":
		if len(os.Args) != 3 {
			utils.FinishError("Umount expected exactly 3 arguments; got: " + strings.Join(os.Args, ","))
		}

		mountPath := os.Args[2]
		utils.Finish(plugin.Unmount(mountPath))
	case "waitforattach":
		if len(os.Args) != 4 {
			utils.FinishError("waitforattach expected exactly 4 arguments; got: " + strings.Join(os.Args, ","))
		}
		opt := plugin.NewOptions()
		if err := json.Unmarshal([]byte(os.Args[3]), opt); err != nil {
			utils.FinishError("waitforattach Options illegal; got: " + os.Args[3])
		}

		devicePath := os.Args[2]
		utils.Finish(plugin.Waitforattach(devicePath, opt))

	case "attach":
		if len(os.Args) != 4 {
			utils.FinishError("Attach expected exactly 4 arguments; got: " + strings.Join(os.Args, ","))
		}

		opt := plugin.NewOptions()
		if err := json.Unmarshal([]byte(os.Args[2]), opt); err != nil {
			utils.FinishError("Attach Options format illegal, except json but got: " + os.Args[2])
		}

		nodeName := os.Args[3]
		utils.Finish(plugin.Attach(opt, nodeName))

	case "detach":
		if len(os.Args) != 4 {
			utils.FinishError("Detach expect 4 args; got: " + strings.Join(os.Args, ","))
		}

		volumeName := os.Args[2]
		utils.Finish(plugin.Detach(volumeName, os.Args[3]))
	default:
		utils.Finish(utils.NotSupport(os.Args))
	}

}

// rotate log file by 2M bytes
func setLogAttribute() {
	driver := filepath.Base(os.Args[0])
	logFile := LOGFILE_PREFIX + driver + ".log"
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		utils.Finish(utils.Fail("Log File open error"))
	}

	// rotate the log file if too large
	if fi, err := f.Stat(); err == nil && fi.Size() > 2*MB_SIZE {
		f.Close()
		timeStr := time.Now().Format("-2006-01-02-15:04:05")
		timedLogfile := LOGFILE_PREFIX + driver + timeStr + ".log"
		os.Rename(logFile, timedLogfile)
		f, err = os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			utils.Finish(utils.Fail("Log File open error2"))
		}
	}
	log.SetOutput(f)
}
