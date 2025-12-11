// Copyright 2025 Boyuan-IT-Club
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lib

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Boyuan-IT-Club/go-kit/logs"
	"github.com/cloudwego/hertz/pkg/common/json"
	"github.com/google/uuid"
)

// JSONF 将对象序列化成json格式字符串
func JSONF(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		logs.Error("JSONF fail, v=%v, err=%v", v, err)
	}
	return string(data)
}

// GzipCompress gzip压缩
func GzipCompress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, _ = w.Write(data)
	_ = w.Close()
	return b.Bytes(), nil
}

// GzipDecompress gzip解压
func GzipDecompress(src []byte) ([]byte, error) {
	// 1. 空数据检查
	if len(src) == 0 {
		return nil, nil
	}

	// 2. 创建GZIP读取器
	r, err := gzip.NewReader(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("创建解压器失败: %w", err)
	}
	defer func() { _ = r.Close() }()

	// 3. 读取解压数据
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return nil, fmt.Errorf("解压数据读取失败: %w", err)
	}

	// 4. 返回解压结果
	return buf.Bytes(), nil
}

// IntToBytes 将整数变成字节数组
func IntToBytes(n int) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(n))
	return b
}

// BytesToInt 将字节数组变成整数
func BytesToInt(data []byte) (int, error) {
	if len(data) != 4 || data == nil {
		return 0, fmt.Errorf("BytesToInt err")
	}
	return int(binary.BigEndian.Uint32(data)), nil
}

// BuildBytes 将传入的byte拼接并返回一个新的bytes数组
func BuildBytes(data ...[]byte) []byte {
	var b bytes.Buffer
	for _, d := range data {
		b.Write(d)
	}
	return b.Bytes()
}

// NNCtx 非空获取一个BackgroundCtx否则本身
func NNCtx(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// NewUID 常见一个包含时间的UUID
func NewUID() string {
	timestamp := time.Now().Format("060102-150405") // 格式化为YYMMDD-HHMMSS
	uuidPart := uuid.New().String()[:4]             // 取UUID前4位
	return fmt.Sprintf("%s-%s", timestamp, uuidPart)
}
