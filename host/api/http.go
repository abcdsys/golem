//go:build !lib

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"sync"

	"github.com/sbgayhub/golem/host/config"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type client struct {
	baseUrl string
	token   string
	client  *http.Client
}

// Response 统一 API 响应格式
type Response struct {
	Code    int64           `json:"code"`
	Data    json.RawMessage `json:"data,omitempty"`
	Message string          `json:"message,omitempty"`
}

type Request struct {
	client *client
	method string
	uri    string
	query  url.Values
	body   io.Reader
	part   *multipart.Writer
}

// GetHttp 获取 HTTP 客户端单例
var GetHttp = sync.OnceValue(func() *client {
	cfg := config.Get()
	return &client{
		baseUrl: cfg.URL,
		token:   cfg.Token,
		client:  http.DefaultClient,
	}
})

func (c *client) NewRequest(method string, uri string) *Request {
	return &Request{client: c, method: method, uri: uri}
}

func (c *client) Get(uri string) *Request { return c.NewRequest(http.MethodGet, uri) }

func (c *client) Post(uri string) *Request { return c.NewRequest(http.MethodPost, uri) }

func (c *client) Put(uri string) *Request { return c.NewRequest(http.MethodPut, uri) }

func (c *client) Delete(uri string) *Request { return c.NewRequest(http.MethodDelete, uri) }

func (r *Request) Query(args ...any) *Request {
	if r.query == nil {
		r.query = make(url.Values, len(args)/2)
	}
	for i := 0; i+1 < len(args); i += 2 {
		r.query.Set(fmt.Sprint(args[i]), fmt.Sprint(args[i+1]))
	}
	return r
}

func (r *Request) Path(path string) *Request {
	r.uri = fmt.Sprintf(r.uri, path)
	return r
}

func (r *Request) Body(body any) *Request {
	if marshal, err := json.Marshal(body); err != nil {
		slog.Error("[http] 序列化body出现错误", "err", err)
		r.body = bytes.NewReader([]byte("{}"))
		return r
	} else {
		r.body = bytes.NewReader(marshal)
		return r
	}
}

func (r *Request) Multipart(files map[string][]byte, fields map[string]string) *Request {
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	defer func() {
		if err := writer.Close(); err != nil {
			slog.Error("[http] 关闭表单写入器出现错误", "err", err)
		}
	}()

	for name, file := range files {
		if part, err := writer.CreateFormFile(name, name); err != nil {
			slog.Error("[http] 创建文件表单出现错误", "err", err)
			return r
		} else {
			if _, err := part.Write(file); err != nil {
				slog.Error("[http] 写入文件数据出现错误", "err", err)
				return r
			}
		}
	}
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			slog.Error("[http] 写入表单数据出现错误", "err", err)
			return r
		}
	}
	r.part = writer
	r.body = &buffer
	return r
}

func (r *Request) Do() ([]byte, error) {
	// 构建请求地址
	path := r.client.baseUrl + r.uri
	if r.query != nil {
		path = path + "?" + r.query.Encode()
	}

	// 创建请求
	request, err := http.NewRequest(r.method, path, r.body)
	if err != nil {
		return nil, fmt.Errorf("[http] 创建http请求失败: %w", err)
	}

	// 设置请求头
	if r.client.token != "" {
		request.Header.Set("Authorization", "Bearer "+r.client.token)
	}
	if r.part != nil {
		request.Header.Set("Content-Type", r.part.FormDataContentType())
	} else {
		request.Header.Set("Content-Type", "application/json")
	}

	// 发送请求
	resp, err := r.client.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("[http] 发送http请求失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 读取响应
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("[http] 读取http响应失败: %w", err)
	}

	var response Response
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("[http] 响应状态错误: %d, %s", resp.StatusCode, string(all))
	}
	if err := json.Unmarshal(all, &response); err != nil {
		return all, nil
	}
	if response.Code != 0 {
		return nil, fmt.Errorf("[http] api返回错误, code: %d, message: %s", response.Code, response.Message)
	}

	return response.Data, nil
}

func (r *Request) DoProto(result proto.Message) error {
	if data, err := r.Do(); err != nil {
		return err
	} else {
		if err := protojson.Unmarshal(data, result); err != nil {
			return fmt.Errorf("[http] 反序列化出现错误: %w", err)
		}
	}
	return nil
}

func (r *Request) DoJson(result any) error {
	if data, err := r.Do(); err != nil {
		return err
	} else {
		if err := json.Unmarshal(data, result); err != nil {
			return fmt.Errorf("[http] 反序列化出现错误: %w", err)
		}
	}
	return nil
}
