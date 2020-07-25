# z_tools
用于协助快速上传文章至z项目


#### 1.下载doc.zip文件，解压进入文件夹 
##### 目录结构
1. doc  mac版程序
2. doc.exe Windows程序
3. /repo 本地仓库
4. ignore 忽略的文件列表 (类似git的.ignore文件)

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
title: hello.md
---
```
内容是Markdown格式的，可以根据需要修改title和date

例如编辑完:

```
---
title: helloWorld
date: 2020-07-22 00:08:39
tags:
---

hello world
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
注意: 如果有图片,会被替换成七牛地址(支持jpeg,gif,png,jpg)

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
