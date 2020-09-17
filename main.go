package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"z_tools/pkg"
	"z_tools/pkg/qiniu"
)

var (
	ServerHost = "http://z.jiaoliuqu.com"
	UserToken  string // 用户token
	env        string // 环境
	version    = "0.1.12"
)

var (
	repoObjPath   = "./.repo/objects/"
	tokenPath     = "./.repo/token"
	envPath       = "./.repo/env"
	updatePath    = "./.repo/updateTime"
	indexPath     = "./.repo/index"
	kIndexPath    = "./.repo/kindex"
	imgPath       = "./img/"
	workPostsPath = "./posts/"
	knWorkPath    = "./knowledge/"
)

const (
	StatusUserDel = "-2" // 用户删除
	StatusAdmDel  = "-3" // 管理员删除
)

// PostDesc 文章描述
type PostDesc struct {
	FileName   string `json:"file_name"`   // 文件名称
	UpdateTime string `json:"update_time"` // 更新时间
	Md5        string `json:"file_md5"`    // 文件MD5
	Status     string `json:"status"`      // 文件状态 -2:自己删除 -3:管理员删除 其他状态这边暂时用不到
}

// KnowledgeDesc 知识点描述
type KnowledgeDesc struct {
	KName      string `json:"kName"`       // 知识点名称
	UpdateTime string `json:"update_time"` // 更新时间
	Md5        string `json:"file_md5"`    // 文件MD5
	Changelog  string `json:"changelog"`   // 修改日志
}

func init() {
	err := os.MkdirAll(workPostsPath, os.ModePerm)
	if err != nil {
		log.Printf("创建工作区目录异常:%s", err.Error())
		return
	}
	err = os.MkdirAll(knWorkPath, os.ModePerm)
	if err != nil {
		log.Printf("创建知识点工作区目录异常:%s", err.Error())
		return
	}
	err = os.MkdirAll(imgPath, os.ModePerm)
	if err != nil {
		log.Printf("创建img目录异常:%s", err.Error())
		return
	}
	err = os.MkdirAll(repoObjPath, os.ModePerm)
	if err != nil {
		log.Printf("创建repo/objects目录异常:%s", err.Error())
		return
	}
}

func main() {
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"V"},
		Usage:   "print only the version",
	}

	app := &cli.App{
		Version: version,
		Usage:   "文章上传助手",
		Commands: []*cli.Command{
			{
				Name:        "init",
				Usage:       "初始化环境",
				Description: "1. doc init test test为用户的token",
				ArgsUsage:   "[token]",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入token,命令行格式./doc init 用户token")
						return nil
					}
					env := "online"
					if c.NArg() >= 2 {
						env = c.Args().Get(1)
					}
					InitDoc(c.Args().Get(0), env)
					return nil
				},
			},
			{
				Name:        "new",
				Usage:       "新建文章",
				Description: "1. doc new test 本地自动生成一篇test.md的空文档",
				ArgsUsage:   "[文件名]",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入文件名,命令行格式./doc new xx")
						return nil
					}
					NewDoc(c.Args().Get(0))
					return nil
				},
			},
			{
				Name:        "add",
				Usage:       "提交到本地仓库",
				Description: "1. doc add test.md 提交test.md到本地仓库\n\r   2. doc add . 提交工作区的全部文件到本地仓库",
				ArgsUsage:   "[文件名]",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入文件名,命令行格式./doc add xx.md")
						return nil
					}
					Add(c.Args().Get(0))
					return nil
				},
			},
			{
				Name:        "pull",
				Usage:       "拉取文章列表",
				Description: "1. doc pull 从服务器拉取最新文章列表到本地",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					Pull()
					return nil
				},
			},
			{
				Name:        "push",
				Usage:       "提交到服务器",
				Description: "1. doc push 把本地仓库变更提交到服务器",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					Push()
					return nil
				},
			},
			{
				Name:        "rm",
				Usage:       "删除文件",
				Description: "1. doc rm test.md 把test.md从本地仓库移除",
				ArgsUsage:   "[文件名]",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入文件名,命令行格式./doc rm xx.md")
						return nil
					}
					Rm(c.Args().Get(0))
					return nil
				},
			},
			{
				Name:        "status",
				Usage:       "查看文件变更",
				Description: "1. doc status 比对本地仓库和工作区的文件变更",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					Status()
					return nil
				},
			},
			{
				Name:        "checkout",
				Usage:       "恢复本地仓库的指定文件到工作区",
				Description: "1. doc checkout test.md 从本地仓库恢复test.md到工作区\n\r   2. doc checkout . 恢复本地仓库的全部文件到工作区",
				ArgsUsage:   "[文件名]",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入文件名,命令行格式./doc checkout xx.md 支持点号")
						return nil
					}
					Checkout(c.Args().Get(0))
					return nil
				},
			},
			{
				Name:        "update",
				Usage:       "版本升级",
				Description: "1. doc update 升级程序版本",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					Update(false)
					return nil
				},
			},
			{
				Name:        "updateToSer",
				Usage:       "版本更新到服务器-[需管理员权限]",
				Description: "1. doc updateToSer 0.0.3 将0.0.3版本程序上传到七牛",
				ArgsUsage:   "[版本号]",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入版本号")
						return nil
					}
					Update2Ser(c.Args().Get(0))
					return nil
				},
			},
			{
				Name:        "updateInstall",
				Usage:       "更新install文件-[需管理员权限]",
				Description: "1. doc updateInstall",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					UpdateInstallShell()
					return nil
				},
			},
			{
				Name:        "kpull",
				Usage:       "拉取知识点",
				Description: "1. doc kpull xx 拉取xx的知识点",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入知识点")
						return nil
					}
					kPull(c.Args().Get(0))
					return nil
				},
			},
			{
				Name:        "kadd",
				Usage:       "提交知识点更新到本地",
				Description: "1. doc kadd xx 要提交的知识点",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入知识点")
						return nil
					}

					if c.NArg() < 2 {
						log.Printf("请输入修改日志")
						return nil
					}
					kAdd(c.Args().Get(0), c.Args().Get(1))
					return nil
				},
			},
			{
				Name:        "kpush",
				Usage:       "提交知识点更新到服务器",
				Description: "1. doc kpush xx",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入知识点")
						return nil
					}
					kPush(c.Args().Get(0))
					return nil
				},
			},
			{
				Name:        "knew",
				Usage:       "新建知识点",
				Description: "1. doc knew xx 要新建的知识点",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入知识点")
						return nil
					}
					kNew(c.Args().Get(0))
					return nil
				},
			},
			{
				Name:        " krel",
				Usage:       "给知识点创建别名",
				Description: "1. doc  krel xx 知识点 别名",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入知识点")
						return nil
					}
					if c.NArg() < 2 {
						log.Printf("请输入别名")
						return nil
					}
					krel(c.Args().Get(0), c.Args().Get(1))
					return nil
				},
			},
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	env = ReadEnv()
	if env == "test" {
		ServerHost = "http://10.10.80.222:8000/2016-08-15/proxy"
	}
	UserToken = ReadToken()

	// 用户token校验
	if len(os.Args) >= 2 && os.Args[1] != "init" && UserToken == "" {
		log.Printf("用户token为空,请到小程序我的TAB页复制,并执行./doc init 用户token 进行初始化~")
		return
	}

	autoUpdate()

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

