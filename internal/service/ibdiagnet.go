package service

import (
	"encoding/csv"
	"errors"
	"fmt"
	"log-parser/internal/model"
	"strconv"
	"strings"
)

var ErrParseLog = errors.New("invalid or unsupported log format")

func parseIBDiagExport(content []byte) (*model.ParsedLog, error) {
	text := string(content)
	nodesBody, ok := extractSection(text, "START_NODES", "END_NODES")
	if !ok {
		return nil, fmt.Errorf("%w: missing START_NODES/END_NODES", ErrParseLog)
	}
	portsBody, ok := extractSection(text, "START_PORTS", "END_PORTS")
	if !ok {
		return nil, fmt.Errorf("%w: missing START_PORTS/END_PORTS", ErrParseLog)
	}

	nodes, err := parseNodesCSV(nodesBody)
	if err != nil {
		return nil, err
	}

	portsByNode, err := parsePortsCSV(portsBody)
	if err != nil {
		return nil, err
	}

	for i := range nodes {
		key := normalizeGUIDKey(nodes[i].GUID)
		nodes[i].Ports = portsByNode[key]
	}

	var infos []model.ParsedNodeInfo
	if infoBody, ok := extractSection(text, "START_SYSTEM_GENERAL_INFORMATION", "END_SYSTEM_GENERAL_INFORMATION"); ok {
		infos, err = parseSystemInfoCSV(infoBody)
		if err != nil {
			return nil, err
		}
	}

	return &model.ParsedLog{Nodes: nodes, Infos: infos}, nil
}

func extractSection(text, start, end string) (string, bool) {
	i := strings.Index(text, start)
	if i < 0 {
		return "", false
	}
	from := i + len(start)
	j := strings.Index(text[from:], end)
	if j < 0 {
		return "", false
	}
	return strings.TrimSpace(text[from : from+j]), true
}

func parseNodesCSV(body string) ([]model.ParsedNode, error) {
	r := csv.NewReader(strings.NewReader(body))
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%w: nodes CSV: %v", ErrParseLog, err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("%w: nodes section has no rows", ErrParseLog)
	}
	idx := mapHeaders(records[0])
	descI := idx["NodeDesc"]
	guidI := idx["NodeGUID"]
	typeI := idx["NodeType"]
	if descI < 0 || guidI < 0 || typeI < 0 {
		return nil, fmt.Errorf("%w: nodes header missing NodeDesc/NodeGUID/NodeType", ErrParseLog)
	}

	var out []model.ParsedNode
	for _, rec := range records[1:] {
		if rowEmpty(rec) {
			continue
		}
		if !rowHas(rec, descI, guidI, typeI) {
			return nil, fmt.Errorf("%w: truncated nodes row", ErrParseLog)
		}
		name := strings.Trim(strings.TrimSpace(rec[descI]), `"`)
		guid := strings.TrimSpace(rec[guidI])
		nt, err := strconv.Atoi(strings.TrimSpace(rec[typeI]))
		if err != nil {
			return nil, fmt.Errorf("%w: node type: %v", ErrParseLog, err)
		}
		out = append(out, model.ParsedNode{
			Name: name,
			GUID: guid,
			Type: nodeTypeLabel(nt),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: no nodes parsed", ErrParseLog)
	}
	return out, nil
}

func parsePortsCSV(body string) (map[string][]model.ParsedPort, error) {
	r := csv.NewReader(strings.NewReader(body))
	r.LazyQuotes = true

	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%w: ports CSV: %v", ErrParseLog, err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("%w: ports section has no rows", ErrParseLog)
	}

	idx := mapHeaders(records[0])
	nodeGI := firstIdx(idx, "NodeGuid", "NodeGUID")
	portGI := firstIdx(idx, "PortGuid", "PortGUID")
	portNumI := firstIdx(idx, "PortNum")
	if portNumI < 0 {
		portNumI = idx["LocalPortNum"]
	}
	phyI := idx["PortPhyState"]
	stateI := idx["PortState"]
	msmlidI := idx["MSMLID"]
	lidI := idx["LID"]
	latI := idx["LinkRoundTripLatency"]

	if nodeGI < 0 || portGI < 0 || portNumI < 0 {
		return nil, fmt.Errorf("%w: ports header missing NodeGuid/PortGuid/PortNum", ErrParseLog)
	}

	out := make(map[string][]model.ParsedPort)
	for _, rec := range records[1:] {
		if rowEmpty(rec) {
			continue
		}
		minLen := 0
		for _, i := range []int{nodeGI, portGI, portNumI, msmlidI, lidI, stateI, latI} {
			if i >= 0 && i+1 > minLen {
				minLen = i + 1
			}
		}
		if len(rec) < minLen {
			return nil, fmt.Errorf("%w: truncated ports row", ErrParseLog)
		}
		ng := strings.TrimSpace(rec[nodeGI])
		pg := strings.TrimSpace(rec[portGI])
		pn, err := strconv.Atoi(strings.TrimSpace(rec[portNumI]))
		if err != nil {
			return nil, fmt.Errorf("%w: port number: %v", ErrParseLog, err)
		}
		state := joinStates(field(rec, phyI), field(rec, stateI))
		k := normalizeGUIDKey(ng)
		out[k] = append(out[k], model.ParsedPort{
			NodeGUID:    ng,
			PortGUID:    pg,
			PortNum:     pn,
			State:       state,
			LID:         parseIntPtrField(rec, lidI),
			MSMLID:      parseIntPtrField(rec, msmlidI),
			LinkLatency: parseIntPtrField(rec, latI),
			IBPortState: parseIntPtrField(rec, stateI),
		})
	}
	return out, nil
}

