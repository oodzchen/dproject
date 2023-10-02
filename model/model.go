package model

type OptionItem struct {
	Value any
	Name  string
}

type StringEnum interface {
	Text(upCaseHead bool) string
}

func ConvertEnumToOPtions(values []StringEnum, upCaseHead bool) []*OptionItem {
	var options []*OptionItem

	for _, val := range values {
		options = append(options, &OptionItem{
			Value: val,
			Name:  val.Text(upCaseHead),
		})
		// if v, ok := val.(StringEnum); ok {
		// 	options = append(options, &OptionItem{
		// 		Value: v,
		// 		Name:  v.Text(upCaseHead),
		// 	})
		// } else {
		// 	return nil, fmt.Errorf("convert failed: %v\n", val)
		// }

	}

	return options
}
