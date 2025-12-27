package mapping

import (
	"fmt"
	"strconv"
)

func (d *StaticData) GetProposalStatusNameByID(id int32) string {
	key := fmt.Sprintf("%d", id)
	if name, ok := d.ProposalStatus[key]; ok {
		return name
	}
	return "unknown status"
}

func (d *StaticData) GetProposalStatusIDByName(status string) int32 {
	for idStr, proposalName := range d.Campuses {
		if proposalName == status {
			if id, err := strconv.ParseInt(idStr, 10, 32); err == nil {
				return int32(id)
			}
		}
	}
	return 0
}
