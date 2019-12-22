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
	netstatBinary  = "netstat"
	tasklistBinary = "tasklist"
)

func (walker) Walk() (Topview, error) {

	var topview = Topview{}

	processes, err := ParseTasklist()
	if err != nil {
		return Topview{}, err
	}

	for _, process := range processes {
		topview.addNode(process)
	}

	processes, topology, err := ParseNETSTAT()
	if err != nil {
		return Topview{}, err
	}
	for _, process := range processes {
		topview.addNode(process)
	}

	for _, str := range topology {
		addresses := strings.SplitN(str, "->", 2)
		fromPID, err := strconv.Atoi(addresses[0])
		if err != nil {
			return Topview{}, fmt.Errorf("invalid field in pid: %#v", addresses[0])
		}
		toAddr := strings.SplitN(addresses[1], ":", 2)
		toPort, err := strconv.Atoi(toAddr[1])
		if err != nil {
			return Topview{}, fmt.Errorf("invalid field in addr: %#v", addresses[1])
		}
		fromprocess, err := topview.GetNodeByPid(fromPID)
		if err != nil {
			continue
		}
		toprocess, err := topview.GetNodeByIpAndPort(toAddr[0], toPort)
		if err != nil {
			continue
		}
		if fromprocess.PID != toprocess.PID {
			topview.AddLink(fromprocess.Key, toprocess.Key)
		} else if fromprocess.PID == 0 {
			topview.AddLink(fromprocess.Key, toprocess.Key)
		}
	}
	return topview, nil

}

func ParseNETSTAT() ([]Process, []string, error) {

	outputBinary, err := exec.Command(
		netstatBinary,
		"-a",
		"-n", "-o",
		"-p", "tcp",
	).CombinedOutput()
	if err != nil {
		return nil, nil, err
	}
	output := mahonia.NewDecoder("gb18030").ConvertString(string(outputBinary))
	var (
		processes []Process
		topology  []string
	)
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if (len(fields) < 1) || !strings.EqualFold(strings.ToUpper(fields[0]), "TCP") {
			continue
		}

		if len(fields) != 5 {
			return nil, nil, fmt.Errorf("invalid field in netstat output: %#v", line)
		}

		var process Process = Process{Type: "process"}
		pid, err := strconv.Atoi(fields[4])
		if err != nil {
			return nil, nil, fmt.Errorf("invalid field in netstat output: %#v", line)
		}
		addr := strings.SplitN(fields[1], ":", 2)
		port, err := strconv.Atoi(addr[1])
		if err != nil {
			return nil, nil, fmt.Errorf("invalid field in netstat output: %#v", line)
		}

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
						if ipnet.IP.To4() != nil {
							process.AddIp(ipnet.IP.String())
						}
					}
				}
			}
		}
		process.AddPort(port)
		process.PID = pid
		process.Key = fields[4]
		if pid != 0 {
			processes = append(processes, process)
		} else {
			continue
		}

		if strings.EqualFold(fields[3], "LISTENING") {
		} else {
			var process = Process{Type: "process"}
			addr := strings.SplitN(fields[2], ":", 2)
			port, err := strconv.Atoi(addr[1])
			if err != nil {
				return nil, nil, fmt.Errorf("invalid field in netstat output: %#v", line)
			}
			topology = append(topology, strings.Join([]string{strconv.Itoa(pid), fields[2]}, "->"))
			process.AddIp(addr[0])
			process.AddPort(port)
			processes = append(processes, process)
		}
	}
	return processes, topology, nil
}

func ParseTasklist() ([]Process, error) {

	outputBinary, err := exec.Command(
		tasklistBinary,
		"/NH",
		"/FO", "CSV",
	).CombinedOutput()
	if err != nil {
		return nil, err
	}
	output := mahonia.NewDecoder("utf-8").ConvertString(string(outputBinary))

	var processes []Process
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Split(line, "\",\"")

		if len(fields) != 5 {
			continue
		}

		var process Process = Process{Type: "process"}

		pid, err := strconv.Atoi(strings.Trim(fields[1], "\"\r"))
		if err != nil {
			continue
		}
		process.PID = pid
		process.Name = strings.Trim(fields[0], "\"\r")
		process.Memory = strings.Trim(fields[4], "\"\r")
		processes = append(processes, process)
	}
	return processes, nil
}
