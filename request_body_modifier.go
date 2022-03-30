package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/url"

	"github.com/joncalhoun/qson"
	"github.com/luraproject/lura/v2/proxy"
)

func main() {}

var ModifierRegisterer = registerer("requestBody.Modifier")

type registerer string

func (r registerer) RegisterModifiers(f func(
	name string,
	modifierFactory func(map[string]interface{}) func(interface{}) (interface{}, error),
	appliesToRequest bool,
	appliesToResponse bool,
)) {
	f(string(r), r.modifierFactory, true, false)
}

type pluginConfig struct {
	Keys   []string `json:"keys"`
	Values []string `json:"values"`
}

func (r registerer) modifierFactory(
	extra map[string]interface{},
) func(interface{}) (interface{}, error) {
	return func(input interface{}) (interface{}, error) {
		jsonString, _ := json.Marshal(extra[string(r)])
		var config pluginConfig
		json.Unmarshal(jsonString, &config)

		request := input.(proxy.RequestWrapper)

		return requestWrapper{
			params:  request.Params(),
			headers: request.Headers(),
			body:    modifyBody(request.Body(), config),
			method:  request.Method(),
			url:     request.URL(),
			query:   request.Query(),
			path:    request.Path(),
		}, nil
	}
}

func modifyBody(body io.ReadCloser, config pluginConfig) io.ReadCloser {
	var data string

	var bodyBytes []byte
	bodyBytes, _ = ioutil.ReadAll(body)
	bodyString := string(bodyBytes)

	if isJson(bodyString) {
		data, _ = modifyJson(bodyString, config)
	} else {
		data, _ = modifyUrl(bodyString, config)
	}

	return ioutil.NopCloser(bytes.NewBufferString(data))
}

func isJson(s string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(s), &js) == nil
}

func modifyUrl(s string, config pluginConfig) (string, error) {
	data, _ := url.ParseQuery(s)
	for index, key := range config.Keys {
		data.Set(key, config.Values[index])
	}
	return data.Encode(), nil
}

func modifyJson(s string, config pluginConfig) (string, error) {
	var dataMap map[string]string
	json.Unmarshal([]byte(s), &dataMap)

	urlData := url.Values{}
	for key, value := range dataMap {
		urlData.Set(key, value)
	}

	for index, key := range config.Keys {
		urlData.Set(key, config.Values[index])
	}

	data, err := qson.ToJSON(urlData.Encode())

	return string(data), err
}

func (r requestWrapper) Method() string               { return r.method }
func (r requestWrapper) URL() *url.URL                { return r.url }
func (r requestWrapper) Query() url.Values            { return r.query }
func (r requestWrapper) Path() string                 { return r.path }
func (r requestWrapper) Body() io.ReadCloser          { return r.body }
func (r requestWrapper) Params() map[string]string    { return r.params }
func (r requestWrapper) Headers() map[string][]string { return r.headers }

type requestWrapper struct {
	method  string
	url     *url.URL
	query   url.Values
	path    string
	body    io.ReadCloser
	params  map[string]string
	headers map[string][]string
}
