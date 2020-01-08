#!/bin/sh

# Get System version
host_os="centos-7-4"

/cds/nsenter --mount=/proc/1/ns/mnt which lsb_release
lsb_release_exist=$?
if [ "$lsb_release_exist" != "0" ]; then
  /cds/nsenter --mount=/proc/1/ns/mnt ls /etc/os-release
  os_release_exist=$?
fi

if [ "$lsb_release_exist" = "0" ]; then
    os_info=`/cds/nsenter --mount=/proc/1/ns/mnt lsb_release -a`

    if [ `echo $os_info | grep CentOS | grep 7.2 | wc -l` != "0" ]; then
        host_os="centos-7-2"
    elif [ `echo $os_info | grep CentOS | grep 7.3 | wc -l` != "0" ]; then
        host_os="centos-7-3"
    elif [ `echo $os_info | grep CentOS | grep 7.4 | wc -l` != "0" ]; then
        host_os="centos-7-4"
    elif [ `echo $os_info | grep CentOS | grep 7.5 | wc -l` != "0" ]; then
        host_os="centos-7-5"
    elif [ `echo $os_info | grep CentOS | grep 7. | wc -l` != "0" ]; then
        host_os="centos-7"
    elif [ `echo $os_info | grep 14.04 | wc -l` != "0" ]; then
        host_os="ubuntu-1404"
    elif [ `echo $os_info | grep 16.04 | wc -l` != "0" ]; then
        host_os="ubuntu-1604"
    else
        echo "OS is not ubuntu 1604/1404, Centos7"
        echo "system information: "$os_info
        exit 1
    fi

elif [ "$os_release_exist" = "0" ]; then
    osId=`/cds/nsenter --mount=/proc/1/ns/mnt cat /etc/os-release | grep "ID="`
    osVersion=`/cds/nsenter --mount=/proc/1/ns/mnt cat /etc/os-release | grep "VERSION_ID="`

    if [ `echo $osId | grep "centos" | wc -l` != "0" ]; then
        if [ `echo $osVersion | grep "7" | wc -l` = "1" ]; then
          host_os="centos-7"
        fi
    elif [ `echo $osId | grep "alios" | wc -l` != "0" ];then
       if [ `echo $osVersion | grep "7" | wc -l` = "1" ]; then
         host_os="centos-7"
       fi
    elif [ `echo $osId | grep "ubuntu" | wc -l` != "0" ]; then
        if [ `echo $osVersion | grep "14.04" | wc -l` != "0" ]; then
          host_os="ubuntu-1404"
        elif [ `echo $osVersion | grep "16.04" | wc -l` != "0" ]; then
          host_os="ubuntu-1604"
        fi
    fi
fi

restart_kubelet="false"

install_nas() {
    # install nfs-client
    if [ ! `/cds/nsenter --mount=/proc/1/ns/mnt which mount.nfs4` ]; then
        if [ "$host_os" = "centos-7-4" ] || [ "$host_os" = "centos-7-3" ] || [ "$host_os" = "centos-7-5" ] || [ "$host_os" = "centos-7" ] ; then
            /cds/nsenter --mount=/proc/1/ns/mnt yum install -y nfs-utils

        elif [ "$host_os" = "ubuntu-1404" ] || [ "$host_os" = "ubuntu-1604" ]; then
            /cds/nsenter --mount=/proc/1/ns/mnt apt-get update -y
            /cds/nsenter --mount=/proc/1/ns/mnt apt-get install -y nfs-common
        fi
    fi

    # install lsof tool
    #if [ ! `/cds/nsenter --mount=/proc/1/ns/mnt which lsof` ]; then
    #    if [ "$host_os" = "centos-7-4" ] || [ "$host_os" = "centos-7-3" ] || [ "$host_os" = "centos-7-5" ] || [ "$host_os" = "centos-7" ]; then
    #        /cds/nsenter --mount=/proc/1/ns/mnt yum install -y lsof
    #    fi
    #fi

    # first install
    if [ ! -f "/host/usr/libexec/kubernetes/kubelet-plugins/volume/exec/cdscloud~nas/nas" ];then
        mkdir -p /host/usr/libexec/kubernetes/kubelet-plugins/volume/exec/cdscloud~nas/
        cp /cds/flexvolume /host/usr/libexec/kubernetes/kubelet-plugins/volume/exec/cdscloud~nas/nas
        chmod 755 /host/usr/libexec/kubernetes/kubelet-plugins/volume/exec/cdscloud~nas/nas

    # update nas
    else
        oldmd5=`md5sum /host/usr/libexec/kubernetes/kubelet-plugins/volume/exec/cdscloud~nas/nas | awk '{print $1}'`
        newmd5=`md5sum /cds/flexvolume | awk '{print $1}'`

        # install a new bianary
        if [ "$oldmd5" != "$newmd5" ]; then
            rm -rf /host/usr/libexec/kubernetes/kubelet-plugins/volume/exec/cdscloud~nas/nas
            cp /cds/flexvolume /host/usr/libexec/kubernetes/kubelet-plugins/volume/exec/cdscloud~nas/nas
            chmod 755 /host/usr/libexec/kubernetes/kubelet-plugins/volume/exec/cdscloud~nas/nas
        fi
    fi

}

# if kubelet not disable controller, exit
enableADController="false"
count=`ps -ef | grep kubelet | grep "enable-controller-attach-detach=false" | grep -v "grep" | wc -l`
if [ "$count" = "0" ]; then
  configInFile=`/cds/nsenter --mount=/proc/1/ns/mnt cat /var/lib/kubelet/config.yaml | grep enableControllerAttachDetach | grep false | grep -v grep | wc -l`
  if [ "$configInFile" = "0" ]; then
    enableADController=true
  fi
fi

if [ "$enableADController" = "true" ]; then
  echo "kubelet not running in: enable-controller-attach-detach=false, mount maybe failed"
fi

# install plugins
if [ "$CDS_NAS" = "true" ]; then
  install_nas
fi

## monitoring should be here
/cds/flexvolume monitoring
