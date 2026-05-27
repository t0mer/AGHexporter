package instances

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
)

// ResolvePort returns the effective port: ADGUARD_EXPORTER_PORT env var wins over flagPort.
func ResolvePort(flagPort int) int {
	if v := os.Getenv("ADGUARD_EXPORTER_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			return p
		}
	}
	return flagPort
}

// DiscoverInstances combines Format A (indexed env), B (CSV env), C (--instance flags).
func DiscoverInstances(instanceFlags []string) ([]Instance, error) {
	var all []Instance

	a, err := discoverFormatA()
	if err != nil {
		return nil, err
	}
	all = append(all, a...)

	b, err := discoverFormatB()
	if err != nil {
		return nil, err
	}
	all = append(all, b...)

	c, err := discoverFormatC(instanceFlags)
	if err != nil {
		return nil, err
	}
	all = append(all, c...)

	if len(all) == 0 {
		return nil, fmt.Errorf(
			"no instances configured; declare at least one using:\n" +
				"  Format A: ADGUARD_URL_1=http://host ADGUARD_USERNAME_1=u ADGUARD_PASSWORD_1=p\n" +
				"  Format B: ADGUARD_URLS=http://host ADGUARD_USERNAMES=u ADGUARD_PASSWORDS=p\n" +
				"  Format C: --instance url=http://host,username=u,password=p",
		)
	}

	seen := make(map[string]bool, len(all))
	for _, inst := range all {
		if seen[inst.Name] {
			return nil, fmt.Errorf(
				"duplicate instance name %q; set ADGUARD_NAME_<N> or name= explicitly to disambiguate",
				inst.Name,
			)
		}
		seen[inst.Name] = true
	}

	for _, inst := range all {
		if err := inst.Validate(); err != nil {
			return nil, err
		}
	}

	return all, nil
}

// discoverFormatA scans ADGUARD_URL_<N> keys and builds instances sorted by index.
func discoverFormatA() ([]Instance, error) {
	var indices []int
	for _, env := range os.Environ() {
		key, _, ok := strings.Cut(env, "=")
		if !ok || !strings.HasPrefix(key, "ADGUARD_URL_") {
			continue
		}
		suffix := strings.TrimPrefix(key, "ADGUARD_URL_")
		n, err := strconv.Atoi(suffix)
		if err != nil || n < 1 {
			continue
		}
		indices = append(indices, n)
	}
	sort.Ints(indices)

	var out []Instance
	for _, n := range indices {
		ns := strconv.Itoa(n)
		rawURL := os.Getenv("ADGUARD_URL_" + ns)
		if rawURL == "" {
			continue
		}

		username, err := resolveCredential(
			"ADGUARD_USERNAME_"+ns, "ADGUARD_USERNAME_FILE_"+ns,
			fmt.Sprintf("Format A index %d username", n),
		)
		if err != nil {
			return nil, err
		}

		password, err := resolveCredential(
			"ADGUARD_PASSWORD_"+ns, "ADGUARD_PASSWORD_FILE_"+ns,
			fmt.Sprintf("Format A index %d password", n),
		)
		if err != nil {
			return nil, err
		}

		skipTLS := true
		if v := os.Getenv("ADGUARD_SKIP_TLS_" + ns); v != "" {
			skipTLS, _ = strconv.ParseBool(v)
		}

		inst, err := buildInstance(rawURL, os.Getenv("ADGUARD_NAME_"+ns), username, password, skipTLS)
		if err != nil {
			return nil, fmt.Errorf("Format A index %d: %w", n, err)
		}
		out = append(out, inst)
	}
	return out, nil
}

