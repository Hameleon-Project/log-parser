package service

import (
	"log-parser/internal/model"
	"sort"
	"strconv"
	"strings"
)

const ibPortActive = 4

func InferPortLinks(parsed *model.ParsedLog) []model.InferredLink {
	if parsed == nil {
		return nil
	}
	isSwitch := make(map[string]bool)
	for _, n := range parsed.Nodes {
		isSwitch[normalizeGUIDKey(n.GUID)] = n.Type == "switch"
	}

	type flat struct {
		node string
		pn   int
		lat  *int
		ib   *int
	}
	var flats []flat
	for _, n := range parsed.Nodes {
		g := normalizeGUIDKey(n.GUID)
		for _, p := range n.Ports {
			flats = append(flats, flat{g, p.PortNum, p.LinkLatency, p.IBPortState})
		}
	}

	active := func(f flat) bool {
		return f.ib != nil && *f.ib == ibPortActive
	}

	seen := map[string]struct{}{}
	var out []model.InferredLink
	addLink := func(fromG string, fromP int, toG string, toP int, kind string) {
		k1 := edgeKey(fromG, fromP, toG, toP)
		k2 := edgeKey(toG, toP, fromG, fromP)
		if _, ok := seen[k1]; ok {
			return
		}
		if _, ok := seen[k2]; ok {
			return
		}
		seen[k1] = struct{}{}
		out = append(out, model.InferredLink{
			FromNodeGUID: fromG, FromPortNum: fromP,
			ToNodeGUID: toG, ToPortNum: toP,
			Kind: kind,
		})
	}

	byLat := make(map[int][]flat)
	for _, f := range flats {
		if !active(f) || f.lat == nil || *f.lat <= 0 {
			continue
		}
		byLat[*f.lat] = append(byLat[*f.lat], f)
	}
	for lat, list := range byLat {
		if len(list) != 2 {
			continue
		}
		x, y := list[0], list[1]
		if x.node == y.node {
			continue
		}
		addLink(x.node, x.pn, y.node, y.pn, "link_round_trip_latency:"+strconv.Itoa(lat))
	}

	var isl []flat
	for _, f := range flats {
		if !active(f) || f.pn != 65 || !isSwitch[f.node] {
			continue
		}
		if f.lat == nil || *f.lat != 0 {
			continue
		}
		isl = append(isl, f)
	}
	if len(isl) >= 2 {
		sort.Slice(isl, func(i, j int) bool {
			if isl[i].node == isl[j].node {
				return isl[i].pn < isl[j].pn
			}
			return isl[i].node < isl[j].node
		})
		var uniq []flat
		last := ""
		for _, f := range isl {
			if f.node == last {
				continue
			}
			last = f.node
			uniq = append(uniq, f)
		}
		if len(uniq) >= 2 {
			for i := 0; i < len(uniq); i++ {
				a := uniq[i]
				b := uniq[(i+1)%len(uniq)]
				addLink(a.node, a.pn, b.node, b.pn, "switch_isl_ring_port65")
			}
		}
	}

	return out
}

func edgeKey(a string, ap int, b string, bp int) string {
	return strings.Join([]string{a, strconv.Itoa(ap), b, strconv.Itoa(bp)}, "|")
}
