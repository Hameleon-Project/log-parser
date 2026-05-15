package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log-parser/internal/model"
	"sort"
	"strconv"
	"strings"
)

type LogRepository interface {
	SaveParsedLog(ctx context.Context, filePath string, parsed *model.ParsedLog, links []model.InferredLink) (int, error)
	GetPortsByNodeID(ctx context.Context, nodeID int) ([]model.Port, error)
	GetLogByID(ctx context.Context, logID int) (*model.LogMeta, error)
	GetTopology(ctx context.Context, logID int) (*model.Topology, error)
	GetNodeByID(ctx context.Context, nodeID int) (*model.Node, error)
}

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *PostgresRepo {
	return &PostgresRepo{db: db}
}

func (r *PostgresRepo) SaveParsedLog(ctx context.Context, filePath string, parsed *model.ParsedLog, links []model.InferredLink) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	var logID int
	err = tx.QueryRowContext(ctx,
		`INSERT INTO logs (status, file_path, nodes_count, ports_count) VALUES ($1, $2, 0, 0) RETURNING id`,
		"parsed", filePath,
	).Scan(&logID)
	if err != nil {
		return 0, fmt.Errorf("insert log: %w", err)
	}

	guidToNodeID := make(map[string]int)
	portIDs := make(map[string]int)

	for _, n := range parsed.Nodes {
		var nodeID int
		err = tx.QueryRowContext(ctx,
			`INSERT INTO nodes (log_id, name, type, guid) VALUES ($1, $2, $3, $4) RETURNING id`,
			logID, n.Name, n.Type, n.GUID,
		).Scan(&nodeID)
		if err != nil {
			return 0, fmt.Errorf("insert node %q: %w", n.Name, err)
		}
		guidToNodeID[normalizeGUID(n.GUID)] = nodeID

		for _, p := range n.Ports {
			var pid int
			err = tx.QueryRowContext(ctx, `
				INSERT INTO ports (node_id, port_guid, port_num, state, lid, msmlid, link_latency, ib_port_state)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				ON CONFLICT (node_id, port_num) DO UPDATE SET
					port_guid = EXCLUDED.port_guid,
					state = EXCLUDED.state,
					lid = EXCLUDED.lid,
					msmlid = EXCLUDED.msmlid,
					link_latency = EXCLUDED.link_latency,
					ib_port_state = EXCLUDED.ib_port_state
				RETURNING id`,
				nodeID, p.PortGUID, p.PortNum, nullIfEmpty(p.State),
				intPtrSQL(p.LID), intPtrSQL(p.MSMLID), intPtrSQL(p.LinkLatency), intPtrSQL(p.IBPortState),
			).Scan(&pid)
			if err != nil {
				return 0, fmt.Errorf("insert port %d on node %q: %w", p.PortNum, n.Name, err)
			}
			portIDs[portMapKey(n.GUID, p.PortNum)] = pid
		}
	}

	for _, info := range parsed.Infos {
		nodeID, ok := guidToNodeID[normalizeGUID(info.NodeGUID)]
		if !ok {
			continue
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO nodes_info (node_id, description, hardware_rev, serial_number, part_number)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (node_id) DO UPDATE SET
				description = EXCLUDED.description,
				hardware_rev = EXCLUDED.hardware_rev,
				serial_number = EXCLUDED.serial_number,
				part_number = EXCLUDED.part_number`,
			nodeID, nullIfEmpty(info.ProductName), nullIfEmpty(info.Revision),
			nullIfEmpty(info.Serial), nullIfEmpty(info.PartNumber),
		)
		if err != nil {
			return 0, fmt.Errorf("insert nodes_info: %w", err)
		}
	}

	for _, lk := range links {
		a, okA := portIDs[portMapKey(lk.FromNodeGUID, lk.FromPortNum)]
		b, okB := portIDs[portMapKey(lk.ToNodeGUID, lk.ToPortNum)]
		if !okA || !okB {
			continue
		}
		from, to := a, b
		if from > to {
			from, to = to, from
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO links (log_id, from_port_id, to_port_id, kind)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (log_id, from_port_id, to_port_id) DO NOTHING`,
			logID, from, to, lk.Kind,
		)
		if err != nil {
			return 0, fmt.Errorf("insert link: %w", err)
		}
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE logs SET
			nodes_count = (SELECT COUNT(*) FROM nodes WHERE log_id = $1),
			ports_count = (SELECT COUNT(*) FROM ports WHERE node_id IN (SELECT id FROM nodes WHERE log_id = $1))
		WHERE id = $1`, logID)
	if err != nil {
		return 0, fmt.Errorf("update log counts: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return logID, nil
}

func portMapKey(guid string, portNum int) string {
	return normalizeGUID(guid) + "|" + strconv.Itoa(portNum)
}

func intPtrSQL(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func normalizeGUID(g string) string {
	g = strings.TrimSpace(strings.ToLower(g))
	if strings.HasPrefix(g, "0x") {
		return "0x" + g[2:]
	}
	return g
}

func (r *PostgresRepo) GetTopology(ctx context.Context, logID int) (*model.Topology, error) {
	nodeQuery := `SELECT id, name, type, guid FROM nodes WHERE log_id = $1 ORDER BY id`
	rows, err := r.db.QueryContext(ctx, nodeQuery, logID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var topo model.Topology
	topo.TopologyGroups = []model.TopologyGroup{}
	topo.Edges = []model.TopologyEdge{}
	nodeIndex := make(map[int]int)

	for rows.Next() {
		var n model.Node
		if err := rows.Scan(&n.ID, &n.Name, &n.Type, &n.GUID); err != nil {
			return nil, err
		}
		nodeIndex[n.ID] = len(topo.Nodes)
		topo.Nodes = append(topo.Nodes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var allIDs []int
	for _, n := range topo.Nodes {
		allIDs = append(allIDs, n.ID)
	}

	portQuery := `
		SELECT id, node_id, port_guid, port_num, COALESCE(state, '')
		FROM ports WHERE node_id IN (SELECT id FROM nodes WHERE log_id = $1)
		ORDER BY node_id, port_num`

	pRows, err := r.db.QueryContext(ctx, portQuery, logID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = pRows.Close() }()

	for pRows.Next() {
		var p model.Port
		if err := pRows.Scan(&p.ID, &p.NodeID, &p.PortGUID, &p.PortNum, &p.State); err != nil {
			return nil, err
		}
		p.Name = fmt.Sprintf("port-%d", p.PortNum)
		if idx, ok := nodeIndex[p.NodeID]; ok {
			topo.Nodes[idx].Ports = append(topo.Nodes[idx].Ports, p)
		}
	}
	if err := pRows.Err(); err != nil {
		return nil, err
	}

	linkQuery := `
		SELECT l.from_port_id, l.to_port_id, l.kind, p1.node_id, p2.node_id
		FROM links l
		JOIN ports p1 ON p1.id = l.from_port_id
		JOIN ports p2 ON p2.id = l.to_port_id
		WHERE l.log_id = $1`
	lRows, err := r.db.QueryContext(ctx, linkQuery, logID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = lRows.Close() }()

	for lRows.Next() {
		var e model.TopologyEdge
		if err := lRows.Scan(&e.FromPortID, &e.ToPortID, &e.Kind, &e.FromNodeID, &e.ToNodeID); err != nil {
			return nil, err
		}
		topo.Edges = append(topo.Edges, e)
	}
	if err := lRows.Err(); err != nil {
		return nil, err
	}

	topo.TopologyGroups = buildTopologyGroups(allIDs, topo.Edges)
	return &topo, nil
}

func buildTopologyGroups(nodeIDs []int, edges []model.TopologyEdge) []model.TopologyGroup {
	if len(nodeIDs) == 0 {
		return []model.TopologyGroup{}
	}
	sort.Ints(nodeIDs)
	if len(edges) == 0 {
		return []model.TopologyGroup{
			{ID: "fabric-all", Name: "all_nodes_default", NodeIDs: append([]int(nil), nodeIDs...)},
		}
	}

	adj := make(map[int][]int)
	for _, e := range edges {
		adj[e.FromNodeID] = append(adj[e.FromNodeID], e.ToNodeID)
		adj[e.ToNodeID] = append(adj[e.ToNodeID], e.FromNodeID)
	}

	seen := make(map[int]bool)
	var groups []model.TopologyGroup
	gid := 0
	for _, start := range nodeIDs {
		if seen[start] {
			continue
		}
		gid++
		comp := bfsComponent(start, adj, seen)
		sort.Ints(comp)
		groups = append(groups, model.TopologyGroup{
			ID:      fmt.Sprintf("component-%d", gid),
			Name:    "connected_component",
			NodeIDs: comp,
		})
	}
	return groups
}

func bfsComponent(start int, adj map[int][]int, seen map[int]bool) []int {
	q := []int{start}
	seen[start] = true
	var comp []int
	for i := 0; i < len(q); i++ {
		v := q[i]
		comp = append(comp, v)
		for _, w := range adj[v] {
			if !seen[w] {
				seen[w] = true
				q = append(q, w)
			}
		}
	}
	return comp
}

func (r *PostgresRepo) GetNodeByID(ctx context.Context, nodeID int) (*model.Node, error) {
	query := `
		SELECT n.id, n.name, n.type, n.guid,
			COALESCE(ni.serial_number, ''), COALESCE(ni.part_number, ''),
			COALESCE(ni.hardware_rev, ''), COALESCE(ni.description, '')
		FROM nodes n
		LEFT JOIN nodes_info ni ON ni.node_id = n.id
		WHERE n.id = $1`
	var n model.Node
	err := r.db.QueryRowContext(ctx, query, nodeID).Scan(
		&n.ID, &n.Name, &n.Type, &n.GUID,
		&n.Serial, &n.PartNumber, &n.HardwareRev, &n.ProductName,
	)
	if err != nil {
		return nil, err
	}

	portQuery := `SELECT id, node_id, port_guid, port_num, COALESCE(state, '') FROM ports WHERE node_id = $1 ORDER BY port_num`
	rows, err := r.db.QueryContext(ctx, portQuery, nodeID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var p model.Port
		if err := rows.Scan(&p.ID, &p.NodeID, &p.PortGUID, &p.PortNum, &p.State); err != nil {
			return nil, err
		}
		p.Name = fmt.Sprintf("port-%d", p.PortNum)
		n.Ports = append(n.Ports, p)
	}
	return &n, rows.Err()
}

func (r *PostgresRepo) GetPortsByNodeID(ctx context.Context, nodeID int) ([]model.Port, error) {
	query := `SELECT id, node_id, port_guid, port_num, COALESCE(state, '') FROM ports WHERE node_id = $1 ORDER BY port_num`
	rows, err := r.db.QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var ports []model.Port
	for rows.Next() {
		var p model.Port
		if err := rows.Scan(&p.ID, &p.NodeID, &p.PortGUID, &p.PortNum, &p.State); err != nil {
			return nil, err
		}
		p.Name = fmt.Sprintf("port-%d", p.PortNum)
		ports = append(ports, p)
	}
	return ports, rows.Err()
}

func (r *PostgresRepo) GetLogByID(ctx context.Context, logID int) (*model.LogMeta, error) {
	var meta model.LogMeta
	query := `
		SELECT id, status, COALESCE(file_path, ''), nodes_count, ports_count, created_at
		FROM logs WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, logID).Scan(
		&meta.ID, &meta.Status, &meta.FilePath, &meta.NodesCount, &meta.PortsCount, &meta.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}
