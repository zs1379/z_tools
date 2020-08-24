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
	"sort"
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
	version    = "0.0.6"
)

var (
	repoObjPath   = "./.repo/objects/"
	tokenPath     = "./.repo/token"
	envPath       = "./.repo/env"
	indexPath     = "./.repo/index"
	imgPath       = "./img/"
	workPostsPath = "./posts/"
)

// PostDesc 文章描述
type PostDesc struct {
	Title      string `json:"title"`       // title
	FileName   string `json:"file_name"`   // 文件名称
	UpdateTime string `json:"update_time"` // 文章更新时间
	Md5        string `json:"file_md5"`    // 文章MD5
	Status     string `json:"status"`      // 文件状态 -2:自己删除 -3:管理员删除
}

func init() {
	err := os.MkdirAll(workPostsPath, os.ModePerm)
	if err != nil {
		log.Printf("创建工作区目录异常:%s", err.Error())
		return
	}
	err = os.MkdirAll(imgPath, os.ModePerm)
	if err != nil {
		log.Printf("创建img目录异常:%s", err.Error())
		return
	}
	err = os.MkdirAll(repoObjPath, os.ModePerm)
	if err != nil {
		log.Printf("创建object目录异常:%s", err.Error())
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
						log.Printf("请输入token")
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
				Description: "1. doc new test.md 本地自动生成一篇test.md的空文档",
				ArgsUsage:   "[文件名]",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入文件名")
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
						log.Printf("请输入文件名")
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
						log.Printf("请输入文件名")
						return nil
					}
					fileName := c.Args().Get(0)
					Rm(fileName)
					return nil
				},
			},
			{
				Name:        "status",
				Usage:       "文件变更",
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
						log.Printf("请输入文件名")
						return nil
					}
					fileName := c.Args().Get(0)
					Checkout(fileName)
					return nil
				},
			},
			{
				Name:        "update",
				Usage:       "版本升级",
				Description: "1. doc update 升级程序版本",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					Update()
					return nil
				},
			},
			{
				Name:        "SetVersion",
				Usage:       "版本设置-[需管理员]",
				Description: "1. doc SetVersion 0.0.3 版本设置到0.0.3",
				ArgsUsage:   "[版本号]",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入版本号")
						return nil
					}
					SetVersion(c.Args().Get(0))
					return nil
				},
			},
			{
				Name:        "updateFile",
				Usage:       "程序上传-[需管理员]",
				Description: "1. doc updateFile 0.0.3 将0.0.3版本程序上传到七牛",
				ArgsUsage:   "[版本号]",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入版本号")
						return nil
					}
					UpdateFile(c.Args().Get(0))
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

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
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

// Pull 拉取远程
func Pull() {
	if UserToken == "" {
		log.Printf("用户token为空,请先初始化")
		return
	}

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
		m := v.(map[string]interface{})
		remote.FileName, _ = m["file_name"].(string)
		remote.Md5, _ = m["file_md5"].(string)
		remote.UpdateTime, _ = m["update_time"].(string)
		remote.Status, _ = m["status"].(string)

		if remote.FileName == "" || remote.Md5 == "" || remote.UpdateTime == "" {
			log.Printf("拉取文章异常,返回字段不全,file:%s,md5:%s,time:%s", remote.FileName, remote.Md5, remote.UpdateTime)
			continue
		}

		// 如果文件远程被删除,则本地也相应删除
		if remote.Status == "-2" || remote.Status == "-3" {
			local, ok := localRepoPosts[remote.FileName]
			if ok && local.Status != "-2" && local.Status != "-3" {
				os.Remove(repoObjPath + local.Md5)
				os.Remove(local.FileName)
				log.Printf("文件远程被删除,删除本地文件:%s", remote.FileName)
			}
			localRepoPosts[remote.FileName] = &remote
			continue
		}

		// 更新本地repo
		local, ok := localRepoPosts[remote.FileName]
		if ok {
			if local.Md5 == remote.Md5 || pkg.TimeCompare(local.UpdateTime, remote.UpdateTime) {
				continue
			}
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
		remote.Title, _ = data["title"].(string)

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
		localRepoPosts[remote.FileName] = &remote
	}

	WriteIndex(localRepoPosts)
}

