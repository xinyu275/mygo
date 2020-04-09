package Excel

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Opt struct {
	FuncName string
	Keys []string
	Fields []string
	RecordName string    // 如果存在，那Fields失效
	Atoms []string       // 原子字段列表

	leftSuffer string	// fieldStr前面填充
	rightSuffer string  // fieldStr 后面填充
	endSuffer string    // 一行结束填充

	FP *os.File
}

// CreateLine 生成1行1行的记录
// 例如：get(10, 2) -> #goods{"id" = 10, "name" = "物品1"};
// 例如：get(10, 1) -> {10, "物品1"};
// 例如：get(10, 2) -> #goods{"name" = "物品1"};		// 字段可以定制
func (e *Excel) CreateLine(opt *Opt) error {
	// 验证key field 字段是否有效
	err := e.checkKeyFieldValid(opt)
	if err != nil {
		return  err
	}
	opt.leftSuffer = "{"
	if opt.RecordName != "" {
		opt.leftSuffer = "#" + opt.RecordName + "{"
	}
	opt.rightSuffer = "}"
	opt.endSuffer = ";"
	err = e.createLine(opt)
	return  err
}
// createLine
func (e *Excel) createLine(opt *Opt) (err error){
	for _, line := range e.data {
		keyStr 	:= ""
		fieldStr := ""
		//封装keys
		for _, key := range opt.Keys {
			for _, item := range line {
				if key == item.Name {
					keyStr += formatItemValue(opt, item)
				}
			}
		}
		keyStr = strings.Trim(keyStr, ",")
		//如果是0， 那就是包含所有的fields
		for _, item := range line {
			if !InStringSlice(opt.Fields, item.Name) && len(opt.Fields) != 0 {
				continue
			}
			if opt.RecordName == "" {
				if item.DataType == "string" && !InStringSlice(opt.Atoms, item.Name){
					fieldStr += fmt.Sprintf("%q,", item.value)
				}else {
					fieldStr += fmt.Sprintf("%v,", item.value)
				}
			}else {
				if item.DataType == "string" && !InStringSlice(opt.Atoms, item.Name) {
					fieldStr += fmt.Sprintf("%s=%q,", item.Name, item.value)
				}else {
					fieldStr += fmt.Sprintf("%s=%v,", item.Name, item.value)
				}
			}
		}
		fieldStr = strings.Trim(fieldStr, ",")
		str := fmt.Sprintf("%s(%s)->%s%s%s%s\r\n", opt.FuncName, keyStr, opt.leftSuffer, fieldStr, opt.rightSuffer, opt.endSuffer)
		opt.FP.WriteString(str)
	}
	//默认值处理
	if opt.endSuffer != "." {
		defaultStr := formatDefaultString(opt, false)
		opt.FP.WriteString(defaultStr)
	}
	return
}

// CreateIds 生成列表, id 去重
// 例如: get_ids() -> [1, 2, 3].
// 例如: get_ids1(1, 2) -> [1, 2, 3, 4]
func (e *Excel) CreateIds(opt *Opt) error {
	// 验证key field 字段是否有效
	err := e.checkKeyFieldValid(opt)
	if err != nil {
		return  err
	}
	if len(opt.Fields) != 1 {
		return fmt.Errorf("CreateIds fields length != 1 , fields = %v", opt.Fields)
	}
	fieldName := opt.Fields[0]
	//验证field字段对应meta的类型是否是int
	for _, meta := range e.metaList {
		if meta.Name == fieldName && meta.DataType != "int"{
			return  fmt.Errorf("CreateIds field dataType not int, fields=%v, dataType=%s", opt.Fields, meta.DataType)
		}
	}
	return e.createIds(opt)
}
func (e *Excel) createIds(opt *Opt) error {
	m := make(map[string][]int)
	fieldName := opt.Fields[0]
	for _, line := range e.data {
		//封装keys
		keyStr := ""
		fieldId := 0
		for _, key := range opt.Keys {
			for _, item := range line {
				if key == item.Name {
					keyStr += formatItemValue(opt, item)
				}else if fieldName == item.Name {
					fieldId = int(item.value.(int64))
				}
			}
		}
		keyStr = strings.Trim(keyStr, ",")
		sc, _ := m[keyStr]
		for _, item := range line {
			if fieldName == item.Name {
				fieldId = int(item.value.(int64))
				sc = append(sc, fieldId)
				m[keyStr] = sc
			}
		}
	}
	str := ""
	var allKeys []string = make([]string, 0, len(m))
	for kStr, _ := range m {
		allKeys = append(allKeys, kStr)
	}
	sort.Strings(allKeys)
	for _, keyStr := range allKeys {
		sc := m[keyStr]
		sc = filterRepeated(sc)
		scStr := "["
		for _, intV := range sc {
			scStr += strconv.Itoa(intV) + ","
		}
		scStr = strings.Trim(scStr, ",")
		scStr += "]"
		if len(opt.Keys) == 0 {
			str = fmt.Sprintf("%s(%s)->%s.\r\n", opt.FuncName, keyStr, scStr)
		}else {
			str = fmt.Sprintf("%s(%s)->%s;\r\n", opt.FuncName, keyStr, scStr)
		}
		opt.FP.WriteString(str)
	}
	if len(opt.Keys) > 0 {
		//默认值
		defaultStr := formatDefaultString(opt, []int{})
		opt.FP.WriteString(defaultStr)
	}
	return nil
}

