package utils

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/edsrzf/mmap-go"
	"golang.org/x/sys/unix"
)

var Blacklist = []string{
	".gov",
	".edu",
	".json",
	".[0-9.]+$",
	"github.com",
}

func In(target string, str_array []string) bool {
	// 判断字符串是否 存在于字符串数组内
	sort.Strings(str_array)
	index := sort.SearchStrings(str_array, target)
	if index < len(str_array) && str_array[index] == target {
		return true
	}
	return false
}

func DomainMatch(url string, blacklist []string) []string {
	/*
		提取域名
	*/
	if blacklist == nil {
		blacklist = Blacklist
	}

	// 黑名单正则
	var black_pattern []string
	for _, black := range blacklist {
		black_pattern = append(black_pattern, fmt.Sprintf(".*%s", black))
	}

	// 特殊过滤
	// black_pattern = append(black_pattern, filterlist...)
	pattern := fmt.Sprintf(`(?!%s)(https?:\/\/)?[a-zA-Z0-9*][-a-zA-Z0-9]{0,62}(\.[a-zA-Z0-9][-a-zA-Z0-9\/?=&\*]{0,80})+`, strings.Join(black_pattern, "|"))

	domain_rege := regexp2.MustCompile(pattern, 0)
	// domain_rege := regexp.MustCompile(`^(?!.*gov|.*edu)[a-zA-Z0-9][-a-zA-Z0-9]{0,62}(\.[a-zA-Z0-9][-a-zA-Z0-9]{0,62})+`)

	// return dedupe_from_list(domain_rege.FindAllString(url, -1))
	return DedupeFromList(Regexp2FindAllString(domain_rege, url))
}

func IsURL(path string) bool {
	// 常见的 URL 协议前缀
	protocols := []string{"http://", "https://"}

	for _, protocol := range protocols {
		if strings.HasPrefix(strings.ToLower(path), protocol) {
			return true
		}
	}
	return false
}

func IsFile(path string) (is_file bool) {
	// 使用os.Stat获取文件信息
	// fmt.Println(path)
	is_file = true
    
	_, err := os.Stat(path)
	if err != nil {
		// 如果路径不存在或有其他错误，则返回错误
		if os.IsNotExist(err) {
			is_file = false // 路径不存在，不是文件
		}
		is_file =  false // 其他错误
	}

	
	
	if !is_file{
		current_dir, _ := os.Getwd()

		_, err = os.Stat(filepath.Join(current_dir, path))
		if err != nil {
			// 如果路径不存在或有其他错误，则返回错误
			if os.IsNotExist(err) {
				is_file = false // 路径不存在，不是文件
			}
			is_file =  false // 其他错误
		}

	}

	return
}

func ReadFileToMap(filename string) map[string]bool {
	// 读取文件到 map

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}
	size := info.Size()

	if size == 0 {
		if _, err := file.WriteString("\n"); err != nil {
			log.Fatal(err)
		}
	}

	hash := make(map[string]bool)
	reader := bufio.NewReader(file)
	mm, err := mmap.Map(file, unix.PROT_READ, 0)
	if err != nil {
		log.Fatal(err)
	}
	defer mm.Unmap()

	for i := 0; int64(i) < size; i++ {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		hash[strings.Replace(line, "\n", "", -1)] = true
	}
	return hash
}

func ReadFileToList(filename string) (lines []string, err error) {

	re := regexp.MustCompile(`^~`)

	path := re.ReplaceAllString(filename, HomeDir())

	// 尝试打开文件
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close() // 确保在函数结束时关闭文件


	// 创建bufio.Scanner，用于逐行读取文件
	scanner := bufio.NewScanner(file)

	// 使用Scan方法逐行读取
	for scanner.Scan() {
		// 将当前行添加到切片中
		lines = append(lines, scanner.Text())
	}
	err = scanner.Err();


	return

}

func SaveTargetsToFile(filename string, target string) {

	// 保存目标到文件内

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	writer.WriteString(target + "\n")
	
	writer.Flush()
	file.Sync()

}

func HomeDir() string {
	// 获取 $home 路径
	usr, err := user.Current()
	if err != nil {
		fmt.Println("Could not get user home directory:", err)
	}
	return usr.HomeDir
}

func DedupeFromList(source []string) []string {
	// 列表去重
	var new_list []string

	dedupe_set := make(map[string]bool)
	for _, v := range source {
		dedupe_set[v] = true
	}

	for k := range dedupe_set {

		new_list = append(new_list, k)
	}

	return new_list
}

func Regexp2FindAllString(re *regexp2.Regexp, s string) []string {
	// 正则匹配提取
	var matches []string
	m, _ := re.FindStringMatch(s)
	for m != nil {
		matches = append(matches, m.String())
		m, _ = re.FindNextMatch(m)
	}
	return matches
}

func DomainValid(domain string) bool {

	// 域名正则表达式
	domainRegex := regexp.MustCompile(`(https?:\/\/)?(?:[a-zA-Z0-9*](?:(?:[a-zA-Z0-9]|-)*[a-zA-Z0-9])?\.)+(?:[a-zA-Z]{2,})([/\w?&\.=\-]+)?`)

	return domainRegex.MatchString(domain)
}

func ScopeHandleClassifier(name string) (hc HandleClassifier){

	hackerone_re := regexp.MustCompile(`https://hackerone\.com/([^/?]+)`)

	bugcrowd_re := regexp.MustCompile(`https://bugcrowd\.com/engagements/([^/?]+)`)

	intigriti_re := regexp.MustCompile(`https://app.intigriti.com/researcher/programs/([^/?]+)`)


	if matches := hackerone_re.FindStringSubmatch(name);len(matches)>1 {
		
		hc = HandleClassifier{Type: "hackerone",Handle: matches[1]}
	}else if matches := bugcrowd_re.FindStringSubmatch(name);len(matches)>1 {
		
		hc =  HandleClassifier{Type: "hackerone",Handle: matches[1]}
	}else if matches := intigriti_re.FindStringSubmatch(name);len(matches)>1 {

		hc =  HandleClassifier{Type: "intigriti",Handle: matches[1]}
	}

	return 
}