// autoUpdate 自动升级
func autoUpdate() {
	lastUpdate, _ := strconv.Atoi(ReadUpdateTime())
	if int(time.Now().Unix())-lastUpdate < 86400 {
		return
	}

	WriteUpdateTime()
	Update(true)
}

// ReadIndex 读取索引
func ReadIndex() (map[string]*PostDesc, error) {
	m := make(map[string]*PostDesc)

	b, _ := ioutil.ReadFile(indexPath)
	if len(b) == 0 {
		return m, nil
	}

	var list []*PostDesc
	err := json.Unmarshal(b, &list)
	if err != nil {
		return nil, err
	}

	for _, v := range list {
		m[v.FileName] = v
	}
	return m, nil
}

// WriteIndex 写入索引
func WriteIndex(m map[string]*PostDesc) error {
	var list []*PostDesc
	for _, v := range m {
		list = append(list, v)
	}
	if len(list) == 0 {
		return nil
	}

	b, err := json.Marshal(list)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(indexPath, b, 0644)
	if err != nil {
		return err
	}
	return nil
}

// ReadKIndex 读取知识点索引
func ReadKIndex() (map[string]*KnowledgeDesc, error) {
	m := make(map[string]*KnowledgeDesc)

	b, _ := ioutil.ReadFile(kIndexPath)
	if len(b) == 0 {
		return m, nil
	}

	var list []*KnowledgeDesc
	err := json.Unmarshal(b, &list)
	if err != nil {
		return nil, err
	}

	for _, v := range list {
		m[v.KName] = v
	}
	return m, nil
}

// WriteKIndex 写入知识点索引
func WriteKIndex(m map[string]*KnowledgeDesc) error {
	var list []*KnowledgeDesc
	for _, v := range m {
		list = append(list, v)
	}
	if len(list) == 0 {
		return nil
	}

	b, err := json.Marshal(list)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(kIndexPath, b, 0644)
	if err != nil {
		return err
	}
	return nil
}

// InitDoc 初始化
func InitDoc(token string, env string) {
	err := ioutil.WriteFile(tokenPath, []byte(token), 0644)
	if err != nil {
		log.Printf("初始化token异常:%s", err.Error())
		return
	}
	err = ioutil.WriteFile(envPath, []byte(env), 0644)
	if err != nil {
		log.Printf("初始化env异常:%s", err.Error())
		return
	}
	WriteUpdateTime()
	log.Printf("初始化成功")
}

// ReadToken 读取用户token
func ReadToken() string {
	b, _ := ioutil.ReadFile(tokenPath)
	return string(b)
}

// ReadEnv 读取环境变量
func ReadEnv() string {
	b, _ := ioutil.ReadFile(envPath)
	return string(b)
}

// ReadUpdateTime 读取安装时间
func ReadUpdateTime() string {
	b, _ := ioutil.ReadFile(updatePath)
	return string(b)
}

// WriteUpdateTime 刷新安装时间
func WriteUpdateTime() {
	ioutil.WriteFile(updatePath, []byte(fmt.Sprintf("%d", time.Now().Unix())), 0644)
	return
}

