#!/bin/bash

mkdir tool
cd tool
echo "开始下载工具..."
curl https://zpic.jiaoliuqu.com/doc_0.0.11 > doc
echo "下载工具完成"

chmod +x doc
