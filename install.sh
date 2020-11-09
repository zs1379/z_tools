#!/bin/bash

mkdir docWorkSpace
cd docWorkSpace

echo "开始下载工具..."
curl https://zpic.xiaoy.name/doc_0.4.4 > doc
echo "下载工具完成"

chmod +x doc
