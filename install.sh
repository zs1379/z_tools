#!/bin/bash

mkdir docWorkSpace
cd docWorkSpace

echo "开始下载工具..."
curl https://zpic.jiaoliuqu.com/doc_0.3.5 > doc
echo "下载工具完成"

chmod +x doc