// Pull 拉取远程
func Pull() {
	localRepoPosts, err := ReadIndex()
	if err != nil {
		log.Printf("读取本地仓库异常:%s", err.Error())
		return
	}

	data, err := pkg.ClientCall(fmt.Sprintf("%s/info/client?action=getList&token=%s", ServerHost, UserToken), url.Values{})
	if err != nil {
		log.Printf("拉取远程文章列表异常:%s", err.Error())
		return
	}

	remotePosts, _ := data.([]interface{})
	for _, v := range remotePosts {
		var remote PostDesc
		m, ok := v.(map[string]interface{})
		if !ok {
			log.Printf("拉取文章异常,返回字段格式不对:%v", v)
			continue
		}

		remote.FileName, _ = m["file_name"].(string)
		remote.Md5, _ = m["file_md5"].(string)
		remote.UpdateTime, _ = m["update_time"].(string)
		remote.Status, _ = m["status"].(string)

		if remote.FileName == "" || remote.Md5 == "" || remote.UpdateTime == "" {
			log.Printf("拉取文章异常,返回字段不全,file:%s,md5:%s,time:%s", remote.FileName, remote.Md5, remote.UpdateTime)
			continue
		}

		// 如果文件远程被删除,则本地也相应删除
		if remote.Status == StatusUserDel || remote.Status == StatusAdmDel {
			local, ok := localRepoPosts[remote.FileName]
			if ok && local.Status != StatusUserDel && local.Status != StatusAdmDel {
				os.Remove(repoObjPath + local.Md5)
				os.Remove(workPostsPath + local.FileName)
				log.Printf("文件远程被删除,删除本地文件:%s", remote.FileName)
			}
			localRepoPosts[remote.FileName] = &remote
			continue
		}

		// 更新本地repo
		local, ok := localRepoPosts[remote.FileName]
		if ok {
			if (local.Md5 == remote.Md5 && local.Status == remote.Status) || pkg.TimeCompare(local.UpdateTime, remote.UpdateTime) {
				continue
			}
		}

		localRepoPosts[remote.FileName] = &remote

		// 如果只是状态变更，文件没变更，则不做处理
		if ok && local.Md5 == remote.Md5 {
			continue
		}

		form := url.Values{"filename": {remote.FileName}}
		retData, err := pkg.ClientCall(fmt.Sprintf("%s/info/client?token=%s&action=get", ServerHost, UserToken), form)
		if err != nil {
			log.Printf("拉取文章详情异常:%s,文章:%s", err.Error(), remote.FileName)
			continue
		}

		data, ok := retData.(map[string]interface{})
		if !ok {
			log.Printf("拉取远程文章格式异常:%v,文章:%s", retData, remote.FileName)
			continue
		}

		content, _ := data["content"].(string)

		err = pkg.Write2File([]byte(content), repoObjPath+remote.Md5)
		if err != nil {
			log.Printf("写入文章异常:%s,文章:%s", err.Error(), remote.FileName)
			continue
		}

		_, err = pkg.CopyFile(workPostsPath+remote.FileName, repoObjPath+remote.Md5)
		if err != nil {
			log.Printf("拷贝文章异常:%s,文章:%s", err.Error(), remote.FileName)
			continue
		}

		if local != nil {
			os.Remove(repoObjPath + local.Md5)
		}

		log.Printf("拉取远程文章成功:%s", remote.FileName)
	}

	WriteIndex(localRepoPosts)
}

// Push 推到远程服务器
func Push() {
	localRepoPosts, err := ReadIndex()
	if err != nil {
		log.Printf("读取本地仓库异常:%s", err.Error())
		return
	}

	data, err := pkg.ClientCall(fmt.Sprintf("%s/info/client?action=getList&token=%s", ServerHost, UserToken), url.Values{})
	if err != nil {
		log.Printf("拉取远程文章列表异常:%s", err.Error())
		return
	}

	remotePosts := make(map[string]PostDesc)
	l, _ := data.([]interface{})
	for _, v := range l {
		m := v.(map[string]interface{})
		var p PostDesc
		p.FileName, _ = m["file_name"].(string)
		p.Md5, _ = m["file_md5"].(string)
		p.UpdateTime, _ = m["update_time"].(string)
		p.Status, _ = m["status"].(string)

		if p.FileName == "" || p.Md5 == "" || p.UpdateTime == "" {
			log.Printf("拉取文章异常,返回字段不全:file:%s,md5:%s,time:%s", p.FileName, p.Md5, p.UpdateTime)
			continue
		}

		remotePosts[p.FileName] = p

		// 如果远程文章被删除,则本地也一并删除
		if p.Status == StatusUserDel || p.Status == StatusAdmDel {
			local, ok := localRepoPosts[p.FileName]
			if ok && local.Status != StatusUserDel && local.Status != StatusAdmDel {
				os.Remove(repoObjPath + local.Md5)
				os.Remove(workPostsPath + local.FileName)
				log.Printf("文件远程被删除,删除本地文件:%s", p.FileName)
			}
			localRepoPosts[p.FileName] = &p
		}
	}
	WriteIndex(localRepoPosts)

	for _, v := range localRepoPosts {
		r, ok := remotePosts[v.FileName]
		if ok {
			if (r.Md5 == v.Md5 && r.Status == v.Status) || pkg.TimeCompare(r.UpdateTime, v.UpdateTime) {
				continue
			}

			// 删除远程文件
			if v.Status == StatusUserDel && r.Status != StatusUserDel {
				form := url.Values{"filename": {v.FileName}}
				url := fmt.Sprintf("%s/info/client?token=%s&action=delete", ServerHost, UserToken)
				_, err = pkg.ClientCall(url, form)
				if err != nil {
					log.Printf("删除远程文章异常:%s,文章:%s", err.Error(), v.FileName)
				} else {
					log.Printf("删除远程文章成功,文章:%s", v.FileName)
				}
				continue
			}
		}

		// 本地删除的情况跳过
		if v.Status == StatusUserDel {
			continue
		}

		b, err := ioutil.ReadFile(repoObjPath + v.Md5)
		if err != nil {
			log.Printf("读取文章异常:%s,文章:%s", err.Error(), v.FileName)
			continue
		}

		title, category, err := getMDTileCategory(repoObjPath + v.Md5)
		if err != nil {
			log.Printf("读取文章title和分类异常:%s,文章:%s", err.Error(), v.FileName)
			continue
		}

		content := string(b)
		form := url.Values{
			"filename":   {v.FileName},
			"token":      {UserToken},
			"md5":        {v.Md5},
			"content":    {content},
			"title":      {title},
			"category":   {category},
			"updateTime": {v.UpdateTime},
		}

		url := fmt.Sprintf("%s/info/client?token=%s&action=add", ServerHost, UserToken)
		_, err = pkg.ClientCall(url, form)
		if err != nil {
			log.Printf("文章推到远程异常:%s,文章:%s", err.Error(), v.FileName)
			continue
		}

		log.Printf("文章推到远程成功文章:%s", v.FileName)
	}
}

