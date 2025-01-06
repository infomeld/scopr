package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/infomeld/scopr/pkg/assetparser"
	"github.com/infomeld/scopr/pkg/engine/bugcrowd"
	"github.com/infomeld/scopr/pkg/engine/hackerone"
	"github.com/infomeld/scopr/pkg/engine/intigriti"
	"github.com/infomeld/scopr/pkg/notify"
	"github.com/infomeld/scopr/pkg/proxypool"
	"github.com/infomeld/scopr/pkg/utils"
)

var source_path = filepath.Join(utils.HomeDir(), ".config/scopr/")

type Task struct {
	Name    string
	Timeout time.Duration
	fn      func()
}

func NewTask(name string, timeout time.Duration, fn func()) *Task {
	return &Task{
		Name:    name,
		Timeout: timeout,
		fn:      fn,
	}
}

func (t *Task) Run() {
	ticker := time.NewTicker(t.Timeout)
	defer ticker.Stop()


	for {
		select {
		case <-ticker.C:
			t.fn()
		}
	}
}

type (
	Scopr struct {
		BugcrowdScp  bugcrowd.BugcrowdScp
		HackeroneScp hackerone.HackeroneScp
		IntigritiScp intigriti.IntigritiScp
		DingTalk     utils.DingTalk
		Config       utils.Config
		Pool         proxypool.Pool
	}
)

func NewScopr(source_path string) *Scopr {

	/*
		https://hackerone.com/directory/programs
		https://bugcrowd.com/programs
		https://www.intigriti.com/programs
	*/
	
	
	config := utils.GetConfig(source_path)
	
	proxy_pool := proxypool.NewProxyPool(config.EnableProxy).InitPool()
	

	return &Scopr{
		HackeroneScp: *hackerone.NewHackeroneScp(config.HackerOne, proxy_pool),
		BugcrowdScp:  *bugcrowd.NewBugcrowdScp(config.Bugcrowd, proxy_pool),
		IntigritiScp: *intigriti.NewIntigritiScp(config.Intigriti, proxy_pool),
		// DingTalk:     config.DingTalk,
		Config:       config,
		Pool:         proxy_pool,
	}
}

func (s Scopr) bugcrowd(new_bugcrowd chan utils.NewScope){

	s.BugcrowdScp.Program()
}

func (s Scopr) hackerone(new_hackerone chan utils.NewScope, handle string ){

	/*
		hackerone 
	*/

	if handle == "" {
		// 判断是否指定 项目名称，如果没有则获取所有
		s.HackeroneScp.ProgramsScope(new_hackerone)
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)

	s.HackeroneScp.GetScope(handle, new_hackerone, &wg)

	go func() {
		wg.Wait()
	}()

	
}

func (s Scopr) intigriti(new_intigriti chan utils.NewScope){

	s.IntigritiScp.Program()
}

func (s Scopr) DomainMatch(url string) []string {
	return utils.DomainMatch(url, s.Config.Blacklist)
}

func (s Scopr) start(target string , new_scope_chan chan utils.NewScope ){

	if utils.IsURL(target){

		handle_classifier := utils.ScopeHandleClassifier(target)
			
		if s.Config.HackerOne.Enable && handle_classifier.Type =="hackerone" {
			s.hackerone(new_scope_chan, handle_classifier.Handle)
		}
		if s.Config.Bugcrowd.Enable  && handle_classifier.Type =="bugcrowd" {
			s.bugcrowd(new_scope_chan)
		}
		if s.Config.Intigriti.Enable  && handle_classifier.Type =="intigriti"{
			s.intigriti(new_scope_chan)
		}

	}else{

		target_list, err  := utils.ReadFileToList(target)
		// 判断是否是列表文件
		if err != nil{
			
			if s.Config.HackerOne.Enable {
				s.hackerone(new_scope_chan, target)
			}
			if s.Config.Bugcrowd.Enable {
				s.bugcrowd(new_scope_chan)
			}
			if s.Config.Intigriti.Enable {
				s.intigriti(new_scope_chan)
			}
			
		}else{

			for _, item := range target_list {
				// 读取文件列表
				handle_classifier := utils.ScopeHandleClassifier(item)
				
				if s.Config.HackerOne.Enable && handle_classifier.Type =="hackerone" {
					s.hackerone(new_scope_chan, handle_classifier.Handle)
				}
				if s.Config.Bugcrowd.Enable  && handle_classifier.Type =="bugcrowd" {
					s.bugcrowd(new_scope_chan)
				}
				if s.Config.Intigriti.Enable  && handle_classifier.Type =="intigriti"{
					s.intigriti(new_scope_chan)
				}
			}

		}

	}

}

