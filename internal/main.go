package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"reflect"
	"strings"
)

var res string

func first2Upper(s string) string {
	sarr := strings.Split(s, "_")
	if len(sarr) > 1 {
		res := ""
		for _, sub := range sarr {
			res += first2Upper(sub)
		}
		return res
	}
	runes := []rune(s)
	return strings.ToUpper(string(runes[0])) + string(runes[1:])
}

func hump2Underline(s string) string {
	data := make([]byte, 0, len(s)*2)
	flag := false
	num := len(s)
	for i := 0; i < num; i++ {
		d := s[i]
		if i > 0 && d >= 'A' && d <= 'Z' && flag {
			data = append(data, '_')
		}
		if d != '_' {
			flag = true
		}
		data = append(data, d)
	}
	return strings.ToLower(string(data[:]))
}

func generateSlice(buffer *bytes.Buffer, data interface{}, structName string) {
	vslice := data.([]interface{})
	if len(vslice) > 0 {
		switch reflect.ValueOf(vslice[0]).Kind() {
		case reflect.Map, reflect.Slice, reflect.Array:

			sname := first2Upper(structName)
			GenerateStruct(sname, vslice[0])
			buffer.WriteString("[]" + sname)
		case reflect.Float64:
			vm := vslice[0].(float64)
			if vm == float64(int(vm)) {
				buffer.WriteString("[]int")
			} else {
				buffer.WriteString("[]float64")
			}
		default:
			buffer.WriteString("[]" + reflect.ValueOf(vslice[0]).Kind().String())
		}
	}
}

func GenerateStruct(structName string, data interface{}) {
	structName = first2Upper(structName)
	v := reflect.ValueOf(data)
	var buffer bytes.Buffer
	buffer.WriteString("type ")
	buffer.WriteString(structName)
	buffer.WriteString(" struct {\n")
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		generateSlice(&buffer, data, structName)
	case reflect.Map:
		for k, m := range data.(map[string]interface{}) {
			buffer.WriteString("\t")
			buffer.WriteString(first2Upper(k))
			buffer.WriteString("\t")
			switch reflect.ValueOf(m).Kind() {
			case reflect.Array, reflect.Slice:
				generateSlice(&buffer, m, k)
			case reflect.Map:
				vtype := first2Upper(k)
				GenerateStruct(vtype, m)
				buffer.WriteString(vtype)
			case reflect.Float64:
				vm := m.(float64)
				if vm == float64(int(vm)) {
					buffer.WriteString("int")
				} else {
					buffer.WriteString("float64")
				}
			default:
				if m == nil {
					buffer.WriteString("interface{}")
				} else {
					buffer.WriteString(reflect.ValueOf(m).Kind().String())
				}
			}
			buffer.WriteString("\t`json:\"")
			// buffer.WriteString(hump2Underline(k))
			buffer.WriteString(k)
			buffer.WriteString("\" schema:\"")
			// buffer.WriteString(hump2Underline(k))
			buffer.WriteString(k)
			buffer.WriteString("\"`")
			buffer.WriteString("\n")
		}

	}
	buffer.WriteString("}")
	buffer.WriteString("\n")
	res += buffer.String() + "\n"
}

func schemaGenerate(name, ss string) {
	values, _ := url.ParseQuery(ss)
	mid := make(map[string]interface{}, len(values))

	for k, v := range values {
		var data interface{}
		err := json.Unmarshal([]byte(v[0]), &data)
		if err != nil {
			mid[k] = v[0]
		} else {
			mid[k] = data
		}

	}

	GenerateStruct(name, mid)
}

func jsonGenerate(name, data string) {
	var rs interface{}
	err := json.Unmarshal([]byte(data), &rs)
	if err != nil {
		fmt.Println(err)
		return
	}
	GenerateStruct(name, rs)

}

func main() {
	name := flag.String("n", "", "结构体名称")
	src := flag.String("s", "", "原json文件地址")
	out := flag.String("o", "", "输出文件地址, 如果为空打印出来")
	jData := flag.String("j", "", "输入json数据")
	urlStr := flag.String("u", "", "url 解析")
	pkg := flag.String("p", "default", "生成文件的包名")
	flag.Parse()

	if *name == "" {
		*name = "Default"
		if *src != "" {
			*name = strings.Split(*src, ".")[0]
		} else if *urlStr != "" {
			*name = "Request"
		}
	}

	if *urlStr != "" && (*src != "" || *jData != "") {
		fmt.Println("str url json only one")
		return
	}

	if *src != "" && *jData != "" {
		fmt.Println("str url json only one")
		return
	}

	if *out != "" && !strings.Contains(*out, ".go") {
		*out = *out + ".go"
	}

	if *urlStr != "" {
		ss := *urlStr
		if strings.HasPrefix(*urlStr, "http") {
			ss = strings.Split(*urlStr, "?")[1]
		}
		schemaGenerate(*name, ss)
	}

	if *src != "" {
		content, err := ioutil.ReadFile(*src)
		if err != nil {
			fmt.Println("open file faild, err: ", err)
			return
		}
		jsonGenerate(*name, string(content))
	}

	if *jData != "" {
		jsonGenerate(*name, *jData)
	}

	if *out == "" {
		fmt.Println(res)
	} else {
		file, err := os.OpenFile(*out, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			fmt.Println("create file error", err)
			return
		}
		res = "package " + strings.ToLower(*pkg) + "\n" + res

		file.Write([]byte(res))
		defer file.Close()

		cmd := exec.Command("gofmt", "-w", "./")
		cmd.Run()
	}

}