// getKNLocalVersion 获取知识点本地版本号
func getKNLocalVersion(knowledge string) string {
	versionPath := fmt.Sprintf("%s%s/version", knWorkPath, knowledge)
	b, _ := ioutil.ReadFile(versionPath)
	return string(b)
}

// kPull 远程拉取知识点
func kPull(knowledge string) {
	data, err := pkg.ClientCall(fmt.Sprintf("%s/info/client?action=kget&token=%s&kname=%s", ServerHost, UserToken, knowledge), url.Values{})
	if err != nil {
		log.Printf("拉取远程知识点异常:%s", err.Error())
		return
	}

	remoteKN, ok := data.(map[string]interface{})
	if !ok {
		log.Printf("拉取远程知识点异常:%v", remoteKN)
		return
	}
	list, ok := remoteKN["list"].([]interface{})
	if !ok {
		log.Printf("拉取远程知识点异常:%v", remoteKN)
		return
	}
	nowVersion, ok := remoteKN["now_version"].(string)
	if !ok || nowVersion == "" {
		log.Printf("拉取远程知识点版本号异常:%v", remoteKN)
		return
	}

	err = os.MkdirAll(knWorkPath+knowledge, os.ModePerm)
	if err != nil {
		log.Printf("创建知识点工作区目录异常:%s", err.Error())
		return
	}

	// 刷进去历史版本文件
	var remoteMaxV int
	for _, v := range list {
		knData, _ := v.(map[string]interface{})
		version, _ := knData["version"].(string)
		content, _ := knData["content"].(string)

		v, _ := strconv.Atoi(version)
		if v > remoteMaxV {
			remoteMaxV = v
		}

		err = os.MkdirAll(fmt.Sprintf("%s%s/%s/", knWorkPath, knowledge, version), os.ModePerm)
		if err != nil {
			log.Printf("创建知识点工作区目录异常:%s", err.Error())
			return
		}
		err = ioutil.WriteFile(fmt.Sprintf("%s%s/%s/%s.md", knWorkPath, knowledge, version, knowledge), []byte(content), 0644)
		if err != nil {
			log.Printf("写入知识点文件异常:%s", err.Error())
			return
		}
	}

	knFilePath := fmt.Sprintf("%s/%s.md", knWorkPath, knowledge)
	ok, _ = pkg.PathExists(knFilePath)
	if ok {
		localV, _ := strconv.Atoi(getKNLocalVersion(knowledge))
		nowV, _ := strconv.Atoi(nowVersion)
		if nowV > localV {
			newVPath := fmt.Sprintf("%s%s/%d/%s.md", knWorkPath, knowledge, nowV, knowledge)
			fileMd5, _ := pkg.GetFileMd5(newVPath)
			ok, _ = pkg.PathExists(newVPath)
			if ok {
				knOldFilePath := fmt.Sprintf("%s/%s-old.md", knWorkPath, knowledge)
				os.Rename(knFilePath, knOldFilePath)
				pkg.CopyFile(fmt.Sprintf("%s/%s.md", knWorkPath, knowledge), newVPath)
				pkg.CopyFile(repoObjPath+fileMd5, newVPath)
				log.Printf("版本冲突，本地文件被重命名为:%s-old.md,版本号:%s", knowledge, nowVersion)
			} else {
				// 大概率不会到这里来
				log.Printf("拉取远程知识点成功:%s,最新版本未通过审核,本地不变更,版本号:%s", knowledge, nowVersion)
			}
		} else {
			log.Printf("拉取远程知识点成功:%s,本地无变更,版本号:%s", knowledge, nowVersion)
		}
	} else {
		log.Printf("拉取远程知识点成功:%s,版本号:%s", knowledge, nowVersion)
		pkg.CopyFile(knFilePath, fmt.Sprintf("%s%s/%d/%s.md", knWorkPath, knowledge, remoteMaxV, knowledge))
	}

	// 本地版本号
	err = ioutil.WriteFile(fmt.Sprintf("%s%s/version", knWorkPath, knowledge), []byte(nowVersion), 0644)
	if err != nil {
		log.Printf("写入知识点版本号异常:%s", err.Error())
		return
	}
}

