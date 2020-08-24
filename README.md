# z_tools
用于协助快速上传文章至z项目


#### 1.下载doc 
##### 目录结构
1. doc  mac版程序
2. doc.exe Windows程序
3. ./.repo 本地仓库
4. ./img 图片路径
4. ./posts 文章路径

#### 2.初始化 

```
./doc init xxx
```
xxx 为用户token

#### 3.创建一个hello.md的文章

```
./doc new hello.md
```
hello.md将自动生成到了工作目录下,打开hello.md看下：

```
---
title: hello
---
```
内容是Markdown格式的，前三行自动生成格式不要修改,可以根据需要修改title内容

例如编辑完:

```
---
title: helloWorld
---

hello world

支持图片,注意路径
![image] (../img/1.png)
```

#### 4.查看变更
```
./doc status
```
控制台输出如下,代表新增了一个文章
```
2020/07/22 00:14:02 存在新文件:hello.md
```

#### 5.提交到本地仓库
```
./doc add hello.md
```
注意: 
1. doc add . 可以添加全部文件导本地仓库
2. 图片仅支持img目录下的路径, eg:![image] (../img/1.png), add的时候会被替换成七牛地址

#### 6.还原工作区文件 
```
./doc checkout hello.md
```

#### 7.本地仓库提交到远程
```
./doc push
```
注意: 如果远程版本比本地版本新，则不会更新远程

#### 8.拉取远程仓库
```
./doc pull
```

#### 9.升级版本
```
./doc update
```