func parseIntPtrField(rec []string, i int) *int {
	if i < 0 || i >= len(rec) {
		return nil
	}
	s := strings.TrimSpace(rec[i])
	if s == "" || strings.EqualFold(s, "N/A") {
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &v
}

func parseSystemInfoCSV(body string) ([]model.ParsedNodeInfo, error) {
	r := csv.NewReader(strings.NewReader(body))
	r.LazyQuotes = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%w: system info CSV: %v", ErrParseLog, err)
	}
	if len(records) < 2 {
		return nil, nil
	}
	idx := mapHeaders(records[0])
	guidI := firstIdx(idx, "NodeGuid", "NodeGUID")
	if guidI < 0 {
		return nil, fmt.Errorf("%w: system info missing NodeGuid", ErrParseLog)
	}
	si := idx["SerialNumber"]
	pi := idx["PartNumber"]
	ri := idx["Revision"]
	pni := idx["ProductName"]

	var out []model.ParsedNodeInfo
	for _, rec := range records[1:] {
		if rowEmpty(rec) || !rowHas(rec, guidI) {
			continue
		}
		info := model.ParsedNodeInfo{NodeGUID: strings.TrimSpace(rec[guidI])}
		if si >= 0 && si < len(rec) {
			info.Serial = strings.Trim(strings.TrimSpace(rec[si]), `"`)
		}
		if pi >= 0 && pi < len(rec) {
			info.PartNumber = strings.Trim(strings.TrimSpace(rec[pi]), `"`)
		}
		if ri >= 0 && ri < len(rec) {
			info.Revision = strings.Trim(strings.TrimSpace(rec[ri]), `"`)
		}
		if pni >= 0 && pni < len(rec) {
			info.ProductName = strings.Trim(strings.TrimSpace(rec[pni]), `"`)
		}
		out = append(out, info)
	}
	return out, nil
}

func mapHeaders(header []string) map[string]int {
	m := make(map[string]int)
	for i, h := range header {
		m[strings.TrimSpace(h)] = i
	}
	return m
}

func firstIdx(m map[string]int, keys ...string) int {
	for _, k := range keys {
		if i, ok := m[k]; ok {
			return i
		}
	}
	return -1
}

func nodeTypeLabel(nt int) string {
	switch nt {
	case 1:
		return "host"
	case 2:
		return "switch"
	default:
		return fmt.Sprintf("node-type-%d", nt)
	}
}

func normalizeGUIDKey(g string) string {
	g = strings.TrimSpace(strings.ToLower(g))
	if strings.HasPrefix(g, "0x") {
		return "0x" + g[2:]
	}
	return g
}

func rowEmpty(rec []string) bool {
	for _, c := range rec {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func rowHas(rec []string, idxs ...int) bool {
	for _, i := range idxs {
		if i < 0 || i >= len(rec) {
			return false
		}
	}
	return true
}

func field(rec []string, i int) string {
	if i < 0 || i >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[i])
}

func joinStates(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	default:
		return a + "/" + b
	}
}
