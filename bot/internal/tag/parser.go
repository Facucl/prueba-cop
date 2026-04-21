package tag

import (
	"bufio"
	"fmt"
	"sort"
	"strings"
)

// canonicalOrder define el orden de aparición en el mensaje cuando el env
// es uno de los conocidos. Cualquier env no listado va al final, alfabético.
var canonicalOrder = []string{"desa", "test", "preprod", "prod"}

// ParseList normaliza un string (space/comma-separated, any case) a la lista
// de ambientes únicos en orden canónico. Acepta cualquier nombre de env — sin
// whitelist — para soportar futuras etiquetas (prod, stage, etc.).
func ParseList(raw string) ([]string, error) {
	seen := map[string]bool{}
	for _, tok := range splitEnvs(raw) {
		env := strings.ToLower(strings.TrimSpace(tok))
		if env != "" {
			seen[env] = true
		}
	}
	if len(seen) == 0 {
		return nil, fmt.Errorf("no env listed in %q", raw)
	}
	out := make([]string, 0, len(seen))
	for e := range seen {
		out = append(out, e)
	}
	return SortCanonical(out), nil
}

// ParseRequestedEnvs busca una línea `deploy: <env> <env> ...` en el mensaje
// del tag (case-insensitive) y devuelve los envs en orden canónico.
func ParseRequestedEnvs(tagMessage string) ([]string, error) {
	seen := map[string]bool{}
	found := false

	scanner := bufio.NewScanner(strings.NewReader(tagMessage))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lower := strings.ToLower(line)
		if !strings.HasPrefix(lower, "deploy:") {
			continue
		}
		found = true
		rest := strings.TrimSpace(line[len("deploy:"):])
		for _, tok := range splitEnvs(rest) {
			env := strings.ToLower(strings.TrimSpace(tok))
			if env != "" {
				seen[env] = true
			}
		}
	}
	if !found {
		return nil, fmt.Errorf("no 'deploy:' line found in tag message")
	}
	if len(seen) == 0 {
		return nil, fmt.Errorf("'deploy:' line found but no env listed")
	}
	out := make([]string, 0, len(seen))
	for e := range seen {
		out = append(out, e)
	}
	return SortCanonical(out), nil
}

// SortCanonical devuelve una copia ordenada: primero los envs conocidos en el
// orden de canonicalOrder, después el resto alfabético.
func SortCanonical(envs []string) []string {
	rank := make(map[string]int, len(canonicalOrder))
	for i, e := range canonicalOrder {
		rank[e] = i
	}
	out := make([]string, len(envs))
	copy(out, envs)
	sort.SliceStable(out, func(i, j int) bool {
		ri, oki := rank[out[i]]
		rj, okj := rank[out[j]]
		if oki && okj {
			return ri < rj
		}
		if oki {
			return true
		}
		if okj {
			return false
		}
		return out[i] < out[j]
	})
	return out
}

func splitEnvs(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		return r == ' ' || r == ',' || r == '\t' || r == '\n'
	})
}
