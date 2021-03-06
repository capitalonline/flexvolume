package nas

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/capitalonline/flexvolume/provider/utils"
)

// NasOptions nas options
type NasOptions struct {
	Server     string `json:"server"`
	Path       string `json:"path,omitempty"`
	Vers       string `json:"vers,omitempty"`
	Mode       string `json:"mode,omitempty"`
	Opts       string `json:"options,omitempty"`
	VolumeName string `json:"kubernetes.io/pvOrVolumeName"`
}

// const values
const (
	NASPORTNUM     = "2049"
	NASTEMPMNTPath = "/mnt/cds_mnt/k8s_nas/" // used for create sub directory;
	MODECHAR       = "01234567"
	defaultV3V4Path = "/nfsshare"
	defaultV3Opts  = "noresvport,nolock,tcp"
	defaultV4Opts  = "noresvport"
)

// NasPlugin nas plugin

// Change NasPlugin struct null
type NasPlugin struct {
}

// NewOptions new options.
func (p *NasPlugin) NewOptions() interface{} {
	return &NasOptions{}
}

// Init plugin init
func (p *NasPlugin) Init() utils.Result {
	return utils.Succeed()
}

// Mount nas support mount and unmount
func (p *NasPlugin) Mount(opts interface{}, mountPath string) utils.Result {

	log.Infof("Nas Plugin Mount: %s", strings.Join(os.Args, ","))

	opt := opts.(*NasOptions)
	log.Infof("kubelet params is: %s, %s, %s, %s, %s", opt.Vers, opt.Server, opt.Path, opt.Opts, opt.Mode)
	if err := p.checkOptions(opt); err != nil {
		utils.FinishError("Nas, check option error: " + err.Error())
	}

	if utils.IsMounted(mountPath) {
		log.Infof("Nas, Mount Path Already Mount, options: %s", mountPath)
		return utils.Result{Status: "Success"}
	}

	// Add NAS white list if needed
	// updateNasWhiteList(opt)

	// if system not set nas, config it.
	checkSystemNasConfig()

	// Create Mount Path
	if err := utils.CreateDest(mountPath); err != nil {
		utils.FinishError("Nas, Mount error with create Path fail: " + mountPath)
	}

	// Do mount
	mntCmd := fmt.Sprintf("mount -t nfs -o vers=%s %s:%s %s", opt.Vers, opt.Server, opt.Path, mountPath)
	if opt.Opts != "" {
		mntCmd = fmt.Sprintf("mount -t nfs -o vers=%s,%s %s:%s %s", opt.Vers, opt.Opts, opt.Server, opt.Path, mountPath)
	}
	log.Infof("Exec Nas Mount Cmd: %s", mntCmd)
	_, err := utils.Run(mntCmd)

	// Mount to nfs Sub-directory
	if err != nil && opt.Path != "/" {
		if strings.Contains(err.Error(), "reason given by server: No such file or directory") || strings.Contains(err.Error(), "access denied by server while mounting") {
			p.createNasSubDir(opt)
			if _, err := utils.Run(mntCmd); err != nil {
				utils.FinishError("Nas, Mount Nfs sub directory fail: " + err.Error())
			}
		} else {
			utils.FinishError("Nas, Mount Nfs fail with error: " + err.Error())
		}
		// mount error
	} else if err != nil {
		utils.FinishError("Nas, Mount nfs fail: " + err.Error())
	}

	// change the mode
	if opt.Mode != "" && opt.Path != "/" {
		var wg1 sync.WaitGroup
		wg1.Add(1)

		go func(*sync.WaitGroup) {
			cmd := fmt.Sprintf("chmod %s %s", opt.Mode, mountPath)
			if _, err := utils.Run(cmd); err != nil {
				log.Errorf("Nas chmod cmd fail: %s %s", cmd, err)
			} else {
				log.Infof("Nas chmod cmd success: %s", cmd)
			}
			wg1.Done()
		}(&wg1)

		if waitTimeout(&wg1, 1) {
			log.Infof("Chmod use more than 1s, running in Concurrency: %s", mountPath)
		}
	}

	// check mount
	if !utils.IsMounted(mountPath) {
		utils.FinishError("Check mount fail after mount:" + mountPath)
	}
	log.Info("Mount success on: " + mountPath)
	return utils.Result{Status: "Success"}
}

// check system config,
// if tcp_slot_table_entries not set to 128, just config.

