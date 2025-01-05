package intigriti

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/infomeld/scopr/pkg/proxypool"
	"github.com/infomeld/scopr/pkg/utils"
	"github.com/tidwall/gjson"
	"golang.org/x/net/html"
)

type IntigritiScp struct {
	Url         string             `json:"url"`
	Programs    []Intigriti `json:"programs"`
	Config      utils.Intigriti    `json:"config"`
	Pool        proxypool.Pool     `json:"pool"`
}

func NewIntigritiScp(config  utils.Intigriti , pool proxypool.Pool) *IntigritiScp {

	return &IntigritiScp{
		Programs:    []Intigriti{},
		Config: config,
		Pool:        pool,
	}
}

func (i IntigritiScp) ProgramRquest(target_url string) (body []byte, err error) {

	proxyUrl, _ := url.Parse(i.Pool.RandProxy())

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyUrl),
	}

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}

	req, err := http.NewRequest("GET", target_url, nil)
	if err != nil {
		return
	}

	// 请求JSON数据
	resp, err := client.Do(req)
	if err != nil {
		return
	}

	// 读取响应体
	body, err = ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		err = fmt.Errorf(resp.Status)
	}

	return
}

func (i IntigritiScp) FindByClass(n *html.Node, className string) (elements []*html.Node) {
	if n.Type == html.ElementNode && n.Data == "div" {
		for _, attr := range n.Attr {
			if attr.Key == "class" && attr.Val == className {
				elements = append(elements, n)
				return
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		elements = append(elements, i.FindByClass(c, className)...)
	}

	return
}

func (i IntigritiScp) GetText(n *html.Node) (content string) {
	if n.Type == html.TextNode {
		content = strings.TrimSpace(n.Data)
		return
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		content = i.GetText(c)
		if content != "" {
			break
		}
	}
	return
}

func (i IntigritiScp) BuildId() (tag string, err error) {

	// 请求JSON数据
	resp, err := http.Get("https://www.intigriti.com/program")
	if err != nil {
		return
	}

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	re := regexp.MustCompile(`/_next/static/([^/]+)/_buildManifest\.js`)
	match := re.FindStringSubmatch(string(body))

	if len(match) > 1 {
		tag = match[1]
	}

	return

}

func (i IntigritiScp) Program() (programs []Intigriti) {

	var new_program []Intigriti
	new_intigriti_program := make(chan Intigriti) // 创建缓冲通道
	semaphore := make(chan struct{}, i.Config.Concurrency)      //控制并发数

	tag, err := i.BuildId()
	if err != nil {
		fmt.Println("intigriti 获取 BuildId 失败", err)
	}

	url := fmt.Sprintf("https://www.intigriti.com/_next/data/%s/en/programs.json", tag)

	res_data, err := i.ProgramRquest(url)
	if err != nil {
		fmt.Println("intigriti 获取 programs 失败", err)
		return
	}

	result := gjson.GetBytes(res_data, "pageProps.programs")

	json.Unmarshal([]byte(result.Raw), &new_program)

	var wg sync.WaitGroup

	for _, item := range new_program {
		wg.Add(1)

		if item.ConfidentialityLevel == 4 {
			go i.Scope(item, new_intigriti_program, semaphore, &wg)
		} else {
			wg.Done()
		}
	}

	// 从缓冲通道读取数据
	for {
		select {
		case program := <-new_intigriti_program:
			programs = append(programs, program)
		case <-time.After(3 * time.Second):
			wg.Wait()
			return
		}
	}

}

func (i IntigritiScp) Scope(intigriti Intigriti, new_intigriti_program chan Intigriti, semaphore chan struct{}, wg *sync.WaitGroup) (in_scopes []IntigritiScope, out_scopes []IntigritiScope) {
	/*
		获取项目赏金目标
	*/
	defer wg.Done()

	semaphore <- struct{}{}

	url := fmt.Sprintf("https://app.intigriti.com/programs/%s/%s/detail", intigriti.Handle, intigriti.Handle)

	res_data, err := i.ProgramRquest(url)
	if err != nil {
		fmt.Println("intigriti 获取 target 失败", err)
		<-semaphore

		new_intigriti_program <- intigriti
		return
	}

	doc, _ := html.Parse(strings.NewReader(string(res_data)))

	container := i.FindByClass(doc, "domain-container")

	for _, item := range container {
		domain_endpoint := i.FindByClass(item, "domainEndpoint")
		domain_type := i.FindByClass(item, "domainType")
		impact_type := i.FindByClass(item, "impact")

		new_scope := IntigritiScope{
			Endpoint: i.GetText(domain_endpoint[0]),
			Impact:   i.GetText(impact_type[0]),
			Type:     i.GetText(domain_type[0]),
		}

		if strings.Contains(new_scope.Impact, "Out") {
			out_scopes = append(out_scopes, new_scope)
		} else {
			in_scopes = append(in_scopes, new_scope)
		}

	}

	intigriti.Targets.InScope = in_scopes
	intigriti.Targets.OutOfScope = out_scopes

	<-semaphore

	new_intigriti_program <- intigriti

	// fmt.Printf("【%s】\n", intigriti.Handle)

	return

}