// Push 推到远程服务器
func Push() {
	if UserToken == "" {
		log.Printf("用户token为空,请先初始化")
		return
	}

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
		if p.Status == "-3" || p.Status == "-2" {
			local, ok := localRepoPosts[p.FileName]
			if ok && local.Status != "-2" && local.Status != "-3" {
				os.Remove(repoObjPath + local.Md5)
				os.Remove(local.FileName)
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
		}

		// 本地删除的情况,单独调用接口
		if v.Status == "-2" && r.Status != "-2" {
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

		b, err := ioutil.ReadFile(repoObjPath + v.Md5)
		if err != nil {
			log.Printf("读取文章异常:%s,文章:%s", err.Error(), v.FileName)
			continue
		}

		content := string(b)
		form := url.Values{
			"filename": {v.FileName},
			"token":    {UserToken},
			"md5":      {v.Md5},
			"content":  {content},
			"title":    {v.Title},
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

// NewDoc 新建文件
func NewDoc(fileName string) {
	err := checkFilePath(fileName)
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
---`

	docContent := fmt.Sprintf(docFormat, fileName[0:len(fileName)-3])
	err = ioutil.WriteFile(workPostsPath+fileName, []byte(docContent), 0644)
	if err != nil {
		log.Printf("本地创建文章异常:%s,文章:%s", err.Error(), fileName)
		return
	}
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

	if local.Status == "-3" || local.Status == "-2" {
		log.Printf("该文件已经被删除过:%s", fileName)
		return
	}

	local.Status = "-2"
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

	fileMd5, err := pkg.GetFileMd5(workPostsPath + fileName)
	if err != nil {
		log.Printf("获取文件md5异常,err:%s,文件名:%s", err.Error(), fileName)
		return
	}

	title, err := getMDTile(workPostsPath + fileName)
	if err != nil {
		log.Printf("获取文件title异常,err:%s,文件名:%s", err.Error(), fileName)
		return
	}

	repoPost, ok := localRepoPosts[fileName]
	if ok && repoPost.Md5 == fileMd5 {
		return
	}

	var uuid string
	if repoPost == nil {
		p := &PostDesc{
			Title:      title,
			FileName:   fileName,
			Md5:        fileMd5,
			UpdateTime: time.Now().Format("2006-01-02 15:04:05"),
		}
		localRepoPosts[uuid] = p
	} else {
		// 移除旧文件
		os.Remove(repoObjPath + repoPost.Md5)

		repoPost.Title = title
		repoPost.Md5 = fileMd5
		repoPost.UpdateTime = time.Now().Format("2006-01-02 15:04:05")
		localRepoPosts[uuid] = repoPost
	}

	_, err = pkg.CopyFile(repoObjPath+fileMd5, workPostsPath+fileName)
	if err != nil {
		log.Printf("写入索引异常:%s", err.Error())
	}
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
			if v.Status == "-2" || v.Status == "-3" {
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
func Update() {
	remoteV, err := getRemoteVersion()
	if err != nil {
		log.Printf("获取版本号异常:%s", err.Error())
		return
	}

	// 判断是否需要升级版本
	if !pkg.VersionCompare(remoteV, version) {
		log.Printf("已经是最新版本")
		return
	}

	log.Printf("检测到新版本,当前版本:%s,远程版本:%s,升级中....", version, remoteV)

	newFile := fmt.Sprintf("doc_%s", remoteV)
	err = pkg.DownLoadFile(fmt.Sprintf("https://zpic.jiaoliuqu.com/%s", newFile), newFile)
	if err != nil {
		log.Printf("获取新版本文件异常:%s", err.Error())
		return
	}

	if pkg.GetFileSize(newFile) <= 2048 {
		log.Printf("程序文件大小异常")
		return
	}

	err = os.Chmod(newFile, 0777)
	if err != nil {
		log.Printf("修改程序权限异常:%s", err.Error())
		return
	}

	err = os.Rename(newFile, "doc")
	if err != nil {
		log.Printf("版本覆盖失败:%s", err.Error())
		return
	}

	log.Printf("升级版本完成当前版本号:%s", remoteV)
}

// SetVersion 设置服务器版本号
func SetVersion(version string) {
	form := url.Values{"version": {version}}
	_, err := pkg.ClientCall(ServerHost+"/info/client?action=SetVersion&token="+UserToken, form)
	if err != nil {
		log.Printf("版本设置失败:%s", err.Error())
		return
	}
	log.Printf("版本设置成功当前服务器版本号:%s", version)
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

// replaceImg 替换图片
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

	// 拉取七牛token
	var imgToken string
	if len(c) > 0 {
		if UserToken == "" {
			return errors.New("用户token为空,请先初始化")
		}

		data, err := pkg.ClientCall(ServerHost+"/basic/getPicToken?token="+UserToken, url.Values{})
		if err != nil {
			return errors.New("拉取图片上传token异常:" + err.Error())
		}

		i, ok := data.(map[string]interface{})
		if !ok {
			return errors.New(fmt.Sprintf("拉取图片上传token异常,返回值格式异常:%v", data))
		}
		imgToken = i["token"].(string)
		if imgToken == "" {
			return errors.New("拉取图片上传token异常,token为空")
		}
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
			log.Printf("该图片路径非法%s,格式为../img/xx", imgURL)
			continue
		}

		qNKey := pkg.GetKey() + ext
		ret, err := qiniu.UploadFile(imgURL[1:], qNKey, imgToken)
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

// getMDTile 获取title
func getMDTile(filePath string) (string, error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	r := bufio.NewReader(strings.NewReader(string(b)))
	line1, _, err := r.ReadLine()
	if err != nil {
		return "", errors.New("第一行读取错误:" + err.Error())
	}

	if string(line1) != "---" {
		return "", errors.New("格式错误,文档第一行需---开头")
	}

	line2, _, err := r.ReadLine()
	if err != nil {
		return "", errors.New("第二行读取错误:" + err.Error())
	}

	line2Str := string(line2)
	if len(line2Str) <= 6 {
		return "", errors.New("格式错误,文档第二行需title:开头")
	}
	if line2Str[:6] != "title:" {
		return "", errors.New("格式错误,文档第二行需title:开头")
	}

	title := strings.TrimSpace(line2Str[6:])
	if title == "" {
		return "", errors.New("title不能为空")
	}
	return title, nil
}

func getRemoteVersion() (string, error) {
	data, err := pkg.ClientCall(ServerHost+"/info/client?action=version&token="+UserToken, url.Values{})
	if err != nil {
		return "", err
	}
	i, ok := data.(map[string]interface{})
	if !ok {
		return "", errors.New(fmt.Sprintf("获取版本返回值格式异常:%v", data))
	}

	v := i["version"].(string)
	if v == "" {
		return "", errors.New("拉取版本异常,token为空")
	}
	arr := strings.Split(v, ".")
	if len(arr) != 3 {
		return "", errors.New("拉取版本异常,格式错误" + v)
	}
	return v, nil
}

// UpdateFile 上传程序到七牛
func UpdateFile(version string) {
	data, err := pkg.ClientCall(ServerHost+"/basic/getPicToken?token="+UserToken, url.Values{})
	if err != nil {
		log.Printf("拉取七牛上传token异常:%s", err.Error())
		return
	}

	i, ok := data.(map[string]interface{})
	if !ok {
		log.Printf(fmt.Sprintf("拉取七牛上传token异常,返回值格式异常:%v", data))
		return
	}
	token := i["token"].(string)
	if token == "" {
		errors.New("拉取图片上传token异常,token为空")
		return
	}

	fileName := "doc_" + version
	_, err = qiniu.UploadFile(fileName, fileName, token)
	if err != nil {
		log.Printf("程序上传异常:%s", err.Error())
		return
	}

	log.Printf("上传成功,文件:%s", fileName)
}