func checkSystemNasConfig() {
	updateNasConfig := false
	sunRpcFile := "/etc/modprobe.d/sunrpc.conf"
	if !utils.IsFileExisting(sunRpcFile) {
		updateNasConfig = true
	} else {
		chkCmd := fmt.Sprintf("cat %s | grep tcp_slot_table_entries | grep 128 | grep -v grep | wc -l", sunRpcFile)
		out, err := utils.Run(chkCmd)
		if err != nil {
			log.Warnf("Update Nas system config check error: ", err.Error())
			return
		}
		if strings.TrimSpace(out) == "0" {
			updateNasConfig = true
		}
	}

	if updateNasConfig {
		upCmd := fmt.Sprintf("echo \"options sunrpc tcp_slot_table_entries=128\" >> %s && echo \"options sunrpc tcp_max_slot_table_entries=128\" >> %s && sysctl -w sunrpc.tcp_slot_table_entries=128", sunRpcFile, sunRpcFile)
		_, err := utils.Run(upCmd)
		if err != nil {
			log.Warnf("Update Nas system config error: ", err.Error())
			return
		}
		log.Warnf("Successful update Nas system config")
	}
}

// Unmount mnt
func (p *NasPlugin) Unmount(mountPoint string) utils.Result {
	log.Info("Nas Plugin Unmount: ", strings.Join(os.Args, ","))

	if !utils.IsMounted(mountPoint) {
		return utils.Succeed()
	}

	// do umount command
	// check if needed force umount
	networkUnReachable := false
	noOtherPodUsed := false
	nfsServer := p.getNasServerInfo(mountPoint)
	if nfsServer != "" && !p.isNasServerReachable(nfsServer) {
		log.Warnf("NFS, Connect to server: %s failed, umount to %s", nfsServer, mountPoint)
		networkUnReachable = true
	}
	if networkUnReachable && p.noOtherNasUser(nfsServer, mountPoint) {
		log.Warnf("NFS, Other pods is using the NAS server %s, %s", nfsServer, mountPoint)
		noOtherPodUsed = true
	}
	// default umount
	cmd := exec.Command("umount", mountPoint)
	if networkUnReachable || noOtherPodUsed {
		cmd = exec.Command("umount", "-f", mountPoint)
	}
	var timer *time.Timer
	timeout := false
	timer = time.AfterFunc(5*time.Second, func() {
		timer.Stop()
		cmd.Process.Kill()
		timeout = true
	})
	err := cmd.Run()
	if timeout == true {
		log.Warnf("Nas, Umount nfs Fail with time out: " + err.Error())
	} else if utils.IsMounted(mountPoint) {
		utils.FinishError("Nas, Umount nfs Fail: " + err.Error())
	}

	log.Info("Umount nfs Successful:", mountPoint)
	return utils.Succeed()
}

// to get nasServerInfo
func (p *NasPlugin) getNasServerInfo(mountPoint string) string {
	getNasServerPath := fmt.Sprintf("findmnt %s | grep %s | grep -v grep | awk '{print $2}'", mountPoint, mountPoint)
	serverAndPath, _ := utils.Run(getNasServerPath)
	serverAndPath = strings.TrimSpace(serverAndPath)

	serverInfoPartList := strings.Split(serverAndPath, ":")
	if len(serverInfoPartList) != 2 {
		log.Warnf("NFS, Get Nas Server error format: %s, %s", serverAndPath, mountPoint)
		return ""
	}
	return serverInfoPartList[0]
}

func (p *NasPlugin) noOtherNasUser(nfsServer, mountPoint string) bool {
	checkCmd := fmt.Sprintf("mount | grep -v %s | grep %s | grep -v grep | wc -l", mountPoint, nfsServer)
	if checkOut, err := utils.Run(checkCmd); err != nil {
		return false
	} else if strings.TrimSpace(checkOut) != "0" {
		return false
	}
	return true
}

