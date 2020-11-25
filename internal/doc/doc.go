package doc

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"z_tools/pkg"
	"z_tools/pkg/qiniu"
)

var (
	Version = "0.4.7"
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

type Doc struct {
	UserToken  string // 用户token
	ServerHost string // 服务器域名
}

func NewDoc() (*Doc, error) {
	d := &Doc{}
	err := d.Init()
	if err != nil {
		return nil, err
	}
	d.autoUpdate()
	return d, nil
}

func (d *Doc) Init() error {
	err := os.MkdirAll(workPostsPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("创建工作区目录异常:%s", err.Error())
	}
	err = os.MkdirAll(knWorkPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("创建知识点工作区目录异常:%s", err.Error())
	}
	err = os.MkdirAll(imgPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("创建img目录异常:%s", err.Error())
	}
	err = os.MkdirAll(repoObjPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("创建repo/objects目录异常:%s", err.Error())
	}

	env := d.ReadEnv()
	if env == "test" {
		d.ServerHost = "http://10.10.80.222:8000/2016-08-15/proxy"
	} else {
		d.ServerHost = "http://z1.xiaoy.name"
	}

	d.UserToken = d.ReadToken()

	// 用户token校验
	if len(os.Args) >= 2 && os.Args[1] != "init" && d.UserToken == "" {
		return fmt.Errorf("用户token为空,请到小程序我的TAB页复制,并执行./doc init 用户token 进行初始化~")
	}
	return nil
}

// ReadToken 读取用户token
func (d *Doc) ReadToken() string {
	b, _ := ioutil.ReadFile(tokenPath)
	return strings.TrimSpace(string(b))
}

// ReadEnv 读取环境变量
func (d *Doc) ReadEnv() string {
	b, _ := ioutil.ReadFile(envPath)
	return string(b)
}

// getUploadToken 获取七牛token
func (d *Doc) getUploadToken(key string) (string, error) {
	u := d.ServerHost + "/basic/getPicToken?token=" + d.UserToken
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

// InitDoc 初始化
func (d *Doc) InitDoc(token string, env string) {
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
	d.WriteUpdateTime()
	log.Printf("初始化成功")
}

// ReadUpdateTime 读取安装时间
func (d *Doc) ReadUpdateTime() string {
	b, _ := ioutil.ReadFile(updatePath)
	return string(b)
}

// WriteUpdateTime 刷新安装时间
func (d *Doc) WriteUpdateTime() {
	ioutil.WriteFile(updatePath, []byte(fmt.Sprintf("%d", time.Now().Unix())), 0644)
	return
}

// autoUpdate 自动升级
func (d *Doc) autoUpdate() {
	lastUpdate, _ := strconv.Atoi(d.ReadUpdateTime())
	if int(time.Now().Unix())-lastUpdate < 86400 {
		return
	}

	d.WriteUpdateTime()
	d.Update(true)
}

// getRemoteVersion 获取服务器版本号
func (d *Doc) getRemoteVersion() (string, error) {
	data, err := pkg.ClientCall(d.ServerHost+"/info/client?action=version&token="+d.UserToken, url.Values{})
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

// Update 版本升级
func (d *Doc) Update(auto bool) {
	remoteV, err := d.getRemoteVersion()
	if err != nil {
		log.Printf("获取版本号异常:%s", err.Error())
		return
	}

	// 更新时间
	d.WriteUpdateTime()

	// 判断是否需要升级版本
	if !pkg.VersionCompare(remoteV, Version) {
		if !auto {
			log.Printf("当前已经是最新版本:%s", Version)
		}
		return
	}
	defer os.Exit(0)

	// 自动检测的给用户选择
	if auto {
		fmt.Println()
		if runtime.GOOS == "windows" {
			fmt.Printf("    %s\n", "检测到有新版本,按n取消,按其余任意键升级~")
		} else {
			fmt.Printf("    \x1b[%dm%s \x1b[0m\n", 36, "检测到有新版本,按n取消,按其余任意键升级~")
		}
		fmt.Printf("    请输入:")
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		v := strings.TrimSpace(input.Text())
		if strings.ToLower(v) == "n" {
			fmt.Println("    取消升级~")
			return
		}
	}

	log.Printf("当前版本:%s,远程版本:%s,升级中....", Version, remoteV)

	oldFile := "doc"
	newFile := fmt.Sprintf("doc_%s", remoteV)

	sysType := runtime.GOOS
	if sysType == "windows" {
		oldFile += ".exe"
		newFile += ".exe"
	}

	err = pkg.DownLoadFile(fmt.Sprintf("https://zpic.xiaoy.name/%s", newFile), newFile)
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
func (d *Doc) Update2Ser(version string) {
	fileNameMac := "doc_" + version
	token, err := d.getUploadToken(fileNameMac)
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
	token, err = d.getUploadToken(fileNameExe)
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
	_, err = pkg.ClientCall(d.ServerHost+"/info/client?action=setVersion&token="+d.UserToken, form)
	if err != nil {
		log.Printf("版本设置失败:%s", err.Error())
		return
	}

	log.Printf("版本设置成功当前服务器版本号:%s", version)
}

// UpdateInstallShell 更新安装脚本
func (d *Doc) UpdateInstallShell() {
	installMac := "install.sh"

	token, err := d.getUploadToken(installMac)
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

// replaceImg 本地图片替换成七牛图片
func (d *Doc) replaceImg(filePath string) error {
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
	imgToken, err = d.getUploadToken("")
	if err != nil {
		return errors.New("拉取七牛文件上传凭证异常:" + err.Error())
	}

	change := false
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

		if strings.Contains(imgURL, "jiaoliuqu.com") || strings.Contains(imgURL, "xiaoy.name") {
			continue
		}

		ret, err := qiniu.UploadFile(imgURL, pkg.GetKey()+ext, imgToken)
		if err != nil {
			log.Printf("上传图片异常:%s,imgURL:%s", err.Error(), imgURL)
			continue
		}

		change = true
		newImg := fmt.Sprintf("https://zpic.xiaoy.name/%s", ret.Key)
		content = strings.Replace(content, imgURL, newImg, -1)
		log.Printf("图片替换成功,原始图片:%s,新图片:%s", imgURL, newImg)
	}

	if change {
		err = ioutil.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			return errors.New("写入文章异常:" + err.Error())
		}
	}
	return nil
}

// readImgPath 读取文件里面的图片地址
func (d *Doc) readImgPath(filePath string) []string {
	var imgs []string
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return imgs
	}
	if len(b) == 0 {
		return imgs
	}

	content := string(b)
	re, _ := regexp.Compile(`\!\[.*?\]\((.*?)\)`)
	c := re.FindAllSubmatch([]byte(content), -1)
	if len(c) == 0 {
		return imgs
	}

	for _, v := range c {
		if len(v) < 2 {
			continue
		}

		imgURL := string(v[1])
		if strings.Contains(imgURL, "jiaoliuqu.com") || strings.Contains(imgURL, "xiaoy.name") {
			imgURL = strings.Replace(imgURL, "https://zpic.xiaoy.name/", "", 1)
			imgURL = strings.Replace(imgURL, "https://zpic.jiaoliuqu.com/", "", 1)
			imgs = append(imgs, imgURL)
		}
	}

	return imgs
}
