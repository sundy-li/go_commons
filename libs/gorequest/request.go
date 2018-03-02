package gorequest

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type GoRequest struct {
	queryData url.Values
	url       string
	method    string
	header    http.Header
	body      []byte
	client    http.Client
}

func New() *GoRequest {
	return &GoRequest{
		header:    make(http.Header),
		queryData: make(url.Values),
	}
}

func (req *GoRequest) Post(url string) *GoRequest {
	req.url = url
	req.method = "POST"
	return req
}

func (req *GoRequest) Get(url string) *GoRequest {
	req.url = url
	req.method = "GET"
	return req
}

func (req *GoRequest) Delete(url string) *GoRequest {
	req.url = url
	req.method = "DELETE"
	return req
}

func (req *GoRequest) Put(url string) *GoRequest {
	req.url = url
	req.method = "PUT"
	return req
}

func (req *GoRequest) Param(key, value string) *GoRequest {
	req.queryData.Add(key, value)
	return req
}

func (req *GoRequest) SetHeader(k, v string) *GoRequest {
	req.header.Add(k, v)
	return req
}
func (req *GoRequest) SetTimeout(timeout time.Duration) *GoRequest {
	req.client.Timeout = timeout
	return req
}
func (req *GoRequest) SendStr(body string) *GoRequest {
	return req.SendBytes([]byte(body))
}

func (req *GoRequest) SendBytes(bs []byte) *GoRequest {
	req.body = bs
	return req
}

func (req *GoRequest) End() (resp *http.Response, bs []byte, err error) {
	request, err := http.NewRequest(req.method, req.url, bytes.NewReader(req.body))
	if err != nil {
		return
	}
	q := request.URL.Query()
	for k, v := range req.queryData {
		for _, vv := range v {
			q.Add(k, vv)
		}
	}
	request.URL.RawQuery = q.Encode()
	request.Header = req.header
	resp, err = req.client.Do(request)
	if err != nil {
		return
	}
	bs, err = ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	return
}

func (req *GoRequest) EndStruct(entity interface{}) (resp *http.Response, err error) {
	var bs []byte
	resp, bs, err = req.End()
	if err != nil {
		return
	}
	err = json.Unmarshal(bs, entity)
	return
}
