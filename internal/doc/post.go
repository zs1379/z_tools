package doc

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"z_tools/pkg"
)

const (
	StatusUserDel = "-2" // 用户删除
	StatusAdmDel  = "-3" // 管理员删除
)

type PostManger struct {
	*Doc
}

func NewPostManger() (*PostManger, error) {
	d, err := NewDoc()
	if err != nil {
		return nil, err
	}

	p := &PostManger{
		Doc: d,
	}
	return p, nil
}

// PostDesc 文章描述
type PostDesc struct {
	FileName   string `json:"file_name"`   // 文件名称
	UpdateTime string `json:"update_time"` // 更新时间
	Md5        string `json:"file_md5"`    // 文件MD5
	Status     string `json:"status"`      // 文件状态 -2:自己删除 -3:管理员删除 其他状态这边暂时用不到
}

// WriteIndex 写入索引
func (p *PostManger) WriteIndex(m map[string]*PostDesc) error {
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

// ReadIndex 读取索引
func (p *PostManger) ReadIndex() (map[string]*PostDesc, error) {
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

// Push 推到远程服务器
func (p *PostManger) Push() {
	localRepoPosts, err := p.ReadIndex()
	if err != nil {
		log.Printf("读取本地仓库异常:%s", err.Error())
		return
	}

	data, err := pkg.ClientCall(fmt.Sprintf("%s/info/client?action=getList&token=%s", p.ServerHost, p.UserToken), url.Values{})
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
	p.WriteIndex(localRepoPosts)

	for _, v := range localRepoPosts {
		r, ok := remotePosts[v.FileName]
		if ok {
			if (r.Md5 == v.Md5 && r.Status == v.Status) || pkg.TimeCompare(r.UpdateTime, v.UpdateTime) {
				continue
			}

			// 删除远程文件
			if v.Status == StatusUserDel && r.Status != StatusUserDel {
				form := url.Values{"filename": {v.FileName}}
				url := fmt.Sprintf("%s/info/client?token=%s&action=delete", p.ServerHost, p.UserToken)
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

		title, category, tag, err := getMDTileCategory(repoObjPath + v.Md5)
		if err != nil {
			log.Printf("读取文章title和分类异常:%s,文章:%s", err.Error(), v.FileName)
			continue
		}

		// pics参数
		pics := p.readImgPath(repoObjPath + v.Md5)
		if len(pics) > 3 {
			pics = pics[0:3]
		}

		content := string(b)
		form := url.Values{
			"filename":   {v.FileName},
			"token":      {p.UserToken},
			"md5":        {v.Md5},
			"content":    {content},
			"title":      {title},
			"category":   {category},
			"updateTime": {v.UpdateTime},
			"pics":       {strings.Join(pics, ",")},
		}
		var tagStr string
		for _, v := range tag {
			tagStr += fmt.Sprintf("&tagNames=%s", v)
		}
		url := fmt.Sprintf("%s/info/client?token=%s&action=add"+tagStr, p.ServerHost, p.UserToken)
		_, err = pkg.ClientCall(url, form)
		if err != nil {
			log.Printf("文章推到远程异常:%s,文章:%s", err.Error(), v.FileName)
			continue
		}

		log.Printf("文章推到远程成功文章:%s", v.FileName)
	}
}

// Pull 拉取远程
func (p *PostManger) Pull() {
	localRepoPosts, err := p.ReadIndex()
	if err != nil {
		log.Printf("读取本地仓库异常:%s", err.Error())
		return
	}

	data, err := pkg.ClientCall(fmt.Sprintf("%s/info/client?action=getList&token=%s", p.ServerHost, p.UserToken), url.Values{})
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
		retData, err := pkg.ClientCall(fmt.Sprintf("%s/info/client?token=%s&action=get", p.ServerHost, p.UserToken), form)
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

	p.WriteIndex(localRepoPosts)
}

// NewDoc 新建文件
func (p *PostManger) NewDoc(fileName string, title string) {
	l, err := p.getCategory()
	fmt.Println()
	fmt.Println(fmt.Sprintf("    选择你文章的分类(单选),目前支持的分类如下:"))

	var str string
	for k, v := range l {
		str += fmt.Sprintf("%d:%s ", k+1, v)
	}

	if runtime.GOOS == "windows" {
		fmt.Printf("    %s\n", str)
	} else {
		fmt.Printf("    \x1b[%dm%s \x1b[0m\n", 36, str)
	}
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

	fmt.Println(fmt.Sprintf("    设置你文章的tag,常用tag如下:"))

	tagList, _ := p.getTagList()
	var tagStr string
	for _, v := range tagList {
		tagStr += fmt.Sprintf("%s ", v)
	}

	if runtime.GOOS == "windows" {
		fmt.Printf("    %s\n", tagStr)
	} else {
		fmt.Printf("    \x1b[%dm%s \x1b[0m\n", 36, tagStr)
	}
	fmt.Println()
	fmt.Print("    请输入tag,多个空格隔开:")

	tagInput := bufio.NewScanner(os.Stdin)

	var tagArr []string
	tagInput.Scan()
	tagArr = strings.Split(strings.TrimSpace(tagInput.Text()), " ")
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

	ok := pkg.PathExists(workPostsPath + fileName)
	if ok {
		log.Printf("文件已经存在,文件:%s", fileName)
		return
	}

	docFormat := `---
title: %s
category: %s
tag: %s
---`

	if title == "" {
		title = fileName[0 : len(fileName)-3]
	}
	docContent := fmt.Sprintf(docFormat, title, category, strings.Join(tagArr, " "))
	err = ioutil.WriteFile(workPostsPath+fileName, []byte(docContent), 0644)
	if err != nil {
		log.Printf("本地创建文章异常:%s,文章:%s", err.Error(), fileName)
		return
	}
	log.Printf("文件创建成功,文件名:%s, 分类:%s, tag:%s", fileName, category, strings.Join(tagArr, " "))
	return
}

// Add 文件工作区加入到本地仓库
func (p *PostManger) Add(fileName string) {
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
			p.doAdd(s.Name())
		}
	} else {
		p.doAdd(fileName)
	}
	return
}

// Rm 删除文件
func (p *PostManger) Rm(fileName string) {
	localRepoPosts, err := p.ReadIndex()
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

	p.WriteIndex(localRepoPosts)
	return
}

// Add 文件工作区加入到本地仓库
func (p *PostManger) doAdd(fileName string) {
	localRepoPosts, err := p.ReadIndex()
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

	ok = pkg.PathExists(workPostsPath + fileName)
	if !ok {
		log.Printf("该文件不存在,文件名:%s", fileName)
		return
	}

	if pkg.GetFileSize(workPostsPath+fileName) > 2*1024*2014 {
		log.Printf("文章大小不支持2M以上,文件名:%s,文章大小:%d", fileName, pkg.GetFileSize(fileName))
		return
	}

	err = p.replaceImg(workPostsPath + fileName)
	if err != nil {
		log.Printf("图片替换异常,err:%s,文件名:%s", err.Error(), fileName)
		return
	}

	_, _, _, err = getMDTileCategory(workPostsPath + fileName)
	if err != nil {
		docFormat := `---
title: 这是标题
category: 文章分类
tag: tag1
---`
		log.Printf("获取文件格式异常,err:%s,文件名:%s", err.Error(), fileName)
		fmt.Println()
		fmt.Println("文档标准格式如下:")
		fmt.Println(docFormat)
		l, _ := p.getCategory()
		fmt.Println()
		fmt.Println(fmt.Sprintf("目前支持的分类如下:"))
		if runtime.GOOS == "windows" {
			fmt.Printf("%s\n", strings.Join(l, " "))
		} else {
			fmt.Printf("\x1b[%dm%s \x1b[0m\n", 36, strings.Join(l, " "))
		}
		fmt.Println()
		tagList, _ := p.getTagList()
		fmt.Println(fmt.Sprintf("目前支持的tag如下,多个空格隔开:"))
		if runtime.GOOS == "windows" {
			fmt.Printf("%s\n", strings.Join(tagList, " "))
		} else {
			fmt.Printf("\x1b[%dm%s \x1b[0m\n", 36, strings.Join(tagList, " "))
		}
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
	p.WriteIndex(localRepoPosts)

	return
}

// Checkout 从本地repo迁出到工作区
func (p *PostManger) Checkout(fileName string) {
	localRepoPosts, err := p.ReadIndex()
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
func (p *PostManger) Status() {
	localRepoPosts, err := p.ReadIndex()
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
		b := pkg.PathExists(workPostsPath + v.FileName)
		if !b && v.Status != "-2" && v.Status != "-3" {
			log.Printf("文件被删除:%s", v.FileName)
		}
	}
}

// getCategory 获取文章分类
func (p *PostManger) getCategory() ([]string, error) {
	u := p.ServerHost + "/info/client?action=getCategory"
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

// getTagList 获取常用的tag
func (p *PostManger) getTagList() ([]string, error) {
	l := []string{
		"java", "php", "go", "node.js", "oc", "spring", "后端",
		"小程序", "ios", "android", "kotlin", "flutter", "xcode",
		"js", "vue", "html", "css", "typescript", "html5",
		"mysql", "redis", "sql", "json", "数据库", "nosql",
		"linux", "nginx", "docker", "k8s",
	}

	return l, nil
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
	ImgExtList := []string{".jpeg", ".gif", ".png", ".jpg", ".webp"}
	for _, v := range ImgExtList {
		if v == ext {
			return true
		}
	}
	return false
}

// getMDTileCategory 获取title和分类
func getMDTileCategory(filePath string) (string, string, []string, error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", "", []string{}, err
	}

	r := bufio.NewReader(strings.NewReader(string(b)))
	line, _, err := r.ReadLine()
	if err != nil {
		return "", "", []string{}, errors.New("第一行读取错误:" + err.Error())
	}

	if string(line) != "---" {
		return "", "", []string{}, errors.New("格式错误,文档第一行需---开头")
	}

	// 获取title
	line, _, err = r.ReadLine()
	if err != nil {
		return "", "", []string{}, errors.New("读取title错误:" + err.Error())
	}
	title, err := getField(string(line), "title:")
	if err != nil {
		return "", "", []string{}, err
	}

	// 获取分类
	line, _, err = r.ReadLine()
	if err != nil {
		return "", "", []string{}, errors.New("读取category错误:" + err.Error())
	}
	category, err := getField(string(line), "category:")
	if err != nil {
		return "", "", []string{}, err
	}

	// 获取tag
	line, _, err = r.ReadLine()
	if err != nil {
		return "", "", []string{}, errors.New("读取tag错误:" + err.Error())
	}
	tag, err := getField(string(line), "tag:")
	if err != nil {
		return title, category, []string{}, nil
	}
	tagArr := TrimStringArr(strings.Split(tag, " "))
	return title, category, tagArr, nil
}

func getField(str string, field string) (string, error) {
	if len(str) <= len(field) {
		return "", fmt.Errorf("格式错误,文档需%s开头", field)
	}
	if str[:len(field)] != field {
		return "", fmt.Errorf("格式错误,文档需%s开头", field)
	}
	return strings.TrimSpace(str[len(field):]), nil
}

func TrimStringArr(arr []string) []string {
	var newArr []string
	for _, v := range arr {
		if v != "" {
			newArr = append(newArr, v)
		}
	}
	return newArr
}
