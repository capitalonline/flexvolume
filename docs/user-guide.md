## 首都云NAS使用指南

### 使用说明

NAS为共享文件存储，可以同时为多个Pod提供共享存储服务

> 1. server：为NAS数据盘的ip地址；
> 2. path：为NAS数据盘的挂载路径，支持挂载nas子目录；且当子目录不存在时，自动创建子目录并挂载；
> 3. vers：定义nfs挂载协议的版本号，支持：4.0；
> 4. mode：定义挂载目录的访问权限，注意：挂载NAS盘根目录时不能配置挂载权限；

### 使用前准备
> 1. 手动创建NAS共享存储盘并记录ip地址和共享目录

### 直接通过 Volume 使用 (replicas = 1)
- Create Pod with spec `nas-deploy.yaml`. 

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-nas-deploy
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx-flexvolume-nas
        image: nginx
        volumeMounts:
          - name: volumeNas
            mountPath: "/data"
      volumes:
        - name: volumeNas
          flexVolume:
            driver: "cdscloud/nas"
            options:
	          server: NasServerIP, please replace with your own nas server ip
              path: NasSharePath, please replace with your own nas server share path
              vers: "4.0"
			  mode: "644"
```

### 通过 PV/PVC 使用（目前不支持动态pv）

- Create pv with spec `nas-pv.yaml`. 注意pv name 要与 volumeId相同

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: nas-pv
spec:
  capacity:
    storage: 20Gi
  accessModes:
    - ReadWriteMany
  flexVolume:
    driver: "cdscloud/nas"
    options:
      server: NasServerIP, please replace with your own nas server ip
      path: NasSharePath, please replace with your own nas server share path
      vers: "4.0"

```

- Create PVC with spec `nas-pvc.yaml`

```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: nas-pvc
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 20Gi

```

- Create Pod with spec `nas-pod.yaml`

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nas-pod
spec:
  containers:
    - name: "nginx"
      image: "nginx"
      volumeMounts:
          - name: pvc-nas
            mountPath: "/data"
  volumes:
  - name: pvc-nas
    persistentVolumeClaim:
        claimName: nas-pvc

```