// kPush 推到远程服务器
func kPush(knowledge string) {
	localKNs, err := ReadKIndex()
	if err != nil {
		log.Printf("读取本地仓库异常:%s", err.Error())
		return
	}

	knDes, ok := localKNs[knowledge]
	if !ok {
		log.Printf("本地仓库该知识点不存在:%s", knowledge)
		return
	}

	b, err := ioutil.ReadFile(repoObjPath + knDes.Md5)
	if err != nil {
		log.Printf("读取知识点异常:%s,知识点:%s", err.Error(), knDes.KName)
		return
	}

	localV := getKNLocalVersion(knDes.KName)
	if localV == "" {
		log.Printf("提交异常,知识点本地版本号不存在:%s", knowledge)
		return
	}

	content := string(b)
	form := url.Values{
		"change_log":   {knDes.Changelog},
		"version":      {localV},
		"token":        {UserToken},
		"kname":        {knDes.KName},
		"file_content": {content},
	}

	url := fmt.Sprintf("%s/info/client?action=kadd", ServerHost)
	_, err = pkg.ClientCall(url, form)
	if err != nil {
		if strings.Contains(err.Error(), "i1069") {
			log.Printf("本地知识非最新,重新拉取中,知识点:%s", knDes.KName)
			kPull(knowledge)
			return
		}

		log.Printf("知识点推到远程异常:%s,知识点:%s", err.Error(), knDes.KName)
		return
	}

	log.Printf("知识点推到远程成功:%s", knDes.KName)
}

func kNew(knowledge string) {
	form := url.Values{
		"token": {UserToken},
		"kname": {knowledge},
	}

	url := fmt.Sprintf("%s/info/client?token=%s&action=knew", ServerHost, UserToken)
	_, err := pkg.ClientCall(url, form)
	if err != nil {
		log.Printf("创建知识点异常:%s,知识点:%s", err.Error(), knowledge)
		return
	}
}

func krel(knowledge string, rel string) {
	form := url.Values{
		"token":     {UserToken},
		"kname":     {knowledge},
		"like_name": {rel},
	}

	url := fmt.Sprintf("%s/info/client?token=%s&action=krel", ServerHost, UserToken)
	_, err := pkg.ClientCall(url, form)
	if err != nil {
		log.Printf("创建知识点别名异常:%s,知识点:%s", err.Error(), knowledge)
		return
	}
}

// kAdd 文件工作区加入到本地仓库
func kAdd(Knowledge string, changelog string) {
	localKN, err := ReadKIndex()
	if err != nil {
		log.Printf("读取本地仓库异常:%s", err.Error())
		return
	}

	knPath := fmt.Sprintf("%s%s.md", knWorkPath, Knowledge)
	ok, _ := pkg.PathExists(knPath)
	if !ok {
		log.Printf("该知识点文件不存在,知识点:%s", Knowledge)
		return
	}

	err = replaceImg(knPath)
	if err != nil {
		log.Printf("图片替换异常,err:%s,文件名:%s", err.Error(), Knowledge)
		return
	}

	fileMd5, err := pkg.GetFileMd5(knPath)
	if err != nil {
		log.Printf("获取文件md5异常,err:%s,知识点:%s", err.Error(), Knowledge)
		return
	}

	knDes := localKN[Knowledge]
	if knDes == nil {
		knDes = &KnowledgeDesc{KName: Knowledge}
	} else {
		if knDes.Md5 == fileMd5 {
			return
		}

		// 移除旧文件
		if knDes.Md5 != "" {
			os.Remove(repoObjPath + knDes.Md5)
		}
	}

	knDes.Changelog = changelog
	knDes.Md5 = fileMd5
	knDes.UpdateTime = time.Now().Format("2006-01-02 15:04:05")
	localKN[knDes.KName] = knDes

	_, err = pkg.CopyFile(repoObjPath+fileMd5, knPath)
	if err != nil {
		log.Printf("写入索引异常:%s", err.Error())
		return
	}
	log.Printf("知识点提交到本地仓库成功:%s", knDes.KName)
	WriteKIndex(localKN)
	return
}

// NewDoc 新建文件
func NewDoc(fileName string) {
	l, err := getCategory()
	fmt.Println()
	fmt.Println(fmt.Sprintf("    选择你文章的分类(单选),目前支持的分类如下:"))

	var str string
	for k, v := range l {
		str += fmt.Sprintf("%d:%s ", k+1, v)
	}
	fmt.Printf("    \x1b[%dm%s \x1b[0m\n", 36, str)
	fmt.Println()
	fmt.Print("    请输入分类编号:")

	input := bufio.NewScanner(os.Stdin)

	var category string
	for {
		input.Scan()
		v := strings.TrimSpace(input.Text())
		if v == "" {
			fmt.Print("    输入为空,请重新输入:")
			continue
		}

		vIndex, err := strconv.Atoi(v)
		if err != nil {
			fmt.Print("    输入的编号需为数字,请重新输入:")
			continue
		}
		if vIndex > len(l) || vIndex < 1 {
			fmt.Print("    输入的编号不存在,请重新输入:")
			continue
		}

		category = l[vIndex-1]
		break
	}
	fmt.Println()

	i := strings.Index(fileName, ".")
	if i > 0 {
		fileName = fileName[:i]
	}
	fileName += ".md"

	err = checkFilePath(fileName)
	if err != nil {
		log.Printf("文件名非法,err:" + err.Error())
		return
	}

	exist, _ := pkg.PathExists(workPostsPath + fileName)
	if exist {
		log.Printf("文件已经存在,文件:%s", fileName)
		return
	}

	docFormat := `---
title: %s
category: %s
---`

	docContent := fmt.Sprintf(docFormat, fileName[0:len(fileName)-3], category)
	err = ioutil.WriteFile(workPostsPath+fileName, []byte(docContent), 0644)
	if err != nil {
		log.Printf("本地创建文章异常:%s,文章:%s", err.Error(), fileName)
		return
	}
	log.Printf("文件创建成功,文件名:%s, 分类:%s", fileName, category)
	return
}

