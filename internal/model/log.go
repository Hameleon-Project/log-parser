package model

import "time"

// LogFilter содержит параметры для фильтрации и пагинации
type LogFilter struct {
	Level  string
	Limit  int
	Offset int
}

type LogEntry struct {
	ID        int       `json:"id"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type Log struct {
	ID        int       `json:"id"`
	Filename  string    `json:"filename"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Node struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Ports []Port `json:"ports"`
}

type Port struct {
	ID          int    `json:"id"`
	NodeID      int    `json:"node_id"`
	Name        string `json:"name"`
	MAC         string `json:"mac"`
	ConnectedTo *int   `json:"connected_to,omitempty"`
}

type Topology struct {
	Nodes []Node `json:"nodes"`
}
