## Flexvolume 部署所需要的的镜像生成方法

> flexvolume 目录下，执行
> docker build -t <image name> .
> 会生成对应的镜像
>> 可能会有的报错：
    ```shell
    Step 3/9 : RUN cd /go/src/github.com/capitalonline/flexvolume/ && ./build.sh
     ---> Running in dafcd4603abc
    ': No such file or directory
    The command '/bin/sh -c cd /go/src/github.com/capitalonline/flexvolume/ && ./build.sh' returned a non-zero code: 127
    ```
    解决：
    ```shell
   （1）使用vi工具
 
      vi test.sh
 
    （2）利用如下命令查看文件格式 
     :set ff 或 :set fileformat 
     可以看到如下信息 
     fileformat=dos 或 fileformat=unix 
     （3） 利用如下命令修改文件格式 
     :set ff=unix 或 :set fileformat=unix 
     :wq (存盘退出)
    
    ```
> docker image ls | grep <image name>