// Add 文件工作区加入到本地仓库
func Add(fileName string) {
	if fileName == "." {
		files, err := ioutil.ReadDir(workPostsPath)
		if err != nil {
			log.Printf("读取工作目录异常:%s,目录:%s", err.Error(), workPostsPath)
			return
		}
		for _, s := range files {
			if s.IsDir() || inIgnoreList(s.Name()) {
				continue
			}
			doAdd(s.Name())
		}
	} else {
		doAdd(fileName)
	}
	return
}

// Rm 删除文件
func Rm(fileName string) {
	localRepoPosts, err := ReadIndex()
	if err != nil {
		log.Printf("读取本地仓库异常:%s", err.Error())
		return
	}

	err = checkFilePath(fileName)
	if err != nil {
		log.Printf("文件名非法,err:%s,文件名:%s", err.Error(), fileName)
		return
	}

	local, ok := localRepoPosts[fileName]
	if !ok {
		log.Printf("本地仓库不存在该文件:%s", fileName)
		return
	}

	if local.Status == StatusUserDel || local.Status == StatusAdmDel {
		log.Printf("该文件已经被删除过:%s", fileName)
		return
	}

	local.Status = StatusUserDel
	local.UpdateTime = time.Now().Format("2006-01-02 15:04:05")

	os.Remove(workPostsPath + fileName)
	os.Remove(repoObjPath + local.Md5)
	localRepoPosts[local.FileName] = local

	WriteIndex(localRepoPosts)
	return
}

// Add 文件工作区加入到本地仓库
func doAdd(fileName string) {
	localRepoPosts, err := ReadIndex()
	if err != nil {
		log.Printf("读取本地仓库异常:%s", err.Error())
		return
	}

	err = checkFilePath(fileName)
	if err != nil {
		log.Printf("文件名非法,err:%s,文件名:%s", err.Error(), fileName)
		return
	}

	fileMd5, err := pkg.GetFileMd5(workPostsPath + fileName)
	if err != nil {
		log.Printf("获取文件md5异常,err:%s,文件名:%s", err.Error(), fileName)
		return
	}
	repoPost, ok := localRepoPosts[fileName]
	if ok && repoPost.Md5 == fileMd5 {
		return
	}

	exist, err := pkg.PathExists(workPostsPath + fileName)
	if err != nil {
		log.Printf("判断文件是否存在异常,err:%s,文件名:%s", err.Error(), fileName)
		return
	}
	if !exist {
		log.Printf("该文件不存在,文件名:%s", fileName)
		return
	}

	if pkg.GetFileSize(workPostsPath+fileName) > 2*1024*2014 {
		log.Printf("文章大小不支持2M以上,文件名:%s,文章大小:%d", fileName, pkg.GetFileSize(fileName))
		return
	}

	err = replaceImg(workPostsPath + fileName)
	if err != nil {
		log.Printf("图片替换异常,err:%s,文件名:%s", err.Error(), fileName)
		return
	}

	_, _, err = getMDTileCategory(workPostsPath + fileName)
	if err != nil {
		docFormat := `---
title: 这是标题
category: 文章分类
---`
		log.Printf("获取文件title和分类异常,err:%s,文件名:%s", err.Error(), fileName)
		fmt.Println()
		fmt.Println("文档标准格式如下:")
		fmt.Println(docFormat)
		l, _ := getCategory()
		fmt.Println()
		fmt.Println(fmt.Sprintf("目前支持的分类如下:"))
		fmt.Printf("\x1b[%dm%s \x1b[0m\n", 36, strings.Join(l, " "))
		fmt.Println()
		return
	}

	// 重新获取md5
	fileMd5, err = pkg.GetFileMd5(workPostsPath + fileName)
	if err != nil {
		log.Printf("获取文件md5异常,err:%s,文件名:%s", err.Error(), fileName)
		return
	}

	if repoPost == nil {
		p := &PostDesc{
			FileName:   fileName,
			Md5:        fileMd5,
			UpdateTime: time.Now().Format("2006-01-02 15:04:05"),
		}
		localRepoPosts[fileName] = p
	} else {
		// 移除旧文件
		os.Remove(repoObjPath + repoPost.Md5)

		repoPost.Md5 = fileMd5
		repoPost.UpdateTime = time.Now().Format("2006-01-02 15:04:05")
		localRepoPosts[fileName] = repoPost
	}

	_, err = pkg.CopyFile(repoObjPath+fileMd5, workPostsPath+fileName)
	if err != nil {
		log.Printf("写入索引异常:%s", err.Error())
	}
	log.Printf("文章提交到本地仓库成功:%s", fileName)
	WriteIndex(localRepoPosts)

	return
}

// Checkout 从本地repo迁出到工作区
func Checkout(fileName string) {
	localRepoPosts, err := ReadIndex()
	if err != nil {
		log.Printf("读取本地仓库异常:%s", err.Error())
		return
	}

	if fileName == "." {
		for _, v := range localRepoPosts {
			if v.Status == StatusUserDel || v.Status == StatusAdmDel {
				continue
			}
			_, err = pkg.CopyFile(workPostsPath+v.FileName, repoObjPath+v.Md5)
			if err != nil {
				log.Printf("拷贝文件异常:%s,文件名:%s", err.Error(), v.FileName)
				return
			}
		}
	} else {
		err := checkFilePath(fileName)
		if err != nil {
			log.Printf("路径非法,err:%s,文件名:%s", err.Error(), fileName)
			return
		}

		v, ok := localRepoPosts[fileName]
		if !ok {
			log.Printf("未匹配到任何文件,文件名:%s", fileName)
			return
		}

		if v.Status == "-2" || v.Status == "-3" {
			return
		}

		_, err = pkg.CopyFile(workPostsPath+v.FileName, repoObjPath+v.Md5)
		if err != nil {
			log.Printf("拷贝文件异常:%s,文件名:%s", err.Error(), v.FileName)
			return
		}
	}
}