// discoverFormatB reads ADGUARD_URLS / ADGUARD_USERNAMES / ADGUARD_PASSWORDS CSV vars.
func discoverFormatB() ([]Instance, error) {
	rawURLs := os.Getenv("ADGUARD_URLS")
	if rawURLs == "" {
		return nil, nil
	}

	urlList := splitCSV(rawURLs)
	userList := splitCSV(os.Getenv("ADGUARD_USERNAMES"))
	passList := splitCSV(os.Getenv("ADGUARD_PASSWORDS"))
	n := len(urlList)

	if len(userList) != n || len(passList) != n {
		return nil, fmt.Errorf(
			"Format B: ADGUARD_URLS has %d entries but ADGUARD_USERNAMES has %d and ADGUARD_PASSWORDS has %d; all must match",
			n, len(userList), len(passList),
		)
	}

	nameList := make([]string, n)
	if v := os.Getenv("ADGUARD_NAMES"); v != "" {
		parts := splitCSV(v)
		if len(parts) != n {
			return nil, fmt.Errorf(
				"Format B: ADGUARD_NAMES has %d entries but ADGUARD_URLS has %d", len(parts), n,
			)
		}
		nameList = parts
	}

	skipList := make([]bool, n)
	for i := range skipList {
		skipList[i] = true
	}
	if v := os.Getenv("ADGUARD_SKIP_TLS"); v != "" {
		parts := splitCSV(v)
		if len(parts) != n {
			return nil, fmt.Errorf(
				"Format B: ADGUARD_SKIP_TLS has %d entries but ADGUARD_URLS has %d", len(parts), n,
			)
		}
		for i, p := range parts {
			skipList[i], _ = strconv.ParseBool(p)
		}
	}

	out := make([]Instance, 0, n)
	for i := range urlList {
		inst, err := buildInstance(urlList[i], nameList[i], userList[i], passList[i], skipList[i])
		if err != nil {
			return nil, fmt.Errorf("Format B index %d: %w", i+1, err)
		}
		out = append(out, inst)
	}
	return out, nil
}

// discoverFormatC parses --instance flag values (each is "key=value,...").
func discoverFormatC(flags []string) ([]Instance, error) {
	out := make([]Instance, 0, len(flags))
	for i, flag := range flags {
		inst, err := parseInstanceFlag(flag)
		if err != nil {
			return nil, fmt.Errorf("--instance flag %d: %w", i+1, err)
		}
		out = append(out, inst)
	}
	return out, nil
}

func parseInstanceFlag(s string) (Instance, error) {
	kv := make(map[string]string)
	for _, part := range strings.Split(s, ",") {
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			return Instance{}, fmt.Errorf("invalid key=value pair %q", part)
		}
		kv[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}

	rawURL := kv["url"]
	if rawURL == "" {
		return Instance{}, fmt.Errorf("url is required")
	}

	username, err := resolveCredentialFromMap(kv, "username", "username_file", "username")
	if err != nil {
		return Instance{}, err
	}
	password, err := resolveCredentialFromMap(kv, "password", "password_file", "password")
	if err != nil {
		return Instance{}, err
	}

	skipTLS := true
	if v, ok := kv["skip_tls"]; ok {
		skipTLS, _ = strconv.ParseBool(v)
	}

	return buildInstance(rawURL, kv["name"], username, password, skipTLS)
}

// resolveCredential returns the credential value from either an inline env var or a file env var.
// Having both set is a fatal configuration error.
func resolveCredential(inlineKey, fileKey, label string) (string, error) {
	inline := os.Getenv(inlineKey)
	filePath := os.Getenv(fileKey)
	if inline != "" && filePath != "" {
		return "", fmt.Errorf("both %s and %s are set for %s; use exactly one", inlineKey, fileKey, label)
	}
	if filePath != "" {
		return readSecret(filePath)
	}
	return inline, nil
}

func resolveCredentialFromMap(kv map[string]string, inlineKey, fileKey, label string) (string, error) {
	inline := kv[inlineKey]
	filePath := kv[fileKey]
	if inline != "" && filePath != "" {
		return "", fmt.Errorf("both %s and %s are set for %s; use exactly one", inlineKey, fileKey, label)
	}
	if filePath != "" {
		return readSecret(filePath)
	}
	return inline, nil
}

// buildInstance normalizes the URL and resolves the name, returning a validated Instance.
func buildInstance(rawURL, name, username, password string, skipTLS bool) (Instance, error) {
	rawURL = strings.TrimRight(rawURL, "/")
	u, err := url.Parse(rawURL)
	if err != nil {
		return Instance{}, fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}
	u.Scheme = strings.ToLower(u.Scheme)
	if u.Scheme != "http" && u.Scheme != "https" {
		return Instance{}, fmt.Errorf("URL scheme must be http or https, got %q in %q", u.Scheme, rawURL)
	}
	if name == "" {
		name = u.Host
	}
	return Instance{
		Name:     name,
		URL:      u.String(),
		Username: username,
		Password: password,
		SkipTLS:  skipTLS,
	}, nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}