// SomeToOne 多对一
// 例如：one_to_one(1) -> "物品A";
// 例如：one_to_one(1, 2) -> "物品A";
func (e *Excel) SomeToOne(opt *Opt) error {
	// 验证key field 字段是否有效
	err := e.checkKeyFieldValid(opt)
	if err != nil {
		return  err
	}
	if len(opt.Fields) != 1 {
		return fmt.Errorf("SomeToOne fields length != 1 , fields = %v", opt.Fields)
	}
	if len(opt.Keys) == 0 {
		return fmt.Errorf("SomeToOne keys length ==0, opt=%#v", *opt)
	}
	opt.endSuffer = ";"
	err = e.createLine(opt)
	return  err
}

// CreateConstantLine 常量行
// 例如：get() -> #goods{"name" = "物品1"}.			// 常量行(就一条记录， 字段可以定制)
func (e *Excel) CreateConstantLine(opt *Opt) error {
	if len(e.data) > 1 {
		return fmt.Errorf("CreateConstantLine error, data length > 1, data=%#v",e.data)
	}
	if opt.RecordName == ""{
		return fmt.Errorf("CreateConstantLine RecordName is empty opt=%#v",*opt)
	}
	opt.leftSuffer = "#" + opt.RecordName + "{"
	opt.rightSuffer = "}"
	opt.endSuffer = "."
	err := e.createLine(opt)
	return  err
}

//CreateMax 最大值
func (e *Excel) CreateMax(opt *Opt) error {
	// 验证key field 字段是否有效
	err := e.checkKeyFieldValid(opt)
	if err != nil {
		return  err
	}
	if len(opt.Fields) != 1 {
		return fmt.Errorf("CreateMax fields length != 1 , fields = %v", opt.Fields)
	}
	fieldName := opt.Fields[0]
	//验证field字段对应meta的类型是否是int
	for _, meta := range e.metaList {
		if meta.Name == fieldName && meta.DataType != "int"{
			return  fmt.Errorf("CreateMax field dataType not int, fields=%v, dataType=%s", opt.Fields, meta.DataType)
		}
	}
	max := 0
	initMax := false
	for _, line := range e.data {
		for _, item := range line {
			if fieldName == item.Name {
				itemValue := int(item.value.(int64))
				if !initMax || itemValue > max {
					initMax = true
					max  = itemValue
				}
			}
		}
	}
	str := fmt.Sprintf("%s()->%d.\r\n", opt.FuncName, max)
	opt.FP.WriteString(str)
	return nil
}

//验证key field字段的有效性
func (e *Excel) checkKeyFieldValid(opt *Opt) error  {
	//判断key field是否都在列表中
	checkKeyFields := make([]string, 0)
	checkKeyFields = append(checkKeyFields, opt.Keys...)
	checkKeyFields = append(checkKeyFields, opt.Fields...)
	exists := false
	for _, kf := range checkKeyFields {
		exists = false
		for _, meta := range e.metaList {
			if meta.Name == kf {
				exists = true
				break
			}
		}
		if !exists {
			return  fmt.Errorf("CreateLine erorr, opt=%#v, field=%s not found", *opt, kf)
		}
	}
	return  nil
}
// 默认处理
func formatDefaultString(opt *Opt, defaultValue interface{}) string {
	dk := ""
	for i:= 0;i<len(opt.Keys);i++ {
		dk += "_,"
	}
	dk = strings.Trim(dk, ",")
	return fmt.Sprintf("%s(%s)->%v.\r\n", opt.FuncName, dk, defaultValue)
}

//格式化IItem
func formatItemValue(opt *Opt, item *EItem) string {
	str := ""
	if item.DataType == "string" && !InStringSlice(opt.Atoms, item.Name) {
		str += fmt.Sprintf("%q,", item.value)
	}else {
		str += fmt.Sprintf("%v,", item.value)
	}
	return str
}

func InStringSlice(sl []string, item string) bool {
	if sl == nil {
		return false
	}
	for _, s := range sl {
		if s == item {
			return true
		}
	}
	return false
}

func filterRepeated(intSlice []int) []int {
	m := make(map[int]int)
	c := make([]int, len(intSlice))
	index := 0
	for _, item := range intSlice {
		if _, ok := m[item];!ok {
			m[item] = 1
			c[index] = item
			index ++
		}
	}
	return c[:index]
}