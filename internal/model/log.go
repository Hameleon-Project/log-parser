package model

import "time"

type ParsedLog struct {
	Nodes []ParsedNode
	Infos []ParsedNodeInfo
}

type ParsedNode struct {
	Name  string
	Type  string
	GUID  string
	Ports []ParsedPort
}

type ParsedPort struct {
	NodeGUID    string
	PortGUID    string
	PortNum     int
	State       string
	LID         *int
	MSMLID      *int
	LinkLatency *int
	IBPortState *int
}

type ParsedNodeInfo struct {
	NodeGUID    string
	Serial      string
	PartNumber  string
	Revision    string
	ProductName string
}

type Node struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	GUID        string `json:"guid"`
	Ports       []Port `json:"ports,omitempty"`
	Serial      string `json:"serial_number,omitempty"`
	PartNumber  string `json:"part_number,omitempty"`
	HardwareRev string `json:"hardware_rev,omitempty"`
	ProductName string `json:"product_name,omitempty"`
}

type Port struct {
	ID       int    `json:"id"`
	NodeID   int    `json:"node_id"`
	Name     string `json:"name"`
	PortGUID string `json:"port_guid"`
	PortNum  int    `json:"port_num"`
	State    string `json:"state,omitempty"`
}

type InferredLink struct {
	FromNodeGUID string
	FromPortNum  int
	ToNodeGUID   string
	ToPortNum    int
	Kind         string
}

type TopologyEdge struct {
	FromNodeID int    `json:"from_node_id"`
	ToNodeID   int    `json:"to_node_id"`
	FromPortID int    `json:"from_port_id"`
	ToPortID   int    `json:"to_port_id"`
	Kind       string `json:"kind"`
}

type TopologyGroup struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	NodeIDs []int  `json:"node_ids"`
}

type Topology struct {
	Nodes          []Node          `json:"nodes"`
	Edges          []TopologyEdge  `json:"edges"`
	TopologyGroups []TopologyGroup `json:"topology_groups"`
}

type LogMeta struct {
	ID         int       `json:"id"`
	Status     string    `json:"status"`
	FilePath   string    `json:"file_path"`
	NodesCount int       `json:"nodes_count"`
	PortsCount int       `json:"ports_count"`
	CreatedAt  time.Time `json:"created_at"`
}