// Status 本地工作区和本地repo的差异
func Status() {
	localRepoPosts, err := ReadIndex()
	if err != nil {
		log.Printf("读取本地仓库异常:%s", err.Error())
		return
	}

	files, err := ioutil.ReadDir(workPostsPath)
	for _, s := range files {
		if s.IsDir() || inIgnoreList(s.Name()) {
			continue
		}

		v, ok := localRepoPosts[s.Name()]
		if !ok {
			log.Printf("存在新文件:%s", s.Name())
			continue
		}

		md5, err := pkg.GetFileMd5(workPostsPath + s.Name())
		if err != nil {
			log.Printf("获取md5异常:%s,文件名:%s", err.Error(), s.Name())
			continue
		}
		if md5 != v.Md5 {
			log.Printf("存在变更文件:%s", s.Name())
		}
	}

	for _, v := range localRepoPosts {
		b, _ := pkg.PathExists(workPostsPath + v.FileName)
		if !b && v.Status != "-2" && v.Status != "-3" {
			log.Printf("文件被删除:%s", v.FileName)
		}
	}
}

// Update 版本升级
func Update(auto bool) {
	remoteV, err := getRemoteVersion()
	if err != nil {
		log.Printf("获取版本号异常:%s", err.Error())
		return
	}

	// 更新时间
	WriteUpdateTime()

	// 判断是否需要升级版本
	if !pkg.VersionCompare(remoteV, version) {
		if !auto {
			log.Printf("当前已经是最新版本:%s", version)
		}
		return
	}
	defer os.Exit(0)

	// 自动检测的给用户选择
	if auto {
		fmt.Println()
		fmt.Printf("    \x1b[%dm%s \x1b[0m\n", 36, "检测到有新版本,按n取消,按其余任意键升级~")
		fmt.Printf("    请输入:")
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		v := strings.TrimSpace(input.Text())
		if strings.ToLower(v) == "n" {
			fmt.Println("    取消升级~")
			return
		}
	}

	log.Printf("当前版本:%s,远程版本:%s,升级中....", version, remoteV)

	oldFile := "doc"
	newFile := fmt.Sprintf("doc_%s", remoteV)

	sysType := runtime.GOOS
	if sysType == "windows" {
		oldFile += ".exe"
		newFile += ".exe"
	}

	err = pkg.DownLoadFile(fmt.Sprintf("https://zpic.jiaoliuqu.com/%s", newFile), newFile)
	if err != nil {
		log.Printf("获取新版本文件异常:%s", err.Error())
		return
	}

	if pkg.GetFileSize(newFile) <= 2048 {
		log.Printf("新版本程序文件大小异常,停止更新")
		return
	}

	err = os.Chmod(newFile, 0777)
	if err != nil {
		log.Printf("修改程序权限异常:%s", err.Error())
		return
	}

	err = os.Rename(newFile, oldFile)
	if err != nil {
		log.Printf("版本覆盖失败:%s", err.Error())
		return
	}

	log.Printf("升级版本完成当前版本号:%s", remoteV)
}

// Update2Ser 更新版本到服务器
func Update2Ser(version string) {
	fileNameMac := "doc_" + version
	token, err := getUploadToken(fileNameMac)
	if err != nil {
		log.Printf("拉取七牛文件上传凭证异常:%s", err.Error())
		return
	}
	_, err = qiniu.UploadFile("doc", fileNameMac, token)
	if err != nil {
		log.Printf("程序mac版本上传异常:%s", err.Error())
		return
	}
	log.Printf("程序mac版本上传成功,文件:%s", fileNameMac)

	fileNameExe := "doc_" + version + ".exe"
	token, err = getUploadToken(fileNameExe)
	if err != nil {
		log.Printf("拉取七牛文件上传凭证异常:%s", err.Error())
		return
	}
	_, err = qiniu.UploadFile("doc.exe", fileNameExe, token)
	if err != nil {
		log.Printf("程序win版本上传异常:%s", err.Error())
		return
	}
	log.Printf("程序win版本上传成功,文件:%s", fileNameExe)

	form := url.Values{"version": {version}}
	_, err = pkg.ClientCall(ServerHost+"/info/client?action=setVersion&token="+UserToken, form)
	if err != nil {
		log.Printf("版本设置失败:%s", err.Error())
		return
	}

	log.Printf("版本设置成功当前服务器版本号:%s", version)
}

// UpdateInstallShell 更新安装脚本
func UpdateInstallShell() {
	installMac := "install.sh"

	token, err := getUploadToken(installMac)
	if err != nil {
		log.Printf("拉取七牛文件上传凭证异常:%s", err.Error())
		return
	}

	_, err = qiniu.UploadFile(installMac, installMac, token)
	if err != nil {
		log.Printf("上传安装脚本异常err:%s", err.Error())
		return
	}
	log.Println("安装脚本更新成功")
}

// checkFilePath 检测文件路径是否非法,暂时只支持同级目录
func checkFilePath(path string) error {
	if strings.Contains(path, " ") {
		return errors.New("路径不能含有空格")
	}
	if strings.Contains(path, "/") || strings.Contains(path, "\\") {
		return errors.New("不支持多级路径")
	}
	if strings.ToLower(pkg.GetExt(path)) != ".md" {
		return errors.New("只支持md格式后缀")
	}
	return nil
}

