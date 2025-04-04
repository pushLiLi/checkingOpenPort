go build portscan.go 打包生产exe文件
./portscan -t 目标域名或IP地址(必须的) -p 端口范围 (示例：80,443 或 1-1000) -n 并发线程数 -t 超时时间 -v 显示详细过程

最简单的，扫描目标域名或IP地址：
  ./portscan -t 127.0.0.1
