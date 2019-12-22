package process

import (
	"fmt"
	"github.com/axgle/mahonia"
	"net"
	"os/exec"
	"strconv"
	"strings"
)

func WalkerWindows() Walker {
	return &walker{}
}

type walker struct{}

const (
	ssBinary      = "ss"
	psBinary      = "ps"
)

func (walker) Walk() (Topview, error) {

	var topview = Topview{}

	err := ParseSSLISTEN(&topview)
	if err != nil {
		return Topview{}, err
	}

	err = ParseSSESTAB(&topview)
	if err != nil {
		return Topview{}, err
	}

	return topview, nil
}

func isLocalIp(ip string) bool {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		fmt.Println("net.Interfaces failed, err:", err.Error())
		return false
	}
	for i := 0; i < len(netInterfaces); i++ {
		if (netInterfaces[i].Flags & net.FlagUp) != 0 {
			addrs, _ := netInterfaces[i].Addrs()
			for _, address := range addrs {
				if ipnet, ok := address.(*net.IPNet); ok {
					if ipnet.IP.To4()!=nil {
						if strings.EqualFold(ipnet.IP.String(), ip) {
							return true
						}
					} else if ipnet.IP.To16()!=nil {
						if strings.Contains("["+ipnet.IP.String()+"]", ip) {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func ParseSSOUTPUT(line string) (string, int, string, int, string, int, int, int, error) {
	fields := strings.Fields(line)

	if len(fields) < 6 || !strings.Contains(fields[5], "((") || !strings.Contains(fields[5], "))") {
		return "", -1, "", -1, "", -1, -1, -1, fmt.Errorf("invalid field in message: %#v", fields[5])
	}

	message := strings.Split(fields[5][strings.Index(fields[5], "((")+2:strings.Index(fields[5], "))")], ",")

	name := strings.Trim(message[0],"\"")
	pid, err := strconv.Atoi(strings.Trim(message[1], "pid="))
	if err != nil {
		return "", -1, "", -1, "", -1, -1, -1, fmt.Errorf("invalid field in message: %#v", fields[5])
	}

	addr := strings.Split(fields[3], ":")
	localPort, err := strconv.Atoi(addr[len(addr)-1])
	localIp := strings.Join(addr[0:len(addr)-1], ":")
	if strings.Contains(localIp, "[::ffff:") {
		localIp = localIp[8:strings.Index(localIp, "]")]
	} else if strings.Contains(localIp, "::ffff:") {
		fmt.Println(localIp)
		localIp = localIp[7:len(localIp)]
	}
	if err != nil {
		return "", -1, "", -1, "", -1, -1, -1, fmt.Errorf("invalid field in addr: %#v", fields[3])
	}

	recvq, err := strconv.Atoi(fields[1])
	if err != nil {
		return "", -1, "", -1, "", -1, -1, -1, fmt.Errorf("invalid field in recv-Q: %#v", fields[1])
	}
	sendq, err := strconv.Atoi(fields[2])
	if err != nil {
		return "", -1, "", -1, "", -1, -1, -1, fmt.Errorf("invalid field in send-Q: %#v", fields[2])
	}

	if !strings.EqualFold(fields[0], "LISTEN") {
		addr := strings.Split(fields[4], ":")
		peerPort, err := strconv.Atoi(addr[len(addr)-1])
		peerIp := strings.Join(addr[0:len(addr)-1], ":")
		if strings.Contains(peerIp, "[::ffff:") {
			peerIp = peerIp[8:strings.Index(peerIp, "]")]
		} else if strings.Contains(peerIp, "::ffff:") {
			fmt.Println(localIp)
			peerIp = peerIp[7:len(peerIp)]
		}
		if err != nil {
			return "", -1, "", -1, "", -1, -1, -1, fmt.Errorf("invalid field in addr: %#v", fields[4])
		}
		return localIp, localPort, peerIp, peerPort, name, pid, recvq, sendq, nil
	}
	return localIp, localPort, "", -1, name, pid, recvq, sendq, nil
}

func ParseSSLISTEN(topview *Topview) error {

	outputBinary, err := exec.Command(
		ssBinary,
		"-a",
		"-n", "-o",
		"-p", "-t",
	).CombinedOutput()
	if err != nil {
		return err
	}
	output := mahonia.NewDecoder("utf-8").ConvertString(string(outputBinary))

	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if (len(fields) < 6) || !strings.Contains(strings.ToUpper(fields[0]), "LISTEN") {
			continue
		}

		_, port, _, _, name, pid, recvq, sendq, err := ParseSSOUTPUT(line)

		if err != nil {
			return err
		}

		var process = Process{Key: strconv.Itoa(pid), Name: name, Recvq: recvq, PID: pid, Sendq: sendq, Type: "process"}
		process.AddPort(port)
		netInterfaces, err := net.Interfaces()
		if err != nil {
			fmt.Println("net.Interfaces failed, err:", err.Error())
			continue
		}
		for i := 0; i < len(netInterfaces); i++ {
			if (netInterfaces[i].Flags & net.FlagUp) != 0 {
				addrs, _ := netInterfaces[i].Addrs()
				for _, address := range addrs {
					if ipnet, ok := address.(*net.IPNet); ok {
						if ipnet.IP.To4()!=nil {
							process.AddIp(ipnet.IP.String())
						} else if ipnet.IP.To16()!=nil {
							process.AddIp("["+ipnet.IP.String()+"]")
						}
					}
				}
			}
		}
		topview.addNode(process)
	}
	return nil
}

func ParseSSESTAB(topview *Topview) error {

	outputBinary, err := exec.Command(
		ssBinary,
		"-a",
		"-n", "-o",
		"-p", "-t",
	).CombinedOutput()
	if err != nil {
		return err
	}

	output := mahonia.NewDecoder("utf-8").ConvertString(string(outputBinary))
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if (len(fields) < 6) || !strings.Contains(strings.ToUpper(fields[0]), "ESTAB") {
			continue
		}
		localIp, localPort, peerIp, peerPort, name, pid, recvq, sendq, err := ParseSSOUTPUT(line)

		if err != nil {
			return err
		}

		var process = Process{Key: strings.Join([]string{peerIp, strconv.Itoa(peerPort)}, ":"),
			Name: strings.Join([]string{peerIp, strconv.Itoa(peerPort)}, ":"), Recvq: recvq, Sendq: sendq, Type: "process"}
		process.AddIp(peerIp)
		process.AddPort(peerPort)
		if !isLocalIp(peerIp) {
			topview.addNode(process)
		}

		peerprocess, peererr := topview.GetNodeByIpAndPort(peerIp, peerPort)
		localprocess, localerr := topview.GetNodeByIpAndPort(localIp, localPort)

		if localerr==nil && isLocalIp(peerIp) {
			// ignore
			topview.AddLink(peerprocess.Key, localprocess.Key)
		} else if localerr!=nil && peererr==nil {
			var process = Process{Key: strconv.Itoa(pid), Name: name, Recvq: recvq, PID: pid, Sendq: sendq, Type: "process"}
			netInterfaces, err := net.Interfaces()
			if err != nil {
				fmt.Println("net.Interfaces failed, err:", err.Error())
				continue
			}
			for i := 0; i < len(netInterfaces); i++ {
				if (netInterfaces[i].Flags & net.FlagUp) != 0 {
					addrs, _ := netInterfaces[i].Addrs()
					for _, address := range addrs {
						if ipnet, ok := address.(*net.IPNet); ok {
							if ipnet.IP.To4()!=nil {
								process.AddIp(ipnet.IP.String())
							} else if ipnet.IP.To16()!=nil {
								process.AddIp("["+ipnet.IP.String()+"]")
							}
						}
					}
				}
			}

			topview.addNode(process)
			topview.AddLink(process.Key, peerprocess.Key)
		} else {
			fmt.Println("warning: ", line)
			continue
		}

	}

	return nil
}
