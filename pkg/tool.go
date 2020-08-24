package pkg

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// GetFileMd5 获取文件md5
func GetFileMd5(filePath string) (string, error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	md5Ctx := md5.New()
	md5Ctx.Write(b)
	return hex.EncodeToString(md5Ctx.Sum(nil)), nil
}

// CopyFile 拷贝文件
func CopyFile(dstName, srcName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	defer dst.Close()
	return io.Copy(dst, src)
}

// HttpPost post请求
func HttpPost(url string, form url.Values) ([]byte, error) {
	data := bytes.NewBufferString(form.Encode())
	rsp, err := http.Post(url, "application/x-www-form-urlencoded", data)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// Write2File 生成配置文件
func Write2File(data []byte, pathToFile string) error {
	tmpFile := pathToFile + "_tmp"
	file, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("创建文件%s失败 %s", tmpFile, err)
	}

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("写入内容失败 %s", err)
	}

	file.Close()
	err = os.Rename(tmpFile, pathToFile)
	if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
		return fmt.Errorf("临时文件替换失败 %s %s", pathToFile, err)
	}
	return nil
}

// GetRandomString 生成随机字符串
func GetRandomString(length int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	var result []byte
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < length; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}

// PathExists 判断文件是否存在
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// GetFileSize 获取文件大小
func GetFileSize(filename string) int64 {
	var result int64
	filepath.Walk(filename, func(path string, f os.FileInfo, err error) error {
		result = f.Size()
		return nil
	})
	return result
}

// GetExt 根据文件名获取后缀
func GetExt(src string) string {
	j := strings.LastIndex(src, "?")
	if j != -1 {
		src = src[:j]
	}

	i := strings.LastIndex(src, ".")
	if i != -1 {
		return strings.ToLower(src[i:])
	}
	return ""
}

// GetKey 生成新文件的key
func GetKey() string {
	return GetRandomString(3) + strconv.FormatInt(time.Now().Unix(), 10) + GetRandomString(3)
}

// TimeCompare 时间戳字符串比较
func TimeCompare(timeStr1, timeStr2 string) bool {
	time1, _ := time.ParseInLocation("2006-01-02 15:04:05", timeStr1, time.Local)
	time2, _ := time.ParseInLocation("2006-01-02 15:04:05", timeStr2, time.Local)
	return time1.Unix() > time2.Unix()
}

// respData 返回给客户端的数据
type RespData struct {
	Msg            string      `json:"msg"`
	Data           interface{} `json:"data"`
	ResponseStatus string      `json:"response_status"`
}

// ClientCall http调用
func ClientCall(url string, form url.Values) (interface{}, error) {
	var reply interface{}
	var err error
	for retryNum := 0; retryNum < 3; retryNum++ {
		reply, err = postCall(url, form)
		if err == nil {
			return reply, nil
		}
		time.Sleep(3 * time.Second)
	}
	return nil, err
}

func postCall(url string, form url.Values) (interface{}, error) {
	ret, err := HttpPost(url, form)
	if err != nil {
		return "", err
	}

	r := RespData{}
	err = json.Unmarshal(ret, &r)
	if err != nil {
		return "", errors.New(fmt.Sprintf("err:%s,resp:%s", err.Error(), string(ret)))
	}

	if r.ResponseStatus != "success" {
		return "", errors.New("errMsg:" + r.Msg)
	}
	return r.Data, nil
}

// DownLoadFile 下载文件
func DownLoadFile(URL string, fileName string) error {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	headers := make(http.Header)
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return err
	}
	headers.Set("Accept-Encoding", "gzip, deflate")
	req.Header = headers
	res, err := client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(fileName, body, 0644)
	if err != nil {
		return err
	}

	return nil
}

// VersionCompare 版本比较
func VersionCompare(v1, v2 string) bool {
	v1Arr := strings.Split(v1, ".")
	v2Arr := strings.Split(v2, ".")
	for i := 0; i < len(v1Arr); i++ {
		t1, _ := strconv.Atoi(v1Arr[i])
		t2, _ := strconv.Atoi(v2Arr[i])
		if t1 > t2 {
			return true
		}
	}
	return false
}
