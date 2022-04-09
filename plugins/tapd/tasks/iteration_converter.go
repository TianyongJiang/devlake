package tasks

import (
	"fmt"
	"github.com/merico-dev/lake/models/domainlayer"
	"github.com/merico-dev/lake/models/domainlayer/didgen"
	"github.com/merico-dev/lake/models/domainlayer/ticket"
	"github.com/merico-dev/lake/plugins/core"
	"github.com/merico-dev/lake/plugins/helper"
	"github.com/merico-dev/lake/plugins/tapd/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"reflect"
	"strings"
)

func ConvertIteration(taskCtx core.SubTaskContext) error {
	data := taskCtx.GetData().(*TapdTaskData)
	logger := taskCtx.GetLogger()
	db := taskCtx.GetDb()
	logger.Info("collect board:%d", data.Options.WorkspaceId)
	iterIdGen := didgen.NewDomainIdGenerator(&models.TapdIteration{})
	cursor, err := db.Model(&models.TapdIteration{}).Where("source_id = ? AND workspace_id = ?", data.Source.ID, data.Options.WorkspaceId).Rows()
	if err != nil {
		return err
	}
	defer cursor.Close()
	converter, err := helper.NewDataConverter(helper.DataConverterArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: TapdApiParams{
				SourceId: data.Source.ID,
				//CompanyId:   data.Source.CompanyId,
				WorkspaceId: data.Options.WorkspaceId,
			},
			Table: RAW_ITERATION_TABLE,
		},
		InputRowType: reflect.TypeOf(models.TapdIteration{}),
		Input:        cursor,
		Convert: func(inputRow interface{}) ([]interface{}, error) {
			iter := inputRow.(*models.TapdIteration)
			domainIter := &ticket.Sprint{
				DomainEntity:  domainlayer.DomainEntity{Id: iterIdGen.Generate(data.Source.ID, iter.ID)},
				Url:           fmt.Sprintf("https://www.tapd.cn/%d/prong/iterations/view/%d", iter.WorkspaceId, iter.ID),
				Status:        strings.ToUpper(iter.Status),
				Name:          iter.Name,
				StartedDate:   iter.Startdate,
				EndedDate:     iter.Enddate,
				OriginBoardID: WorkspaceIdGen.Generate(iter.SourceId, iter.WorkspaceId),
				CompletedDate: iter.Completed,
			}
			results := make([]interface{}, 0)
			results = append(results, domainIter)
			var sprintIssues []models.TapdIterationIssue
			err = db.Find(&sprintIssues, "source_id = ? AND iteration_id = ?", data.Source.ID, iter.ID).Error
			if err != nil && err != gorm.ErrRecordNotFound {
				return nil, err
			}
			domainSprintIssues := make([]ticket.SprintIssue, 0, len(sprintIssues))
			for _, si := range sprintIssues {
				dsi := ticket.SprintIssue{
					SprintId:  domainIter.Id,
					IssueId:   IssueIdGen.Generate(data.Source.ID, si.IssueId),
					AddedDate: si.IssueCreatedDate,
				}
				if dsi.AddedDate != nil {
					dsi.AddedStage = getStage(*dsi.AddedDate, domainIter.StartedDate, domainIter.CompletedDate)
				}
				if si.ResolutionDate != nil {
					dsi.ResolvedStage = getStage(*si.ResolutionDate, domainIter.StartedDate, domainIter.CompletedDate)
				}
				domainSprintIssues = append(domainSprintIssues, dsi)
			}
			if len(domainSprintIssues) > 0 {
				err = db.Clauses(clause.OnConflict{DoUpdates: clause.AssignmentColumns([]string{"resolved_stage"})}).Create(domainSprintIssues).Error
				if err != nil {
					return nil, err
				}

			}
			boardSprint := &ticket.BoardSprint{
				BoardId:  domainIter.OriginBoardID,
				SprintId: domainIter.Id,
			}
			results = append(results, boardSprint)
			return results, nil
		},
	})
	if err != nil {
		return err
	}

	return converter.Execute()
}

var ConvertIterationMeta = core.SubTaskMeta{
	Name:             "convertIteration",
	EntryPoint:       ConvertIteration,
	EnabledByDefault: true,
	Description:      "convert Tapd iteration",
}
