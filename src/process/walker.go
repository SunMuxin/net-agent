package process

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Process represents a single process.
type Process struct {
	Key    string   `json:"key"`
	PID    int      `json:"pid"`
	Name   string   `json:"name"`
	Ips    []string `json:"ips"`
	Ports  []int    `json:"port"`
	Memory string   `json:"memory"`
	Type   string   `json:"type"`
	Recvq  int      `json:"recvq"`
	Sendq  int      `json:"sendq"`
}

func (p *Process) fusion(process Process) {
	if p.Name == "" {
		p.Name = process.Name
	}
	if p.Memory == "" {
		p.Memory = process.Memory
	}
	if p.PID == 0 {
		p.PID = process.PID
	}
	for _, ip := range process.Ips {
		if !p.ContainIp(ip) {
			p.Ips = append(p.Ips, ip)
		}
	}
	for _, port := range process.Ports {
		if !p.ContainPort(port) {
			p.Ports = append(p.Ports, port)
		}
	}
}

func (p *Process) ContainIp(ip string) bool {
	for _, i := range p.Ips {
		if strings.EqualFold(i, ip) {
			return true
		}
	}
	return false
}

func (p *Process) ContainPort(port int) bool {
	for _, pt := range p.Ports {
		if port == pt {
			return true
		}
	}
	return false
}

func (p *Process) AddIp(ip string) {
	if !p.ContainIp(ip) {
		p.Ips = append(p.Ips, ip)
	}
}

func (p *Process) AddPort(port int) {
	if !p.ContainPort(port) {
		p.Ports = append(p.Ports, port)
	}
}

func (p *Process) Equals(process Process) bool {
	if process.PID != p.PID && process.PID != 0 && p.PID != 0 {
		return false
	} else if process.PID == p.PID && p.PID != 0 {
		return true
	} else if len(process.Ips) == 0 || len(process.Ports) == 0 ||
		len(p.Ips) == 0 || len(p.Ports) == 0 {
		return false
	} else if len(p.Ips) >= len(process.Ips) {
		for _, ip := range process.Ips {
			if ok := p.ContainIp(ip); !ok {
				return false
			}
		}
		for _, port := range process.Ports {
			if ok := p.ContainPort(port); !ok {
				return false
			}
		}
	} else {
		for _, ip := range p.Ips {
			if ok := process.ContainIp(ip); !ok {
				return false
			}
		}
		for _, port := range p.Ports {
			if ok := process.ContainPort(port); !ok {
				return false
			}
		}
	}
	return true

}

type Link struct {
	From string `json:"from"`
	To   string `json:"to"`
	Flow string `json:"flow"`
}

func (l *Link) Equals(link Link) bool {
	return strings.EqualFold(l.From, link.From) && strings.EqualFold(l.To, link.To)
}

type Topview struct {
	Nodes []Process `json:"nodes"`
	Links []Link    `json:"links"`
}

type TopviewMessage struct {
	Topview Topview `json:"message"`
	Code    int     `json:"code"`
}

func (t *Topview) reduce() {
	for {
		isReduce := true
		nodes := []Process{}
		for _, process := range t.Nodes {
			flag := true
			for index, _ := range nodes {
				if nodes[index].Equals(process) {
					nodes[index].fusion(process)
					isReduce, flag = false, false
				}
			}
			if flag {
				nodes = append(nodes, process)
			}
		}
		t.Nodes = nodes
		if isReduce {
			break
		}
	}
}

func (t *Topview) addNode(process Process) {
	for index, _ := range t.Nodes {
		if t.Nodes[index].Equals(process) {
			t.Nodes[index].fusion(process)
			t.reduce()
			return
		}
	}
	t.Nodes = append(t.Nodes, process)
}

func (t *Topview) AddLink(from string, to string) {
	for _, link := range t.Links {
		if strings.EqualFold(link.From, from) && strings.EqualFold(link.To, to) {
			return
		}
	}
	t.Links = append(t.Links, Link{From: from, To: to, Flow: "tcp connect"})
}

func (t *Topview) GetNodeByPid(pid int) (Process, error) {
	for _, node := range t.Nodes {
		if node.PID == pid {
			return node, nil
		}
	}
	return Process{}, fmt.Errorf("no exist : pid =  %#v", pid)
}

func (t *Topview) GetNodeByIpAndPort(ip string, port int) (Process, error) {
	for _, node := range t.Nodes {
		if node.ContainIp(ip) && node.ContainPort(port) {
			return node, nil
		}
	}
	return Process{}, fmt.Errorf("no exist : ip, port =  %#v %#v", ip, port)
}

func (t *Topview) GetMessage(connection bool) string {
	nodes := []Process{}
	links := []Link{}
	if connection {
		links = t.Links
		for _, node := range t.Nodes {
			isconn := false
			for _, link := range t.Links {
				if strings.EqualFold(link.From, node.Key) || strings.EqualFold(link.To, node.Key) {
					isconn = true
				}
			}
			if isconn {
				nodes = append(nodes, node)
			}
		}
	} else {
		nodes = t.Nodes
		links = t.Links
	}
	topview := Topview{Nodes: nodes, Links: links}
	bMessage, err := json.Marshal(&TopviewMessage{Topview:topview, Code:200})
	if err != nil {
		fmt.Println("error:", err)
	}
	return string(bMessage)
}

type Walker interface {
	Walk() (Topview, error)
}