func main() {

	var banner = `              
	 _____                 
	|   __|___ ___ ___ ___ 
	|__   |  _| . | . |  _|
	|_____|___|___| __|_|  
	              |_|     v1.0

	Keep track of bounty targets
    `

	var cycle_time int64
	var silent bool
	var scope_target string
	var show_app bool
	var show_all bool
	var only_new bool

	flag.Int64Var(&cycle_time, "t", 0, "监控周期(分钟)")
	flag.BoolVar(&silent, "silent", false, "是否静默状态")
	flag.StringVar(&scope_target, "s", "", "指定项目名/获取项目列表名称")
	flag.BoolVar(&show_app, "app", false, "显示app")
	flag.BoolVar(&show_all, "all", false, "显示所有")
	flag.BoolVar(&only_new, "new", false, "是否只输出新增资产")



	// 解析命令行参数写入注册的flag里
	flag.Parse()

	if !silent {
		fmt.Println(string(banner))
		fmt.Println("[*] Starting tracker", "... ")
	}

	os.MkdirAll(source_path, os.ModePerm)

	// 启动定时任务
	if cycle_time > 0 {

		tasks := []*Task{
			NewTask("tracker", time.Duration(cycle_time)*time.Minute, func() {
				run(silent, scope_target, only_new, show_app, show_all)
			}),
		}
		for _, task := range tasks {
			go task.Run()
		}
		// 等待任务结束
		select {}
	} else {
		run(silent, scope_target, only_new, show_app, show_all)
	}
}

func run(silent bool, scope_target string, only_new bool, show_app bool,show_all bool) {

	if !silent {
		now := time.Now().Format("2006-01-02 15:04:05")

		fmt.Println("[*] Date:", now)
	}
	// init config
	scopr := NewScopr(source_path)

	// 读取源目标文件
	// source_targets := utils.ReadFileToMap(filepath.Join(source_path, "domain.txt"))
	source_fail_targets := utils.ReadFileToMap(filepath.Join(source_path, "faildomain.txt"))
	source_bugbounty_url := utils.ReadFileToMap(filepath.Join(source_path, "bugbounty-public.txt"))
	private_bugbounty_url := utils.ReadFileToMap(filepath.Join(source_path, "bugbounty-private.txt"))

	// 获取新增赏金目标
	new_scope_chan := make(chan utils.NewScope)



	var outputWG sync.WaitGroup
	outputWG.Add(1)

	go func() {
		defer outputWG.Done()
		scopr.start(scope_target, new_scope_chan)
	}()

	go func() {
		outputWG.Wait()
		close(new_scope_chan)
	}()


	asset_parser := assetparser.MultiTypeParser{ShowAll:show_all,ShowApp: show_app}


	for scope := range new_scope_chan {



		if scope.NewAsset.Value != "" {
			
			if !only_new{

				target,_ := asset_parser.Parse(scope.NewAsset.Type, scope.NewAsset.Value)
				if target!= ""{
					fmt.Println(target)
				}

			}

			// if !source_targets[scope.NewAsset.Value]{

			// 	re := regexp.MustCompile(strings.Join(scopr.Config.Blacklist, "|"))

			// 	if !re.MatchString(scope.NewTarget){
			// 		if only_new{
			// 			fmt.Println(scope.NewTarget)
			// 		}
			// 		// 保存新增目标
			// 		utils.SaveTargetsToFile(filepath.Join(source_path, "domain.txt"), scope.NewTarget)
			// 	}
			// }
		}

		if scope.NewFailTarget!= "" && !source_fail_targets[scope.NewFailTarget] {

			utils.SaveTargetsToFile(filepath.Join(source_path, "faildomain.txt"), scope.NewFailTarget)
		}

		if  scope.NewPublicURL != "" && !source_bugbounty_url[scope.NewPublicURL]{

			utils.SaveTargetsToFile(filepath.Join(source_path, "bugbounty-public.txt"), scope.NewPublicURL)

		}

		if scope.NewPrivateURL!= "" && !private_bugbounty_url[scope.NewPrivateURL]{

			utils.SaveTargetsToFile(filepath.Join(source_path, "bugbounty-private.txt"), scope.NewPrivateURL)
		}

	}


	// // 发送通知信息
	// var msg_content = notify.BountyContent{
	// 	Hackerone: notify.MessageContent{
	// 		Urls:    new_hackerone.new_bounty_url,
	// 		Targets: new_hackerone.new_targets,
	// 		App:     new_hackerone.new_app,
	// 	},
	// 	Bugcrowd: notify.MessageContent{
	// 		Urls:    new_bugcrowd.new_bounty_url,
	// 		Targets: new_bugcrowd.new_targets,
	// 		App:     new_bugcrowd.new_app,
	// 	},
	// 	Intigriti: notify.MessageContent{
	// 		Urls:    new_intigriti.new_bounty_url,
	// 		Targets: new_intigriti.new_targets,
	// 		App:     new_intigriti.new_app,
	// 	},
	// }

	// bountry.SendDingtalk(msg_content)

}

func (scopr Scopr) SendDingtalk(content notify.BountyContent) {

	var msg_content = notify.TargetMarkdown("Hackerone", content.Hackerone) +
		notify.TargetMarkdown("Bugcrowd", content.Bugcrowd) +
		notify.TargetMarkdown("Intigriti", content.Intigriti)

	if msg_content == "" {
		return
	}

	var receiver notify.Robot
	receiver.AppKey = scopr.DingTalk.AppKey
	receiver.AppSecret = scopr.DingTalk.AppSecret
	webhookurl := receiver.Signature()
	params := receiver.SendMarkdown("Bountytr 资产监控", msg_content, []string{}, []string{}, false)

	notify.SendRequest(webhookurl, params)
}
