package gorequest

import "testing"

func TestGet(t *testing.T) {
	req := New()
	resp, bs, err := req.Get("https://www.baidu.com/s").Param("wd", "github").End()
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	t.Log(resp.StatusCode)
	t.Log(len(bs))
}

func TestEndStruct(t *testing.T) {
	type Resp struct {
		CurrentUserUrl string `json:"current_user_url"`
	}
	entity := &Resp{}
	req := New()
	resp, err := req.Get("https://api.github.com/").EndStruct(entity)
	if err != nil {
		t.Fatal(err.Error())
		return
	}
	if entity.CurrentUserUrl == "" {
		t.Fatal("api url should be not nil")
	}
	t.Log(resp.StatusCode)
	t.Log(entity.CurrentUserUrl)
}
