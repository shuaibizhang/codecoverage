package apollo

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/shima-park/agollo"
	"gopkg.in/yaml.v3"
)

type ApolloCli interface {
	GetString(key string, defaultValue string) string
	GetInt(key string, defaultValue int) int
	GetBool(key string, defaultValue bool) bool
	UnmarshalJSON(key string, target interface{}) error
	UnmarshalYAML(key string, target interface{}) error
}

type apolloClient struct {
	ag         agollo.Agollo
	ns         string
	backupData map[string]string
}

func NewApolloClient(cfg ApolloConfig) (ApolloCli, error) {
	cluster := cfg.Cluster
	if cluster == "" {
		cluster = "default"
	}

	// 预加载可能的 Namespace 后缀
	namespaces := []string{cfg.Namespace, "application"}
	if !hasExtension(cfg.Namespace) {
		namespaces = append(namespaces, cfg.Namespace+".yaml", cfg.Namespace+".yml", cfg.Namespace+".json")
	}

	options := []agollo.Option{
		agollo.Cluster(cluster),
		agollo.PreloadNamespaces(namespaces...),
	}
	if cfg.Secret != "" {
		options = append(options, agollo.AccessKey(cfg.Secret))
	}

	ag, err := agollo.New(cfg.Addr, cfg.AppID, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create apollo client: %w", err)
	}

	// 等待异步拉取，如果 2 秒内库没拿到数据，则启动手动同步 (适应 Traefik 路径前缀)
	log.Printf("[Apollo] Initializing client for AppID: %s, Addr: %s", cfg.AppID, cfg.Addr)
	
	initialized := false
	for i := 0; i < 2; i++ {
		for _, ns := range namespaces {
			if len(ag.GetNameSpace(ns)) > 0 {
				if ns == cfg.Namespace || strings.HasPrefix(ns, cfg.Namespace+".") {
					initialized = true
					break
				}
			}
		}
		if initialized {
			break
		}
		time.Sleep(1 * time.Second)
	}

	var backupData map[string]string
	if !initialized {
		log.Printf("[Apollo] Library sync timeout, starting manual sync fallback...")
		// 尝试所有可能的 Namespace 组合
		for _, ns := range namespaces {
			if data, err := fetchConfigDirectly(cfg, cluster, ns); err == nil && len(data) > 0 {
				log.Printf("[Apollo] Manual sync success for namespace: %s", ns)
				backupData = data
				initialized = true
				break
			}
		}
	}

	if !initialized {
		log.Printf("[Apollo] Warning: failed to fetch config from all possible namespaces")
	}

	return &apolloClient{
		ag:         ag,
		ns:         cfg.Namespace,
		backupData: backupData,
	}, nil
}

func hasExtension(ns string) bool {
	return len(ns) > 5 && (ns[len(ns)-5:] == ".yaml" || ns[len(ns)-4:] == ".yml" || ns[len(ns)-5:] == ".json")
}

func (a *apolloClient) GetString(key string, defaultValue string) string {
	if a.backupData != nil {
		if val, ok := a.backupData[key]; ok {
			return val
		}
	}
	val := a.ag.Get(key, agollo.WithNamespace(a.ns))
	if val == "" {
		return defaultValue
	}
	return val
}

func (a *apolloClient) GetInt(key string, defaultValue int) int {
	valStr := ""
	if a.backupData != nil {
		if v, ok := a.backupData[key]; ok {
			valStr = v
		}
	}
	if valStr == "" {
		valStr = a.ag.Get(key, agollo.WithNamespace(a.ns))
	}
	if valStr == "" {
		return defaultValue
	}
	val, _ := strconv.Atoi(valStr)
	return val
}

func (a *apolloClient) GetBool(key string, defaultValue bool) bool {
	valStr := ""
	if a.backupData != nil {
		if v, ok := a.backupData[key]; ok {
			valStr = v
		}
	}
	if valStr == "" {
		valStr = a.ag.Get(key, agollo.WithNamespace(a.ns))
	}
	if valStr == "" {
		return defaultValue
	}
	val, _ := strconv.ParseBool(valStr)
	return val
}

func (a *apolloClient) UnmarshalJSON(key string, target interface{}) error {
	valStr := ""
	if a.backupData != nil {
		if v, ok := a.backupData[key]; ok {
			valStr = v
		}
	}
	if valStr == "" {
		valStr = a.ag.Get(key, agollo.WithNamespace(a.ns))
	}
	if valStr == "" {
		return fmt.Errorf("config key %s not found", key)
	}
	return json.Unmarshal([]byte(valStr), target)
}

func (a *apolloClient) UnmarshalYAML(key string, target interface{}) error {
	var content string
	var found bool

	// 1. 优先从备份数据中获取
	if a.backupData != nil {
		content, found = a.backupData["content"]
		if !found {
			content, found = a.backupData[key]
		}
	}

	// 2. 如果备份没有，从库中尝试所有可能的后缀
	if !found {
		nsList := []string{a.ns}
		if !hasExtension(a.ns) {
			nsList = append(nsList, a.ns+".yaml", a.ns+".yml")
		}
		for _, ns := range nsList {
			content = a.ag.Get("content", agollo.WithNamespace(ns))
			if content == "" {
				content = a.ag.Get(key, agollo.WithNamespace(ns))
			}
			if content != "" {
				found = true
				break
			}
		}
	}

	if !found {
		return fmt.Errorf("no config found for key '%s' or 'content'", key)
	}

	// 智能 YAML 拆解：如果是整个 YAML 内容，尝试提取 sub-section
	var fullMap map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &fullMap); err == nil {
		if subSection, exists := fullMap[key]; exists {
			subBytes, _ := yaml.Marshal(subSection)
			return yaml.Unmarshal(subBytes, target)
		}
	}

	return yaml.Unmarshal([]byte(content), target)
}

func fetchConfigDirectly(cfg ApolloConfig, cluster string, ns string) (map[string]string, error) {
	testUrl := fmt.Sprintf("%s/configs/%s/%s/%s", cfg.Addr, cfg.AppID, cluster, ns)
	
	// 尝试带签名请求，失败则降级为匿名请求
	if cfg.Secret != "" {
		if data, err := doRequest(testUrl, cfg.AppID, cfg.Secret, cluster, ns); err == nil {
			return data, nil
		}
	}
	return doRequest(testUrl, cfg.AppID, "", cluster, ns)
}

func doRequest(urlStr, appID, secret, cluster, ns string) (map[string]string, error) {
	req, _ := http.NewRequest("GET", urlStr, nil)
	if secret != "" {
		timestamp := fmt.Sprintf("%d", time.Now().UnixNano()/1e6)
		uri := fmt.Sprintf("/configs/%s/%s/%s", appID, cluster, ns)
		stringToSign := timestamp + "\n" + uri
		mac := hmac.New(sha1.New, []byte(secret))
		mac.Write([]byte(stringToSign))
		signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		req.Header.Set("Authorization", fmt.Sprintf("Apollo %s:%s", appID, signature))
		req.Header.Set("Timestamp", timestamp)
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	var result struct {
		Configurations map[string]string `json:"configurations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Configurations, nil
}
