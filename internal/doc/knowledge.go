package doc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"z_tools/pkg"
)

// KnowledgeDesc 知识点描述
type KnowledgeDesc struct {
	KName      string `json:"kName"`       // 知识点名称
	UpdateTime string `json:"update_time"` // 更新时间
	Md5        string `json:"file_md5"`    // 文件MD5
	Changelog  string `json:"changelog"`   // 修改日志
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

// getKNLocalVersion 获取知识点本地版本号
func getKNLocalVersion(kName string) string {
	versionPath := fmt.Sprintf("%s%s/version", knWorkPath, kName)
	b, _ := ioutil.ReadFile(versionPath)
	return string(b)
}

// KPull 远程拉取知识点
func KPull(kName string) {
	data, err := pkg.ClientCall(fmt.Sprintf("%s/info/client?action=kget&token=%s&kname=%s", ServerHost, UserToken, kName), url.Values{})
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

	err = os.MkdirAll(knWorkPath+kName, os.ModePerm)
	if err != nil {
		log.Printf("创建知识点工作区目录异常:%s", err.Error())
		return
	}

	// 刷历史版本文件到本地
	var maxV int
	for _, v := range list {
		m, _ := v.(map[string]interface{})
		ver, _ := m["version"].(string)
		v, _ := strconv.Atoi(ver)
		if v > maxV {
			maxV = v
		}
		content, _ := m["content"].(string)
		err = pkg.WriteFile(fmt.Sprintf("%s%s/%d/%s.md", knWorkPath, kName, v, kName), content)
		if err != nil {
			log.Printf("写入知识点文件异常:%s", err.Error())
			return
		}
	}

	knFilePath := fmt.Sprintf("%s/%s.md", knWorkPath, kName)
	ok = pkg.PathExists(knFilePath)
	if ok {
		localV, _ := strconv.Atoi(getKNLocalVersion(kName))
		nowV, _ := strconv.Atoi(nowVersion)
		// 如果远程版本大于本地版本
		if nowV > localV {
			newVPath := fmt.Sprintf("%s%s/%d/%s.md", knWorkPath, kName, nowV, kName)
			ok = pkg.PathExists(newVPath)
			if ok {
				os.Rename(knFilePath, fmt.Sprintf("%s/%s-old.md", knWorkPath, kName))
				pkg.CopyFile(knFilePath, newVPath)
				log.Printf("版本冲突,本地文件被重命名为:%s-old.md,版本号:%s", kName, nowVersion)
			} else {
				// 如果远程版本最新版未过审--概率很低
				log.Printf("拉取远程知识点成功:%s,最新版本未通过审核,本地不变更,版本号:%s", kName, nowVersion)
			}
		} else {
			log.Printf("拉取远程知识点成功:%s,本地无变更,版本号:%s", kName, nowVersion)
		}
	} else {
		log.Printf("拉取远程知识点成功:%s,版本号:%s", kName, nowVersion)
		pkg.CopyFile(knFilePath, fmt.Sprintf("%s%s/%d/%s.md", knWorkPath, kName, maxV, kName))
	}

	// 更新本地版本号
	err = pkg.WriteFile(fmt.Sprintf("%s%s/version", knWorkPath, kName), nowVersion)
	if err != nil {
		log.Printf("写入知识点版本号异常:%s", err.Error())
		return
	}
}

// KPush 知识点推到远程服务器
func KPush(kName string) {
	localKNs, err := ReadKIndex()
	if err != nil {
		log.Printf("读取本地仓库异常:%s", err.Error())
		return
	}

	knDes, ok := localKNs[kName]
	if !ok {
		log.Printf("本地仓库该知识点不存在:%s", kName)
		return
	}

	b, err := ioutil.ReadFile(repoObjPath + knDes.Md5)
	if err != nil {
		log.Printf("读取知识点异常:%s,知识点:%s", err.Error(), knDes.KName)
		return
	}

	localV := getKNLocalVersion(knDes.KName)
	if localV == "" {
		log.Printf("提交异常,知识点本地版本号不存在:%s", kName)
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
			KPull(kName)
			return
		}

		log.Printf("知识点推到远程异常:%s,知识点:%s", err.Error(), knDes.KName)
		return
	}

	log.Printf("知识点推到远程成功:%s", knDes.KName)
}