func (p *NasPlugin) isNasServerReachable(url string) bool {
	conn, err := net.DialTimeout("tcp", url+":"+NASPORTNUM, time.Second*2)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

// 1. mount to /mnt/cds_mnt/k8s_nas/volumename first
// 2. run mkdir for sub directory
// 3. umount the tmep directory
func (p *NasPlugin) createNasSubDir(opt *NasOptions) {
	// step 1: create mount path
	nasTmpPath := filepath.Join(NASTEMPMNTPath, opt.VolumeName)
	if err := utils.CreateDest(nasTmpPath); err != nil {
		utils.FinishError("Create Nas temp Directory err: " + err.Error())
	}
	if utils.IsMounted(nasTmpPath) {
		utils.Umount(nasTmpPath)
	}

	// step 2: do mount
	usePath := opt.Path
	mntCmd := fmt.Sprintf("mount -t nfs -o vers=%s %s:%s %s", opt.Vers, opt.Server, "/", nasTmpPath)
	_, err := utils.Run(mntCmd)
	if err != nil {
		if strings.Contains(err.Error(), "reason given by server: No such file or directory") || strings.Contains(err.Error(), "access denied by server while mounting") {
			if strings.HasPrefix(opt.Path, defaultV3V4Path+"/") {
				usePath = strings.TrimPrefix(usePath, defaultV3V4Path)
				mntCmd = fmt.Sprintf("mount -t nfs -o vers=%s %s:%s %s", opt.Vers, opt.Server, defaultV3V4Path, nasTmpPath)
				_, err := utils.Run(mntCmd)
				if err != nil {
					utils.FinishError("Nas, Mount to temp directory(with /nfsshare) fail: " + err.Error())
				}
			} else {
				utils.FinishError("Nas, maybe use version 3, but path not startwith /nfsshare: " + err.Error())
			}
		} else {
			utils.FinishError("Nas, Mount to temp directory fail: " + err.Error())
		}
	}
	subPath := path.Join(nasTmpPath, usePath)

	if err := utils.CreateDest(subPath); err != nil {
		utils.FinishError("Nas, Create Sub Directory err: " + err.Error())
	}

	// step 3: umount after create
	utils.Umount(nasTmpPath)
	log.Info("Create Sub Directory success: ", opt.Path)
}

//
func (p *NasPlugin) checkOptions(opt *NasOptions) error {
	// NFS Server url
	if opt.Server == "" {
		return errors.New("NAS url is empty")
	}
	// check network connection
	conn, err := net.DialTimeout("tcp", opt.Server+":"+NASPORTNUM, time.Second*time.Duration(3))
	if err != nil {
		log.Errorf("NAS: Cannot connect to nas host: %s", opt.Server)
		errMsg := fmt.Sprintf("NAS: Cannot connect to nas host: " + opt.Server)
		return errors.New(errMsg)
	}
	defer conn.Close()

	// default nfs version 4.0
	if opt.Vers == "" {
		opt.Vers = "4.0"
	}

	// set vers=3.0 to vers=3, because vers=3.0 mount cmd submit error "parsing error on 'vers=' option"
	if strings.HasPrefix(opt.Vers, "3.0") {
		opt.Vers = "3"
	}

	// if input vers=4, then set vers=4.0
	if strings.HasPrefix(opt.Vers, "4") {
		opt.Vers = "4.0"
	}

	// nfs server path
	if opt.Path == "" {
		opt.Path = defaultV3V4Path
	}
	if !strings.HasPrefix(opt.Path, "/nfsshare") {
		log.Errorf("NAS: Path should start with /nfsshare, %s", opt.Path)
		errMsg := fmt.Sprintf("NAS: Path should start with /nfsshare, %s: " + opt.Path)
		return errors.New(errMsg)
	}

	// check mode
	if opt.Mode != "" {
		modeLen := len(opt.Mode)
		if modeLen != 3 {
			errMsg := fmt.Sprintf("NAS: mode input format error: " + opt.Mode)
			return errors.New(errMsg)
		}
		for i := 0; i < modeLen; i++ {
			if !strings.Contains(MODECHAR, opt.Mode[i:i+1]) {
				log.Errorf("NAS: mode is illegal, %s", opt.Mode)
				errMsg := fmt.Sprintf("NAS: mode input format error: " + opt.Mode)
				return errors.New(errMsg)
			}
		}
	}

	// check options
	if opt.Opts == "" {
		if opt.Vers == "4.0" {
			opt.Opts = defaultV4Opts
		} else {
			opt.Opts = defaultV3Opts
		}
	} else if strings.ToLower(opt.Opts) == "none" {
		opt.Opts = ""
	}
	return nil
}

func waitTimeout(wg *sync.WaitGroup, timeout int) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false
	case <-time.After(time.Duration(timeout) * time.Second):
		return true
	}

}

// Attach not support
func (p *NasPlugin) Attach(opts interface{}, nodeName string) utils.Result {
	return utils.NotSupport()
}

// Detach not support
func (p *NasPlugin) Detach(device string, nodeName string) utils.Result {
	return utils.NotSupport()
}

// Waitforattach no Support
func (p *NasPlugin) Waitforattach(devicePath string, opts interface{}) utils.Result {
	return utils.NotSupport()
}

// Mountdevice Not Support
func (p *NasPlugin) Mountdevice(mountPath string, opts interface{}) utils.Result {
	return utils.NotSupport()
}
