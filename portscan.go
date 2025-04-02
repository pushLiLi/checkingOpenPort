package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	target  string
	ports   string
	threads int
	timeout time.Duration
	verbose bool
)

func init() {
	flag.StringVar(&target, "t", "", "目标域名或IP地址")
	flag.StringVar(&ports, "p", "1-1024", "端口范围 (示例：80,443 或 1-1000)")
	flag.IntVar(&threads, "n", 100, "并发线程数")
	flag.DurationVar(&timeout, "timeout", 1*time.Second, "超时时间")
	flag.BoolVar(&verbose, "v", false, "显示详细过程")
}

func parsePorts(portsFlag string) ([]int, error) {
	var ports []int

	// 处理逗号分隔的多个范围/端口
	ranges := strings.Split(portsFlag, ",")
	for _, r := range ranges {
		if strings.Contains(r, "-") {
			parts := strings.Split(r, "-")
			if len(parts) != 2 {
				return nil, fmt.Errorf("无效的端口范围: %s", r)
			}

			start, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, fmt.Errorf("无效的端口号: %s", parts[0])
			}

			end, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("无效的端口号: %s", parts[1])
			}

			for i := start; i <= end; i++ {
				ports = append(ports, i)
			}
		} else {
			port, err := strconv.Atoi(r)
			if err != nil {
				return nil, fmt.Errorf("无效的端口号: %s", r)
			}
			ports = append(ports, port)
		}
	}
	return ports, nil
}

func scanPort(ctx context.Context, target string, port int, wg *sync.WaitGroup, results chan<- int) {
	defer wg.Done()

	select {
	case <-ctx.Done():
		return
	default:
		address := fmt.Sprintf("%s:%d", target, port)
		conn, err := net.DialTimeout("tcp", address, timeout)
		if err != nil {
			if verbose {
				fmt.Printf("[.] Port %d closed\n", port)
			}
			return
		}
		conn.Close()
		results <- port
	}
}

func main() {
	flag.Parse()

	if target == "" {
		fmt.Println("必须指定目标 (-t)")
		os.Exit(1)
	}

	// 解析域名/IP
	ipAddr, err := net.ResolveIPAddr("ip", target)
	if err != nil {
		fmt.Printf("无法解析目标: %v\n", err)
		os.Exit(1)
	}

	// 解析端口范围
	portList, err := parsePorts(ports)
	if err != nil {
		fmt.Printf("端口解析错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("扫描目标: %s (%s)\n", target, ipAddr.IP)
	fmt.Printf("扫描端口: %d 个\n", len(portList))
	fmt.Printf("并发线程: %d\n", threads)
	fmt.Println("------------------------")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	results := make(chan int)
	pool := make(chan struct{}, threads)

	// 结果收集
	var openPorts []int
	go func() {
		for port := range results {
			openPorts = append(openPorts, port)
			fmt.Printf("[+] Port %d 开放\n", port)
		}
	}()

	// 启动扫描任务
	for _, port := range portList {
		pool <- struct{}{}
		wg.Add(1)
		go func(p int) {
			defer func() { <-pool }()
			scanPort(ctx, ipAddr.IP.String(), p, &wg, results)
		}(port)
	}

	wg.Wait()
	close(results)
	close(pool)

	fmt.Println("------------------------")
	fmt.Printf("发现 %d 个开放端口\n", len(openPorts))
}
