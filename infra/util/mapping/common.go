package mapping

import "github.com/Boyuan-IT-Club/Meowpick-Backend/types/mapping"

// StaticData 存放所有静态映射数据
type StaticData struct {
	Campuses       map[string]string
	Departments    map[string]string
	Categories     map[string]string
	ProposalStatus map[string]string
}

var Data = &StaticData{
	Campuses:       mapping.CampusesMap,
	Departments:    mapping.DepartmentsMap,
	Categories:     mapping.CategoriesMap,
	ProposalStatus: mapping.ProposalStatusMap,
}
