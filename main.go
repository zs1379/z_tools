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
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"z_tools/pkg"
	"z_tools/pkg/qiniu"
)

var (
	conf       *Config                                   // 配置
	workDir    string                                    // 工作目录
	fileSep    = "/"                                     // 目录分隔符
	ignoreList = []string{"doc", "doc.exe", "conf.json"} // 忽略的文件列表
)

// Config 配置
type Config struct {
	ServerHost string `json:"server"`
	UserToken  string `json:"token"`
}

// DocDesc 文章描述
type DocDesc struct {
	Title      string `json:"title"`       // title
	FileName   string `json:"file_name"`   // 文件名称
	UpdateTime string `json:"update_time"` // 文章更新时间
	Md5        string `json:"file_md5"`    // 文章MD5
}

// respData 返回给客户端的数据
type respData struct {
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

var help = `用法:
	./doc 命令 参数
支持的命令有:
	new         新建文章
	add         工作区文章新增/变更提交到本地仓库 
	pull        服务器拉取最新文章列表
	push        本地仓库提交到服务器
	status      比对本地仓库和工作区的文件变更
	checkout    恢复本地仓库的指定文件到工作区 参数:文件名
`

// ParseConfig 解析配置
func ParseConfig() (*Config, error) {
	b, err := ioutil.ReadFile("conf.json")
	if err != nil {
		return nil, err
	}
	var conf Config
	err = json.Unmarshal(b, &conf)
	if err != nil {
		return nil, err
	}
	if conf.ServerHost == "" {
		return nil, errors.New("未配置server")
	}
	if conf.UserToken == "" {
		return nil, errors.New("未配置token")
	}
	return &conf, nil
}

func main() {
	if len(os.Args) == 1 {
		log.Printf(help)
		return
	}

	cfg, err := ParseConfig()
	if err != nil {
		log.Printf("读取配置异常:%s", err.Error())
		return
	}
	conf = cfg

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Printf("获取当前目录异常:%s", err.Error())
		return
	}
	workDir = dir

	sysType := runtime.GOOS
	if sysType == "windows" {
		fileSep = "\\"
	} else {
		fileSep = "/"
	}

	err = os.MkdirAll(workDir+"/repo", os.ModePerm)
	if err != nil {
		log.Printf("创建repo目录异常:%s", err.Error())
		return
	}

	method := strings.ToLower(os.Args[1])
	switch method {
	case "pull":
		Pull()
	case "new":
		if len(os.Args) < 3 {
			log.Printf("请输入文件名")
			return
		}
		fileName := os.Args[2]
		NewDoc(fileName)
	case "add":
		if len(os.Args) < 3 {
			log.Printf("请输入文件名")
			return
		}
		fileName := os.Args[2]
		Add(fileName)
	case "push":
		Push()
	case "checkout":
		if len(os.Args) < 3 {
			log.Printf("请输入文件名")
			return
		}
		fileName := os.Args[2]
		Checkout(fileName)
	case "status":
		Status()
	case "help":
		log.Printf(help)
	case "md5":
		if len(os.Args) < 3 {
			log.Printf("请输入文件名")
			return
		}
		fileName := os.Args[2]
		md5, err := pkg.GetFileMd5(fileName)
		log.Printf("md5:%s,err:%s", md5, err.Error())
	default:
		log.Printf("未支持操作:%s", method)
	}
}

