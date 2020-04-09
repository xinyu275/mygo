package Excel

import (
	"fmt"
	"testing"
)

func TestExcel(t *testing.T) {
	excel, err := ParseExcel("goods.xlsx")
	if err != nil {
		fmt.Println(err)
		return
	}
	excel.CreateLine(&Opt{FuncName:"get", Keys:[]string{"id"}, RecordName:"goods"})
}
