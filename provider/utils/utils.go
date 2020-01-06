package utils

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
)

// DefaultOptions used for global ak
type DefaultOptions struct {
	Global struct {
		KubernetesClusterTag string
		AccessKeyID          string `json:"accessKeyID"`
		AccessKeySecret      string `json:"accessKeySecret"`
		Region               string `json:"region"`
	}
}

// Succeed successful action
func Succeed(a ...interface{}) Result {
	return Result{
		Status:  "Success",
		Message: fmt.Sprint(a...),
	}
}

// NotSupport not support action
func NotSupport(a ...interface{}) Result {
	return Result{
		Status:  "Not supported",
		Message: fmt.Sprint(a...),
	}
}

// Fail fail the flexvolume call
func Fail(a ...interface{}) Result {
	return Result{
		Status:  "Failure",
		Message: fmt.Sprint(a...),
	}
}

// Finish finish call
func Finish(result Result) {
	code := 1
	if result.Status == "Success" {
		code = 0
	}
	res, err := json.Marshal(result)
	if err != nil {
		fmt.Println("{\"status\":\"Failure\",\"message\":", err.Error(), "}")
	} else {
		fmt.Println(string(res))
	}
	os.Exit(code)
}

// FinishError print error info
func FinishError(message string) {
	log.Info("Exit with Error: ", message)
	Finish(Fail(message))
}

// Result of flexvolume
type Result struct {
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
	Device     string `json:"device,omitempty"`
	VolumeName string `json:"volumeName"`
}

// Run run shell command
func Run(cmd string) (string, error) {
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Failed to run cmd: " + cmd + ", with out: " + string(out) + ", with error: " + err.Error())
	}
	return string(out), nil
}

// CreateDest create directory
func CreateDest(dest string) error {
	fi, err := os.Lstat(dest)

	if os.IsNotExist(err) {
		if err := os.MkdirAll(dest, 0777); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if fi != nil && !fi.IsDir() {
		return fmt.Errorf("%v already exist but it's not a directory", dest)
	}
	return nil
}

// IsMounted check directory is mounted or not.
func IsMounted(mountPath string) bool {
	cmd := fmt.Sprintf("mount | grep \"%s type\" | grep -v grep", mountPath)
	out, err := Run(cmd)
	if err != nil || out == "" {
		return false
	}
	return true
}

// Umount umount path.
func Umount(mountPath string) bool {
	cmd := fmt.Sprintf("umount -f %s", mountPath)
	_, err := Run(cmd)
	if err != nil {
		return false
	}
	return true
}

// IsFileExisting check file exist in volume driver;
func IsFileExisting(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

// WriteJosnFile save json data to file
func WriteJosnFile(obj interface{}, file string) error {
	maps := make(map[string]interface{})
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).String() != "" {
			maps[t.Field(i).Name] = v.Field(i).String()
		}
	}
	rankingsJson, _ := json.Marshal(maps)
	if err := ioutil.WriteFile(file, rankingsJson, 0644); err != nil {
		return err
	}
	return nil
}

// ReadJsonFile parse json to struct
func ReadJsonFile(file string) (map[string]string, error) {
	jsonObj := map[string]string{}
	raw, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(raw, &jsonObj)
	if err != nil {
		return nil, err
	}
	return jsonObj, nil
}

// PathExists returns true if the specified path exists.
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}
