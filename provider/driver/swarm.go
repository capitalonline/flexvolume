package driver

import "github.com/AliyunContainerService/flexvolume/provider/utils"

// RunningInSwarm not support now
func RunningInSwarm() {
	utils.Finish(utils.Fail("Not support swam platform"))
}
