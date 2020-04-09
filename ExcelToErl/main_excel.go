package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"mygo/Excel"
	"os"
	"path/filepath"
	"strings"
)

type JsonData struct {
	ErlName string
	RecordFiles []string
	ExcelFuncs []ExcelFunc
}
type ExcelFunc struct {
	ExcelName string
	FuncName string
	Call string          // 调用函数 "CreateLine", "CreateIds"
	Keys []string
	Fields []string
	RecordName string    // 如果存在，那Fields失效
	Atoms []string       // 原子字段列表
}

var (
		JsonName = "data.json"
		DefaultExcelDir = "./"
		DefaultErlDir = "./"
		ExcelDir = ""			// excel目录路径
		ErlDir = ""			// excel文件保存路径
	)
func main()  {
	flag.StringVar(&ExcelDir, "excel_dir", DefaultExcelDir, "excel dir")
	flag.StringVar(&ErlDir, "erl_dir", DefaultErlDir, "erl dir")
	flag.Parse()

	var jd []JsonData
	err := toJsonData(&jd)
	if err != nil{
		fmt.Println(err)
		return
	}
	for _, ErlJson := range jd {
		if ErlJson.ErlName != "base_test.erl"{
			err := parseToErl(ErlJson)
			if err != nil{
				fmt.Println(err)
				return
			}
		}
	}
	fmt.Println("success")
}
//转化为json数据
func toJsonData(jd *[]JsonData) (err error) {
	data, err := ioutil.ReadFile(JsonName)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &jd)
	return
}

//将json数据解析成一个erl文件
func parseToErl(data JsonData) (err error){
	//1.整理函数，那相同excel的方一起
	var mFuncs map[string][]ExcelFunc = make(map[string][]ExcelFunc)
	var mExcels []string = make([]string, 0)
	for _, item := range data.ExcelFuncs {
		efs, _ := mFuncs[item.ExcelName]
		efs = append(efs, item)
		mFuncs[item.ExcelName] = efs
		if !Excel.InStringSlice(mExcels, item.ExcelName) {
			mExcels = append(mExcels, item.ExcelName)
		}
	}
	//2.生成头部信息
	f, err := os.OpenFile(ErlDir + "/" + data.ErlName,  os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return
	}
	defer f.Close()

	head := getHead(data)
	f.WriteString(head)

	//3.对excel解析
	//for excelName, excelFuncs := range mFuncs {
	for _, excelName := range mExcels{
		excelFuncs := mFuncs[excelName]
		fmt.Println("parse " + excelName+"...")
		excel, e1 := Excel.ParseExcel(ExcelDir+"/"+excelName)
		if e1 != nil {
			err = e1
			return
		}
		for _, item := range excelFuncs {
			opt := Excel.Opt{
				FuncName:     item.FuncName,
				Keys:         item.Keys,
				Fields:       item.Fields,
				RecordName:   item.RecordName,
				Atoms:        item.Atoms,
				FP:f,
			}
			var e2 error
			switch item.Call{
			case "CreateLine":
				e2 = excel.CreateLine(&opt)
			case "CreateIds":
				e2 = excel.CreateIds(&opt)
			case "SomeToOne":
				e2 = excel.SomeToOne(&opt)
			case "CreateConstantLine":
				e2 = excel.CreateConstantLine(&opt)
			case "CreateMax":
				e2 = excel.CreateMax(&opt)
			}
			if e2 != nil {
				err = e2
				break
			}
			f.WriteString("\r\n")
		}
	}
	//3.根据call调用不同的函数
	return
}

func getHead(data JsonData) string {
	mod := strings.TrimSuffix(filepath.Base(data.ErlName), ".erl")
	inc := ""
	for _, recordFile := range data.RecordFiles {
		inc += fmt.Sprintf("-include(%q).\r\n", recordFile)
	}
	head := fmt.Sprintf(
		`%%automatic generate, plz do not change, thx!
-module(%s).
%s
-compile(export_all).

`, mod, inc)
	return head
}