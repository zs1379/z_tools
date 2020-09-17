package main

import (
	"log"
	"os"
	"sort"

	"z_tools/internal/doc"

	"github.com/urfave/cli/v2"
)

func main() {
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"V"},
		Usage:   "print only the version",
	}

	app := &cli.App{
		Version: doc.Version,
		Usage:   "文章上传助手",
		Commands: []*cli.Command{
			{
				Name:        "init",
				Usage:       "初始化环境",
				Description: "1. doc init test test为用户的token",
				ArgsUsage:   "[token]",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入token,命令行格式./doc init 用户token")
						return nil
					}
					env := "online"
					if c.NArg() >= 2 {
						env = c.Args().Get(1)
					}
					doc.InitDoc(c.Args().Get(0), env)
					return nil
				},
			},
			{
				Name:        "new",
				Usage:       "新建文章",
				Description: "1. doc new test 本地自动生成一篇test.md的空文档",
				ArgsUsage:   "[文件名]",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入文件名,命令行格式./doc new xx")
						return nil
					}
					doc.NewDoc(c.Args().Get(0))
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
						log.Printf("请输入文件名,命令行格式./doc add xx.md")
						return nil
					}
					doc.Add(c.Args().Get(0))
					return nil
				},
			},
			{
				Name:        "pull",
				Usage:       "拉取文章列表",
				Description: "1. doc pull 从服务器拉取最新文章列表到本地",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					doc.Pull()
					return nil
				},
			},
			{
				Name:        "push",
				Usage:       "提交到服务器",
				Description: "1. doc push 把本地仓库变更提交到服务器",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					doc.Push()
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
						log.Printf("请输入文件名,命令行格式./doc rm xx.md")
						return nil
					}
					doc.Rm(c.Args().Get(0))
					return nil
				},
			},
			{
				Name:        "status",
				Usage:       "查看文件变更",
				Description: "1. doc status 比对本地仓库和工作区的文件变更",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					doc.Status()
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
						log.Printf("请输入文件名,命令行格式./doc checkout xx.md 支持点号")
						return nil
					}
					doc.Checkout(c.Args().Get(0))
					return nil
				},
			},
			{
				Name:        "update",
				Usage:       "版本升级",
				Description: "1. doc update 升级程序版本",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					doc.Update(false)
					return nil
				},
			},
			{
				Name:        "updateToSer",
				Usage:       "版本更新到服务器-[需管理员权限]",
				Description: "1. doc updateToSer 0.0.3 将0.0.3版本程序上传到七牛",
				ArgsUsage:   "[版本号]",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入版本号")
						return nil
					}
					doc.Update2Ser(c.Args().Get(0))
					return nil
				},
			},
			{
				Name:        "updateInstall",
				Usage:       "更新install文件-[需管理员权限]",
				Description: "1. doc updateInstall",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					doc.UpdateInstallShell()
					return nil
				},
			},
			{
				Name:        "kpull",
				Usage:       "拉取知识点",
				Description: "1. doc kpull xx 拉取xx的知识点",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入知识点")
						return nil
					}
					doc.KPull(c.Args().Get(0))
					return nil
				},
			},
			{
				Name:        "kadd",
				Usage:       "提交知识点更新到本地",
				Description: "1. doc kadd xx 要提交的知识点",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入知识点")
						return nil
					}

					if c.NArg() < 2 {
						log.Printf("请输入修改日志")
						return nil
					}
					doc.KAdd(c.Args().Get(0), c.Args().Get(1))
					return nil
				},
			},
			{
				Name:        "kpush",
				Usage:       "提交知识点更新到服务器",
				Description: "1. doc kpush xx",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入知识点")
						return nil
					}
					doc.KPush(c.Args().Get(0))
					return nil
				},
			},
			{
				Name:        "knew",
				Usage:       "新建知识点",
				Description: "1. doc knew xx 要新建的知识点",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入知识点")
						return nil
					}
					doc.KNew(c.Args().Get(0))
					return nil
				},
			},
			{
				Name:        " krel",
				Usage:       "给知识点创建别名",
				Description: "1. doc  krel xx 知识点 别名",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入知识点")
						return nil
					}
					if c.NArg() < 2 {
						log.Printf("请输入别名")
						return nil
					}
					doc.Krel(c.Args().Get(0), c.Args().Get(1))
					return nil
				},
			},
			{
				Name:        "kstatus",
				Usage:       "查看文件变更",
				Description: "1. doc kstatus 比对本地仓库和工作区的知识点变更",
				ArgsUsage:   " ",
				Action: func(c *cli.Context) error {
					doc.StatusKn()
					return nil
				},
			},
			{
				Name:        "kcheckout",
				Usage:       "恢复本地仓库的指定文件到工作区",
				Description: "1. doc kcheckout hash 从本地仓库恢复hash到工作区\n\r   2. doc kcheckout . 恢复本地仓库的全部文件到工作区",
				ArgsUsage:   "[文件名]",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						log.Printf("请输入知识点,命令行格式./doc kcheckout xx 支持点号")
						return nil
					}
					doc.CheckoutKN(c.Args().Get(0))
					return nil
				},
			},
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	doc.ReadDocEnv()

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