// ReadIndex 读取索引
func ReadIndex() (map[string]*DocDesc, error) {
	m := make(map[string]*DocDesc)

	b, _ := ioutil.ReadFile(getIndexPath())
	if len(b) == 0 {
		return m, nil
	}

	var list []*DocDesc
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
func WriteIndex(m map[string]*DocDesc) error {
	var list []*DocDesc
	for _, v := range m {
		list = append(list, v)
	}

	b, err := json.Marshal(list)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(getIndexPath(), b, 0644)
	if err != nil {
		return err
	}
	return nil
}

// Pull 拉取远程
func Pull() {
	localPosts, err := ReadIndex()
	if err != nil {
		log.Printf("读取索引异常:%s", err.Error())
		return
	}

	body, err := pkg.HttpGet(fmt.Sprintf("%s/info/clientPost?action=getList&token=%s", conf.ServerHost, conf.UserToken))
	if err != nil {
		log.Printf("拉取远程列表异常:%s", err.Error())
		return
	}

	var ret respData
	json.Unmarshal(body, &ret)
	if ret.Msg != "" {
		log.Printf("拉取远程列表异常:%s", ret.Msg)
		return
	}

	remotePosts, _ := ret.Data.([]interface{})
	for _, v := range remotePosts {
		var remote DocDesc
		m := v.(map[string]interface{})
		remote.FileName = m["file_name"].(string)
		remote.Md5 = m["file_md5"].(string)
		remote.UpdateTime = m["update_time"].(string)

		// 更新本地repo
		local, ok := localPosts[remote.FileName]
		if ok {
			if local.Md5 == remote.Md5 || timeCompare(local.UpdateTime, remote.UpdateTime) {
				continue
			}
		}

		url := fmt.Sprintf("%s/info/clientPost?token=%s&action=get&filename=%s", conf.ServerHost, conf.UserToken, remote.FileName)
		body, err := pkg.HttpGet(url)
		if err != nil {
			log.Printf("拉取文章异常:%s,文章:%s", err.Error(), remote.FileName)
			continue
		}

		var ret respData
		json.Unmarshal(body, &ret)
		if ret.Msg != "" {
			log.Printf("拉取远程文章异常:%s,文章:%s", ret.Msg, remote.FileName)
			continue
		}
		data, ok := ret.Data.(map[string]interface{})
		if !ok {
			log.Printf("拉取远程文章格式异常:%v,文章:%s", ret.Data, remote.FileName)
			continue
		}

		content := data["content"].(string)
		remote.Title = data["title"].(string)

		err = pkg.Write2File([]byte(content), getRepoFilePath(remote.Md5))
		if err != nil {
			log.Printf("写入文章异常:%s,文章:%s", err.Error(), remote.FileName)
			continue
		}

		_, err = pkg.CopyFile(getWorkFilePath(remote.FileName), getRepoFilePath(remote.Md5))
		if err != nil {
			log.Printf("拷贝文章异常:%s,文章:%s", err.Error(), remote.FileName)
			continue
		}

		if local != nil {
			os.Remove(getRepoFilePath(local.Md5))
		}

		log.Printf("拉取远程文章成功:%s", remote.FileName)
		localPosts[remote.FileName] = &remote
	}

	WriteIndex(localPosts)
}

// Push 推到远程服务器
func Push() {
	localList, err := ReadIndex()
	if err != nil {
		log.Printf("读取索引异常:%s", err.Error())
		return
	}

	ret, err := pkg.HttpGet(fmt.Sprintf("%s/info/clientPost?action=getList&token=%s", conf.ServerHost, conf.UserToken))
	if err != nil {
		log.Printf("拉取远程文章列表异常:%s", err.Error())
		return
	}

	r := respData{}
	json.Unmarshal(ret, &r)
	if r.Msg != "" {
		log.Printf("读取远程文章列表异常:%s", r.Msg)
		return
	}

	remote := make(map[string]DocDesc)
	l, _ := r.Data.([]interface{})
	for _, v := range l {
		m := v.(map[string]interface{})
		var a DocDesc
		a.FileName = m["file_name"].(string)
		a.Md5 = m["file_md5"].(string)
		a.UpdateTime = m["update_time"].(string)
		remote[a.FileName] = a
	}

	for _, v := range localList {
		r, ok := remote[v.FileName]
		if ok {
			if r.Md5 == v.Md5 || timeCompare(r.UpdateTime, v.UpdateTime) {
				continue
			}
		}

		b, err := ioutil.ReadFile(getRepoFilePath(v.Md5))
		if err != nil {
			log.Printf("读取文章异常:%s,文章:%s", err.Error(), v.FileName)
			continue
		}

		content := string(b)
		form := url.Values{
			"file_name": {v.FileName},
			"token":     {conf.UserToken},
			"md5":       {v.Md5},
			"content":   {content},
			"title":     {v.Title},
		}
		url := fmt.Sprintf("%s/info/clientPost?token=%s&action=add&filename=%s", conf.ServerHost, conf.UserToken, v.FileName)
		resp, err := pkg.HttpPost(url, form)
		if err != nil {
			log.Printf("上传文章异常:%s,文章:%s", err.Error(), v.FileName)
			continue
		}

		ret := respData{}
		err = json.Unmarshal(resp, &ret)
		if ret.Msg != "" {
			log.Printf("推到远程出现异常:%s,文件:%s", ret.Msg, v.FileName)
			continue
		}

		log.Printf("文章推到远程成功:%s", v.FileName)
	}
}

// NewDoc 新建文件
func NewDoc(fileName string) {
	err := checkFilePath(fileName)
	if err != nil {
		log.Printf("文件名非法,err:" + err.Error())
		return
	}

	exist, _ := pkg.PathExists(fileName)
	if exist {
		log.Printf("文件已经存在,文件:%s", fileName)
		return
	}
	docFormat := `---
title: %s
date: %s
tags:
---`

	docContent := fmt.Sprintf(docFormat, fileName, time.Now().Format("2006-01-02 15:04:05"))
	err = ioutil.WriteFile(getWorkFilePath(fileName), []byte(docContent), 0644)
	if err != nil {
		log.Printf("本地创建文章异常:%s,文章:%s", err.Error(), fileName)
		return
	}
	return
}

// Add 文件工作区加入到本地仓库
func Add(fileName string) {
	if fileName == "." {
		files, err := ioutil.ReadDir(workDir)
		if err != nil {
			log.Printf("读取工作目录异常:%s,目录:%s", err.Error(), workDir)
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

// Add 文件工作区加入到本地仓库
func doAdd(fileName string) {
	localPosts, err := ReadIndex()
	if err != nil {
		log.Printf("读取索引异常:%s", err.Error())
		return
	}

	err = checkFilePath(fileName)
	if err != nil {
		log.Printf("文件名非法,err:%s,文件名:%s", err.Error(), fileName)
		return
	}

	exist, err := pkg.PathExists(fileName)
	if err != nil {
		log.Printf("判断文件是否存在异常,err:%s,文件名:%s", err.Error(), fileName)
		return
	}
	if !exist {
		log.Printf("该文件不存在,文件名:%s", fileName)
		return
	}

	if pkg.GetFileSize(fileName) > 2*1024*2014 {
		log.Printf("文章大小不支持2M以上,文件名:%s,文章大小:%d", fileName, pkg.GetFileSize(fileName))
		return
	}

	err = replaceImg(fileName)
	if err != nil {
		log.Printf("图片替换异常,err:%s,文件名:%s", err.Error(), fileName)
		return
	}

	fileMd5, err := pkg.GetFileMd5(fileName)
	if err != nil {
		log.Printf("获取文件md5异常,err:%s,文件名:%s", err.Error(), fileName)
		return
	}

	title, err := getMDTile(fileName)
	if err != nil {
		log.Printf("获取文件title异常,err:%s,文件名:%s", err.Error(), fileName)
		return
	}

	var repoArticle *DocDesc
	for _, v := range localPosts {
		if v.FileName == fileName {
			if v.Md5 == fileMd5 {
				log.Printf("内容无变更,文件名:%s", v.FileName)
				return
			}
			repoArticle = v
		}
	}

	var uuid string
	if repoArticle == nil {
		a := &DocDesc{
			Title:      title,
			FileName:   fileName,
			Md5:        fileMd5,
			UpdateTime: time.Now().Format("2006-01-02 15:04:05"),
		}
		localPosts[uuid] = a
	} else {
		// 移除旧文件
		os.Remove(getRepoFilePath(repoArticle.Md5))

		repoArticle.Title = title
		repoArticle.Md5 = fileMd5
		repoArticle.UpdateTime = time.Now().Format("2006-01-02 15:04:05")
		localPosts[uuid] = repoArticle
	}

	_, err = pkg.CopyFile(getRepoFilePath(fileMd5), fileName)
	if err != nil {
		log.Printf("写入索引异常:%s", err.Error())
	}
	WriteIndex(localPosts)

	return
}

// Checkout 从本地repo迁出到工作区
func Checkout(fileName string) {
	localRepo, err := ReadIndex()
	if err != nil {
		log.Printf("读取索引异常:%s", err.Error())
		return
	}

	if fileName == "." {
		for _, v := range localRepo {
			_, err = pkg.CopyFile(v.FileName, getRepoFilePath(v.Md5))
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

		exist := false
		for _, v := range localRepo {
			if v.FileName == fileName {
				_, err = pkg.CopyFile(v.FileName, getRepoFilePath(v.Md5))
				if err != nil {
					log.Printf("拷贝文件异常:%s,文件名:%s", err.Error(), v.FileName)
					return
				}
				exist = true
				break
			}
		}
		if !exist {
			log.Printf("未匹配到任何文件,文件名:%s", fileName)
		}
	}
}

// Status 本地工作区和本地repo的差异
func Status() {
	localRepo, err := ReadIndex()
	if err != nil {
		log.Printf("读取索引异常:%s", err.Error())
		return
	}

	files, err := ioutil.ReadDir(workDir)
	for _, s := range files {
		if s.IsDir() || inIgnoreList(s.Name()) {
			continue
		}

		bExist := false
		for _, v := range localRepo {
			if v.FileName == s.Name() {
				md5, err := pkg.GetFileMd5(getWorkFilePath(s.Name()))
				if err != nil {
					log.Printf("获取md5异常:%s,文件名:%s", err.Error(), s.Name())
					continue
				}
				if md5 != v.Md5 {
					log.Printf("存在变更文件:%s", s.Name())
				}
				bExist = true
			}
		}
		if !bExist {
			log.Printf("存在新文件:%s", s.Name())
		}
	}

	for _, v := range localRepo {
		b, _ := pkg.PathExists(getWorkFilePath(v.FileName))
		if !b {
			log.Printf("文件被删除:%s", v.FileName)
		}
	}
}

// checkFilePath 检测文件路径是否非法,暂时只支持同级目录
func checkFilePath(path string) error {
	if strings.Contains(path, "/") || strings.Contains(path, "\\") {
		return errors.New("不支持多级路径")
	}
	if strings.ToLower(pkg.GetExt(path)) != ".md" {
		return errors.New("只支持md格式后缀")
	}
	return nil
}

func inIgnoreList(file string) bool {
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
func replaceImg(fileName string) error {
	b, err := ioutil.ReadFile(getWorkFilePath(fileName))
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

	body, err := pkg.HttpGet(conf.ServerHost + "/basic/getPicToken?token=" + conf.UserToken)
	if err != nil {
		return errors.New("拉取图片上传token异常:" + err.Error())
	}

	var ret respData
	json.Unmarshal(body, &ret)
	i, ok := ret.Data.(map[string]interface{})
	if !ok {
		return errors.New("拉取图片上传token异常:" + err.Error())
	}
	imgToken := i["token"].(string)
	if imgToken == "" {
		return errors.New("拉取图片上传token异常,token为空")
	}

	for _, v := range c {
		if len(v) < 2 {
			continue
		}
		imgURL := string(v[1])
		ext := pkg.GetExt(imgURL)
		if !isSupportImg(ext) {
			continue
		}
		if strings.Contains(imgURL, "jiaoliuqu.com") {
			continue
		}

		qNKey := pkg.GetKey() + ext
		var localPath string
		if strings.Contains(imgURL, "http") || strings.Contains(imgURL, "https") {
			localPath = fmt.Sprintf("img/%s", qNKey)
			err := pkg.DownLoadFile(imgURL, localPath)
			if err != nil {
				log.Printf("下载图片异常:%s,imgURL:%s", err.Error(), imgURL)
				continue
			}
		} else {
			localPath = imgURL
		}

		ret, err := qiniu.UploadFile(localPath, qNKey, imgToken)
		if err != nil {
			log.Printf("上传图片异常:%s,imgURL:%s", err.Error(), imgURL)
			continue
		}
		newImg := fmt.Sprintf("https://zpic.jiaoliuqu.com/%s", ret.Key)
		content = strings.Replace(content, imgURL, newImg, -1)
		log.Printf("图片替换成功,原始图片:%s,新图片:%s", imgURL, newImg)
	}

	err = ioutil.WriteFile(getWorkFilePath(fileName), []byte(content), 0644)
	if err != nil {
		return errors.New("写入文章异常:" + err.Error())
	}
	return nil
}

// getMDTile 获取title
func getMDTile(fileName string) (string, error) {
	b, err := ioutil.ReadFile(fileName)
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

func timeCompare(timeStr1, timeStr2 string) bool {
	time1, _ := time.ParseInLocation("2006-01-02 15:04:05", timeStr1, time.Local)
	time2, _ := time.ParseInLocation("2006-01-02 15:04:05", timeStr2, time.Local)
	return time1.Unix() > time2.Unix()
}

// getIndexPath 获取索引的路径 eg:xxx/repo/index
func getIndexPath() string {
	return fmt.Sprintf("%s%srepo%sindex", workDir, fileSep, fileSep)
}

// getRepoFilePath
func getRepoFilePath(fileName string) string {
	return fmt.Sprintf("%s%srepo%s%s", workDir, fileSep, fileSep, fileName)
}

// getWorkFilePath  workDir+"/"+remote.FileName
func getWorkFilePath(fileName string) string {
	return fmt.Sprintf("%s%s%s", workDir, fileSep, fileName)
}
