package assetparser

import (
	"fmt"
	"regexp"
	"strings"
)

// MultiTypeParser 用于解析多种类型的数据
type MultiTypeParser struct {
    // 可以添加字段
    ShowAll bool
    ShowApp bool
}

// Parse 解析数据
func (p *MultiTypeParser) Parse(dataType string, data string) (string, error) {
    switch dataType {
    case "DOMAIN":
        return p.parseDomain(data)
    case "URL":
        return p.parseUrl(data)
    case "WILDCARD":
        return p.parseWildcard(data)
    case "API":
        return p.parseApi(data)
    case "OTHER":
        return p.parseOther(data)
    case "GOOGLE_PLAY_APP_ID":
        return p.parseGoogleApp(data)
	case "APPLE_STORE_APP_ID":
        return p.parseAppleApp(data)
    default:
        return "", fmt.Errorf("unsupported data type: %s", dataType)
    }
}

// 解析 DOMAIN
func (p *MultiTypeParser) parseDomain(data string)(result string ,err error) {

    if p.ShowApp && !p.ShowAll{
        return 
    }

    domain_list := []string{}

	data = p.cleanDomain(data)
	domainsSlice := p.domainSplitTrimSpace(data)
	for _, domain := range domainsSlice {
		
		domain_list = append(domain_list, domain)
	
	}
    result = strings.Join(domain_list, "\n")
    
    return 
}

// 解析 URL
func (p *MultiTypeParser) parseUrl(data string) (string, error) {
    return p.parseDomain(data)
}

// 解析 WILDCARD
func (p *MultiTypeParser) parseWildcard(data string) (string, error) {
	return p.parseDomain(data)
}

// 解析 API
func (p *MultiTypeParser) parseApi(data string)(string, error) {
    return p.parseDomain(data)
}

// 解析 OTHER
func (p *MultiTypeParser) parseOther(data string)(string, error) {
    if p.ShowAll{
        return data, nil
    }else if strings.HasPrefix(data, "*")|| strings.HasSuffix(data, "*"){
		return p.parseDomain(data)
	}
    return "", nil
	
}

// 解析 APP
func (p *MultiTypeParser) parseGoogleApp(data string) (string, error) {
    if p.ShowApp || p.ShowAll{
        return "https://play.google.com/store/apps/details?id=" + data, nil 
    }
    return "", nil
   
}

// 解析 APP
func (p *MultiTypeParser) parseAppleApp(data string) (string, error) {

    if p.ShowApp || p.ShowAll{
        return "https://apps.apple.com/us/app/careem-captain/id" + data, nil
    }
    return "", nil
    
}


func (p *MultiTypeParser) cleanDomain(domain string) string {

	pattern := `[\w]+[\w\-_~\.]+\.[a-zA-Z]+|$`
	r, err := regexp.Compile(pattern)
	if err != nil {
		return domain
	}

	cDomain := r.FindString(domain)
	if cDomain != "" {
		return cDomain
	}
	return domain
}

func  (p *MultiTypeParser) domainSplitTrimSpace(domain string) []string {
	domainSlice := strings.Split(domain, ",")
	for i := range domainSlice {
		domainSlice[i] = strings.TrimSpace(domainSlice[i])
	}

	return domainSlice
}


func main() {
    parser := &MultiTypeParser{}

    result, err := parser.Parse("DOMAIN", "example.com")
    if err != nil {
        fmt.Println("Error:", err)
    } else {
        fmt.Println(result)
    }
}