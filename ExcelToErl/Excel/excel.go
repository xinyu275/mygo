package Excel

import (
	"fmt"
	"github.com/tealeg/xlsx"
	"errors"
	"strconv"
	"strings"
)

type Exceler interface {
	CreateLine(*Opt) error
	CreateConstantLine(*Opt) error
	CreateIds(*Opt) error
	SomeToOne(*Opt) error
	CreateMax(*Opt) error
}
type DataMeta struct {
	Name       string
	DataType   string
	Comment    string
	CreateType string
}

type EItem struct {
	Name string
	DataType string   		// 数据类型 （例如：string int）
	value interface{}
}
type Excel struct {
	sheet *xlsx.Sheet
	metaList []*DataMeta
	data [][]*EItem
	err error
}
func ParseExcel(fileName string) (Exceler, error) {
	e := new(Excel)
	return e.parseExcel(fileName)
}
func (e *Excel) parseExcel(fileName string) (Exceler, error)  {
	xlFile, err := xlsx.OpenFile(fileName)
	if err != nil{
		fmt.Println(err)
		return nil, err
	}
	sheet := xlFile.Sheets[0]
	if len(sheet.Rows) < 4 || len(sheet.Cols) == 0{
		return nil, errors.New("no data")
	}
	e.sheet = sheet
	//解析meta
	e.parseSheetMeta()
	//解析cells
	e.parseSheetRows()

	return e, e.err
}
//解析头部
func (e *Excel) parseSheetMeta() {
	rows:= e.sheet.Rows
	metaList := make([]*DataMeta, len(e.sheet.Cols))
	for i:=0; i<len(e.sheet.Cols); i++{
		meta := &DataMeta{
			Name:      rows[0].Cells[i].String() ,
			DataType:   rows[1].Cells[i].String(),
			Comment:    rows[2].Cells[i].String(),
			CreateType: rows[3].Cells[i].String(),
		}
		//if meta.Name == "" {   // 策划备注行会用到
		//	e.err = errors.New("meta name empty")
		//	return
		//}
		metaList[i] = meta
	}
	e.metaList = metaList
}
//解析
func (e *Excel) parseSheetRows() {
	if e.err != nil {
		return
	}
	rowsNum := len(e.sheet.Rows)
	rows := e.sheet.Rows
	for i:= 4;i<rowsNum;i++{
		row := rows[i]
		//没主键
		if row.Cells[0].String() == "" {
			break
		}
		e.parseSheetRow(row)
	}
}
//单行解析
func (e *Excel) parseSheetRow(row *xlsx.Row) {
	var items []*EItem
	var cellVal string = ""
	for j := 0;j<len(e.metaList);j++ {
		meta := e.metaList[j]
		if !strings.Contains(meta.CreateType, "s") {
			continue
		}
		if j >= len(row.Cells) {
			cellVal = ""
		}else {
			cellVal = row.Cells[j].String()
		}
		item := &EItem{
			Name:     meta.Name,
			DataType: meta.DataType,
			value:    e.parseValue(meta.DataType, cellVal),
		}
		items = append(items, item)
	}
	if len(items) > 0 {
		e.data = append(e.data, items)
	}
}
func(e *Excel) parseValue(dataType string, value string) (v interface{}){
	value = strings.TrimSpace(value)
	switch dataType {
	case "int", "int32", "uint32", "uint64", "int64":
		if value == "" {
			v = 0
			return
		}
		v, e.err = strconv.ParseInt(value, 10, 64)
	case "string", "intKV" :
		v = value
	case "float32", "float64", "float":
		if value == "" {
			v = 0.0
			return
		}
		v, e.err = strconv.ParseFloat(value, 64)
	case "bool":
		if value == "" {
			v = false
			return
		}
		v, e.err = strconv.ParseBool(value)
	case "intarr":
		v = "[" + value + "]"
	default:
		e.err = fmt.Errorf("parseValue dataType=%s undefined", dataType)
	}
	return
}