func KNew(kName string) {
	form := url.Values{
		"token": {UserToken},
		"kname": {kName},
	}

	url := fmt.Sprintf("%s/info/client?token=%s&action=knew", ServerHost, UserToken)
	_, err := pkg.ClientCall(url, form)
	if err != nil {
		log.Printf("创建知识点异常:%s,知识点:%s", err.Error(), kName)
		return
	}
}

func Krel(kName string, rel string) {
	form := url.Values{
		"token":     {UserToken},
		"kname":     {kName},
		"like_name": {rel},
	}

	url := fmt.Sprintf("%s/info/client?token=%s&action=krel", ServerHost, UserToken)
	_, err := pkg.ClientCall(url, form)
	if err != nil {
		log.Printf("创建知识点别名异常:%s,知识点:%s", err.Error(), kName)
		return
	}
}

// KAdd 文件工作区加入到本地仓库
func KAdd(kName string, changelog string) {
	localKN, err := ReadKIndex()
	if err != nil {
		log.Printf("读取本地仓库异常:%s", err.Error())
		return
	}

	knPath := fmt.Sprintf("%s%s.md", knWorkPath, kName)
	ok := pkg.PathExists(knPath)
	if !ok {
		log.Printf("该知识点文件不存在,知识点:%s", kName)
		return
	}

	err = replaceImg(knPath)
	if err != nil {
		log.Printf("图片替换异常,err:%s,文件名:%s", err.Error(), kName)
		return
	}

	fileMd5, err := pkg.GetFileMd5(knPath)
	if err != nil {
		log.Printf("获取文件md5异常,err:%s,知识点:%s", err.Error(), kName)
		return
	}

	knDes := localKN[kName]
	if knDes == nil {
		knDes = &KnowledgeDesc{KName: kName}
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

// StatusKn 本地工作区和本地repo的差异
func StatusKn() {
	localKNs, err := ReadKIndex()
	if err != nil {
		log.Printf("读取本地仓库异常:%s", err.Error())
		return
	}

	files, err := ioutil.ReadDir(knWorkPath)
	for _, s := range files {
		if s.IsDir() || inIgnoreList(s.Name()) {
			continue
		}
		if !strings.Contains(s.Name(), ".md") {
			continue
		}

		kName := s.Name()[:strings.Index(s.Name(), ".")]
		v, ok := localKNs[kName]
		if !ok {
			log.Printf("存在新知识点:%s", kName)
			continue
		}

		md5, err := pkg.GetFileMd5(knWorkPath + s.Name())
		if err != nil {
			log.Printf("获取md5异常:%s,文件名:%s", err.Error(), s.Name())
			continue
		}
		if md5 != v.Md5 {
			log.Printf("存在变更知识点:%s", v.KName)
		}
	}
}

// CheckoutKN 签出文件
func CheckoutKN(kName string) {
	localKNs, err := ReadKIndex()
	if err != nil {
		log.Printf("读取本地仓库异常:%s", err.Error())
		return
	}

	if kName == "." {
		for _, v := range localKNs {
			_, err = pkg.CopyFile(knWorkPath+v.KName+".md", repoObjPath+v.Md5)
			if err != nil {
				log.Printf("拷贝文件异常:%s,知识点:%s", err.Error(), v.KName)
				return
			}
		}
	} else {
		v, ok := localKNs[kName]
		if !ok {
			log.Printf("未匹配到任何文件,知识点:%s", kName)
			return
		}

		_, err = pkg.CopyFile(knWorkPath+v.KName+".md", repoObjPath+v.Md5)
		if err != nil {
			log.Printf("拷贝文件异常:%s,文件名:%s", err.Error(), v.KName)
			return
		}
	}
}
