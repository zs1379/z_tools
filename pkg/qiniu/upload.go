package qiniu

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
)

type ImgInfo struct {
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
	Ext    string `json:"ext"`
}

// UploadFile 上传文件至七牛云
func UploadFile(fileName string, qnKey string, token string) (*ImgInfo, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fileInput, err := w.CreateFormFile("file", fileName)
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(fileInput, f); err != nil {
		return nil, err
	}
	// 添加key字段
	keyInput, err := w.CreateFormField("key")
	if err != nil {
		return nil, err
	}

	if _, err = keyInput.Write([]byte(qnKey)); err != nil {
		return nil, err
	}
	tokenInput, err := w.CreateFormField("token")
	if err != nil {
		return nil, err
	}
	_, err = tokenInput.Write([]byte(token))
	if err != nil {
		return nil, err
	}
	w.Close()

	postURL := fmt.Sprintf("http://up-z1.qiniup.com")
	req, _ := http.NewRequest("POST", postURL, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, errors.New("发起七牛云资源直传请求出错 err:" + err.Error())
	}
	defer res.Body.Close()
	result, err := ioutil.ReadAll(res.Body)
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("七牛云资源直传返回状态码错误: code:%d, body:%s", res.StatusCode, result)
	}
	if err != nil {
		return nil, err
	}

	var i ImgInfo
	json.Unmarshal(result, &i)
	return &i, nil
}
