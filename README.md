# z_tools
用于协助快速上传文章至z项目

#### 1.安装doc 
1) sh -c "$(curl -fsSL https://zpic.jiaoliuqu.com/install.sh)"
2) cd ~/docWorkSpace

#### 2.初始化 

```
./doc init xxx
```
xxx 为用户token,初始化完成会自动生成下面目录
1. ./.repo 本地仓库 (不要去动)
2. ./img 图片引用目录
3. ./posts 工作区文章目录

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

样例:

```
---
title: 这是第一篇文章
---

hello world

图片样例 (注意引用路径,只支持工作区img目录下的图片)
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

#### 5.提交文章到本地仓库
```
./doc add hello.md
```
注意: 
1. doc add . 可以添加全部文章导本地仓库
2. 引用本地图片add的时候会被替换成七牛地址

#### 6.还原工作区文章 
```
./doc checkout hello.md
```

#### 7.删除本地仓库文章
```
./doc rm hello.md
```

#### 8.本地仓库提交到远程
```
./doc push
```
注意: 如果远程版本比本地版本新，则不会更新远程

#### 9.拉取远程仓库
```
./doc pull
```

#### 10.升级版本
```
./doc update
```

### 注意文章名称一旦创建,就不允许修改