func inIgnoreList(file string) bool {
	var ignoreList []string
	b, _ := ioutil.ReadFile(".ignore")
	if len(b) > 0 {
		ignoreList = strings.Split(string(b), "\n")
	}

	ignoreList = append(ignoreList, ".DS_Store")

	for _, v := range ignoreList {
		if v == file {
			return true
		}
	}
	return false
}

func isSupportImg(ext string) bool {
	ImgExtList := []string{".jpeg", ".gif", ".png", ".jpg"}
	for _, v := range ImgExtList {
		if v == ext {
			return true
		}
	}
	return false
}

// replaceImg 本地图片替换成七牛图片
func replaceImg(filePath string) error {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return errors.New("读取文章异常:" + err.Error())
	}
	if len(b) == 0 {
		return nil
	}

	content := string(b)
	re, _ := regexp.Compile(`\!\[.*?\]\((.*?)\)`)
	c := re.FindAllSubmatch([]byte(content), -1)
	if len(c) == 0 {
		return nil
	}

	var imgToken string
	imgToken, err = getUploadToken("")
	if err != nil {
		return errors.New("拉取七牛文件上传凭证异常:" + err.Error())
	}

	for _, v := range c {
		if len(v) < 2 {
			continue
		}

		imgURL := string(v[1])
		ext := pkg.GetExt(imgURL)
		if !isSupportImg(ext) {
			log.Printf("该图片格式不支持%s", ext)
			continue
		}

		if strings.Contains(imgURL, "jiaoliuqu.com") {
			continue
		}

		if !strings.HasPrefix(imgURL, "../img/") && strings.HasPrefix(imgURL, `..\img\`) {
			log.Printf("该图片路径非法%s,正确格式为../img/xx", imgURL)
			continue
		}

		ret, err := qiniu.UploadFile(imgURL[1:], pkg.GetKey()+ext, imgToken)
		if err != nil {
			log.Printf("上传图片异常:%s,imgURL:%s", err.Error(), imgURL)
			continue
		}

		newImg := fmt.Sprintf("https://zpic.jiaoliuqu.com/%s", ret.Key)
		content = strings.Replace(content, imgURL, newImg, -1)
		log.Printf("图片替换成功,原始图片:%s,新图片:%s", imgURL, newImg)
	}

	err = ioutil.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return errors.New("写入文章异常:" + err.Error())
	}
	return nil
}

// getMDTileCategory 获取title和分类
func getMDTileCategory(filePath string) (string, string, error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", "", err
	}

	r := bufio.NewReader(strings.NewReader(string(b)))
	line1, _, err := r.ReadLine()
	if err != nil {
		return "", "", errors.New("第一行读取错误:" + err.Error())
	}

	if string(line1) != "---" {
		return "", "", errors.New("格式错误,文档第一行需---开头")
	}

	line2, _, err := r.ReadLine()
	if err != nil {
		return "", "", errors.New("第二行读取错误:" + err.Error())
	}

	line2Str := string(line2)
	if len(line2Str) <= 6 {
		return "", "", errors.New("格式错误,文档第二行需title:开头")
	}
	if line2Str[:6] != "title:" {
		return "", "", errors.New("格式错误,文档第二行需title:开头")
	}

	title := strings.TrimSpace(line2Str[6:])
	if title == "" {
		return "", "", errors.New("title不能为空")
	}

	line3, _, err := r.ReadLine()
	if err != nil {
		return "", "", errors.New("第三行读取错误:" + err.Error())
	}
	line3Str := string(line3)
	if len(line3Str) <= 9 {
		return "", "", errors.New("格式错误,文档第三行需category:开头")
	}
	if line3Str[:9] != "category:" {
		return "", "", errors.New("格式错误,文档第三行需category:开头")
	}

	category := strings.TrimSpace(line3Str[9:])
	if category == "" {
		return "", "", errors.New("category不能为空")
	}
	return title, category, nil
}

// getRemoteVersion 获取服务器版本号
func getRemoteVersion() (string, error) {
	data, err := pkg.ClientCall(ServerHost+"/info/client?action=version&token="+UserToken, url.Values{})
	if err != nil {
		return "", err
	}
	i, ok := data.(map[string]interface{})
	if !ok {
		return "", errors.New(fmt.Sprintf("获取版本返回值格式异常:%v", data))
	}

	v, _ := i["version"].(string)
	if v == "" {
		return "", fmt.Errorf("拉取版本异常,version字段为空,后端返回内容:%v", data)
	}

	arr := strings.Split(v, ".")
	if len(arr) != 3 {
		return "", errors.New("拉取版本异常,格式错误" + v)
	}
	return v, nil
}

// getUploadToken 获取七牛token
func getUploadToken(key string) (string, error) {
	u := ServerHost + "/basic/getPicToken?token=" + UserToken
	if key != "" {
		u += "&key=" + key
	}
	data, err := pkg.ClientCall(u, url.Values{})
	if err != nil {
		return "", err
	}

	i, ok := data.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("获取七牛返回值格式异常%v", data)
	}

	token := i["token"].(string)
	if token == "" {
		return "", fmt.Errorf("获取失败,返回内容:%v", data)
	}
	return token, nil
}

// getCategory 获取文章分类
func getCategory() ([]string, error) {
	u := ServerHost + "/info/client?action=getCategory"
	data, err := pkg.ClientCall(u, url.Values{})
	if err != nil {
		return nil, err
	}
	list, ok := data.([]interface{})
	if !ok {
		return nil, nil
	}
	var l []string
	for _, v := range list {
		m, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		t, ok := m["name"].(string)
		if !ok {
			continue
		}
		l = append(l, t)
	}
	return l, nil
}
